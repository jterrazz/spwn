use std::io::{Cursor, Read};
use std::os::unix::net::{UnixListener, UnixStream};
use std::path::PathBuf;
use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
use std::sync::mpsc::{self, Receiver, Sender};
use std::sync::Arc;
use std::time::{Duration, Instant};

use serde_json::{json, Value};

use super::{
    connect_within, pump, refusal, Exchanges, Limits, Message, Sink, BRIDGED, LIMITS, NOT_BRIDGED,
    NOT_OPENED,
};

const SOON: Duration = Duration::from_secs(2);

struct Heard {
    sent: Sender<Message>,
    alive: Arc<AtomicBool>,
}

impl Sink for Heard {
    fn send(&self, message: Message) -> bool {
        self.alive.load(Ordering::SeqCst) && self.sent.send(message).is_ok()
    }
}

fn heard() -> (Heard, Receiver<Message>, Arc<AtomicBool>) {
    let (sent, received) = mpsc::channel();
    let alive = Arc::new(AtomicBool::new(true));
    (
        Heard {
            sent,
            alive: Arc::clone(&alive),
        },
        received,
        alive,
    )
}

fn socket_path() -> PathBuf {
    static NEXT: AtomicUsize = AtomicUsize::new(0);
    let n = NEXT.fetch_add(1, Ordering::SeqCst);
    std::env::temp_dir().join(format!("window-bridge-{}-{n}.sock", std::process::id()))
}

fn listening() -> (UnixListener, PathBuf) {
    let path = socket_path();
    let _ = std::fs::remove_file(&path);
    let listener = UnixListener::bind(&path).expect("a test socket binds");
    (listener, path)
}

fn call() -> Value {
    json!({ "v": 2, "id": "7", "call": "engine.read", "in": {} })
}

fn watch() -> Value {
    json!({ "v": 2, "id": "8", "watch": "engine.watch", "in": {} })
}

fn error_of(message: &Message) -> Option<(String, String, String)> {
    let Message::Frame(frame) = message else {
        return None;
    };
    let error = frame.get("error")?;
    let text = |key: &str| {
        error
            .get(key)
            .and_then(Value::as_str)
            .unwrap_or_default()
            .to_owned()
    };
    Some((
        frame.get("id")?.as_str()?.to_owned(),
        text("type"),
        text("code"),
    ))
}

fn drained(received: &Receiver<Message>) -> Vec<Message> {
    received.try_iter().collect()
}

#[test]
fn a_request_of_the_envelope_passes_and_any_other_is_refused() {
    // Given - a call, a watch, and frames the engine would refuse
    let refused = [
        json!([1, 2]),
        json!({ "v": 1, "id": "1", "call": "engine.read" }),
        json!({ "v": 2, "id": 1, "call": "engine.read" }),
        json!({ "v": 2, "id": "1" }),
        json!({ "v": 2, "id": "1", "call": "engine.read", "watch": "engine.watch" }),
        json!({ "v": 2, "id": "1", "call": "" }),
        json!({ "v": 2, "id": "1", "call": "engine.read", "out": {} }),
        json!({ "v": 2, "id": "1", "call": "engine.read", "target": { "name": 3 } }),
        json!({ "v": 2, "id": "1", "call": "engine.read", "target": { "host": "x" } }),
        json!({ "v": 2, "id": "1", "call": "engine.read", "in": "x".repeat(2048) }),
    ];

    // When - each is checked against a 1 kB limit
    let passed = [
        call(),
        watch(),
        json!({ "v": 2, "id": "2", "call": "machines.stop", "target": { "name": "a", "on": "b" } }),
    ]
    .iter()
    .map(|request| refusal(request, 1024))
    .collect::<Vec<_>>();
    let failed = refused
        .iter()
        .filter(|request| refusal(request, 1024).is_some())
        .count();

    // Then - the envelopes pass and every other frame is refused
    assert_eq!(passed, vec![None, None, None]);
    assert_eq!(failed, refused.len());
}

#[test]
fn a_line_that_is_no_json_ends_the_exchange_with_the_bad_frame_error() {
    // Given - an engine that answers a progress, then a line that is no JSON, then an out
    let (sink, received, _) = heard();
    let lines = "{\"v\":2,\"id\":\"7\",\"progress\":{\"task\":\"t\"}}\nnot json\n{\"v\":2,\"id\":\"7\",\"out\":{}}\n";

    // When - the exchange is pumped
    let reason = pump(Cursor::new(lines), &call(), &sink, 1024);

    // Then - the progress came, then the engine's own error shape for a bad frame, and no out
    let messages = drained(&received);
    assert_eq!(reason, "answered");
    assert_eq!(messages.len(), 2);
    assert_eq!(
        error_of(&messages[1]),
        Some(("7".into(), "failed".into(), "call.bad-frame".into()))
    );
}

#[test]
fn a_line_longer_than_the_limit_ends_the_exchange_cleanly() {
    // Given - an answer line of 4 kB against a 1 kB limit, then a valid out
    let (sink, received, _) = heard();
    let long = format!(
        "{{\"v\":2,\"id\":\"7\",\"out\":\"{}\"}}\n{{\"v\":2,\"id\":\"7\",\"out\":{{}}}}\n",
        "x".repeat(4096)
    );

    // When - the exchange is pumped
    let reason = pump(Cursor::new(long), &call(), &sink, 1024);

    // Then - one error frame says so, and nothing past it is read
    let messages = drained(&received);
    assert_eq!(reason, "answered");
    assert_eq!(messages.len(), 1);
    assert_eq!(
        error_of(&messages[0]),
        Some(("7".into(), "failed".into(), "call.bad-frame".into()))
    );
}

#[test]
fn a_peer_that_closes_mid_answer_ends_the_exchange_unanswered() {
    // Given - an engine that writes half an answer and closes
    let (sink, received, _) = heard();

    // When - the exchange is pumped
    let reason = pump(
        Cursor::new("{\"v\":2,\"id\":\"7\",\"ou"),
        &call(),
        &sink,
        1024,
    );

    // Then - no frame is forwarded, and the end says the connection closed
    assert_eq!(reason, "the engine closed the connection");
    assert!(drained(&received).is_empty());
}

#[test]
fn a_sink_that_stops_listening_ends_the_reader_at_its_next_frame() {
    // Given - a watch whose sink is dead, over three outs
    let (sink, received, alive) = heard();
    alive.store(false, Ordering::SeqCst);
    let outs = "{\"v\":2,\"id\":\"8\",\"out\":{}}\n".repeat(3);

    // When - the exchange is pumped
    let reason = pump(Cursor::new(outs), &watch(), &sink, 1024);

    // Then - the reader stopped on the first frame it could not hand over
    assert_eq!(reason, "the window stopped listening");
    assert!(drained(&received).is_empty());
}

#[test]
fn a_quiet_watch_ends_its_reader_and_socket_when_its_webview_goes() {
    // Given - a watch opened by the webview main on an engine that never answers
    let (listener, path) = listening();
    let exchanges = Arc::new(Exchanges::default());
    let (sink, received, _) = heard();
    let handle = exchanges
        .open(&path, "main", watch(), sink, LIMITS)
        .expect("the watch opens");
    let (mut engine, _) = listener.accept().expect("the engine accepts");
    let mut asked = vec![0; 1];
    engine.read_exact(&mut asked).expect("the request arrives");

    // When - the webview reloads, closing what it opened
    let closed = exchanges.close_owner("main");

    // Then - the reader ends at once, and the engine sees its socket closed
    assert_eq!((handle, closed), (1, 1));
    assert_eq!(
        received.recv_timeout(SOON).ok(),
        Some(Message::End("the engine closed the connection".into()))
    );
    engine
        .set_read_timeout(Some(SOON))
        .expect("a read timeout sets");
    let mut rest = Vec::new();
    assert!(engine.read_to_end(&mut rest).is_ok());
    assert_eq!(exchanges.count(), 0);
    let _ = std::fs::remove_file(path);
}

#[test]
fn a_cancel_reaches_the_engine_for_a_request_this_window_opened_and_no_other() {
    // Given - a watch of id 8 opened by the webview main, held by its engine
    let (listener, path) = listening();
    let exchanges = Arc::new(Exchanges::default());
    let (sink, _received, _) = heard();
    let handle = exchanges
        .open(&path, "main", watch(), sink, LIMITS)
        .expect("the watch opens");
    let (mut engine, _) = listener.accept().expect("the engine accepts");
    let cancel = |id: &str| json!({ "v": 2, "id": id, "cancel": true });

    // When - main cancels id 8, then an unknown id, another webview id 8, and a frame of another shape
    let passed = exchanges.cancel(handle, "main", &cancel("8"));
    let refused = [
        exchanges.cancel(handle, "main", &cancel("9")),
        exchanges.cancel(handle, "other", &cancel("8")),
        exchanges.cancel(handle + 1, "main", &cancel("8")),
        exchanges.cancel(
            handle,
            "main",
            &json!({ "v": 2, "id": "8", "cancel": true, "in": {} }),
        ),
    ];

    // Then - the engine reads the watch then its cancel, and each other one is refused unsent
    assert_eq!(passed, Ok(()));
    let codes: Vec<&str> = refused
        .iter()
        .map(|one| one.as_ref().map_or_else(|refusal| refusal.code, |_| "sent"))
        .collect();
    assert_eq!(
        codes,
        [NOT_OPENED, NOT_OPENED, NOT_OPENED, "call.bad-frame"]
    );
    exchanges.close(handle);
    engine
        .set_read_timeout(Some(SOON))
        .expect("a read timeout sets");
    let mut lines = String::new();
    engine
        .read_to_string(&mut lines)
        .expect("the engine reads to the end");
    let read: Vec<Value> = lines
        .lines()
        .map(|line| serde_json::from_str(line).expect("each line is JSON"))
        .collect();
    assert_eq!(read, vec![watch(), cancel("8")]);
    let _ = std::fs::remove_file(path);
}

#[test]
fn a_call_answered_is_forwarded_frame_by_frame_then_ended() {
    // Given - an engine that answers a progress then the out
    let (listener, path) = listening();
    let exchanges = Arc::new(Exchanges::default());
    let (sink, received, _) = heard();
    exchanges
        .open(&path, "main", call(), sink, LIMITS)
        .expect("the call opens");
    let (mut engine, _) = listener.accept().expect("the engine accepts");

    // When - it writes both lines
    std::io::Write::write_all(
        &mut engine,
        b"{\"v\":2,\"id\":\"7\",\"progress\":{\"task\":\"t\"}}\n{\"v\":2,\"id\":\"7\",\"out\":{\"State\":\"running\"}}\n",
    )
    .expect("the engine writes");

    // Then - the window hears both frames, then the end, and nothing stays open
    let heard: Vec<Message> = (0..3)
        .filter_map(|_| received.recv_timeout(SOON).ok())
        .collect();
    assert_eq!(heard.len(), 3);
    assert_eq!(heard[2], Message::End("answered".into()));
    assert_eq!(exchanges.count(), 0);
    let _ = std::fs::remove_file(path);
}

#[test]
fn a_refused_request_is_answered_as_the_engine_would_and_never_sent() {
    // Given - no engine at all, and a frame that is not a request
    let exchanges = Arc::new(Exchanges::default());
    let (sink, received, _) = heard();

    // When - the window opens it
    let handle = exchanges.open(
        &socket_path(),
        "main",
        json!({ "v": 2, "id": "9", "hello": {} }),
        sink,
        LIMITS,
    );

    // Then - the refusal is the engine's shape, under the request's id, and nothing was connected
    assert_eq!(handle, Ok(0));
    let messages = drained(&received);
    assert_eq!(
        error_of(&messages[0]),
        Some(("9".into(), "refused".into(), "call.bad-frame".into()))
    );
    assert_eq!(messages[1], Message::End("answered".into()));
}

#[test]
fn a_call_the_window_does_not_send_is_refused_by_name_and_never_sent() {
    // Given - an engine listening, and a well-formed request of engine.guest-exec, which no page sends
    let (listener, path) = listening();
    listener
        .set_nonblocking(true)
        .expect("the listener turns non-blocking");
    let exchanges = Arc::new(Exchanges::default());
    let (sink, received, _) = heard();
    let exec = json!({ "v": 2, "id": "5", "call": "engine.guest-exec", "in": {} });

    // When - the window opens it
    let handle = exchanges.open(&path, "main", exec, sink, LIMITS);

    // Then - it is refused under its id with the bridge's own code, and the engine was never reached
    assert_eq!(handle, Ok(0));
    let messages = drained(&received);
    assert_eq!(
        error_of(&messages[0]),
        Some(("5".into(), "refused".into(), NOT_BRIDGED.into()))
    );
    assert_eq!(messages[1], Message::End("answered".into()));
    assert!(listener.accept().is_err());
    let _ = std::fs::remove_file(path);
}

#[test]
fn every_call_of_the_bridged_list_passes_and_no_other_name() {
    // Given - the list read from src/engine/bridged.ts at build, and names the window never sends
    let unsent = ["engine.guest-exec", "engine.stop", "x.y"];

    // When - a call of each is checked
    let refused = |name: &str| refusal(&json!({ "v": 2, "id": "1", "call": name, "in": {} }), 1024);

    // Then - each bridged name passes, each other is refused as not bridged
    assert!(BRIDGED.len() > 30);
    assert!(BRIDGED.iter().all(|name| refused(name).is_none()));
    assert!(unsent
        .iter()
        .all(|name| refused(name).is_some_and(|refusal| refusal.code == NOT_BRIDGED)));
}

fn full_backlog() -> (UnixListener, PathBuf, Vec<UnixStream>) {
    let (listener, path) = listening();
    // hazard: std listens with the system's backlog; a backlog of 0 lets one connection wait, so the next blocks.
    // SAFETY: listen on the listener's own open descriptor.
    let listened = unsafe { libc::listen(std::os::fd::AsRawFd::as_raw_fd(&listener), 0) };
    assert_eq!(listened, 0);
    let mut waiting = Vec::new();
    while let Ok(stream) = connect_within(&path, Duration::from_millis(200)) {
        waiting.push(stream);
        assert!(waiting.len() < 64, "the backlog never filled");
    }
    (listener, path, waiting)
}

#[test]
fn an_engine_that_accepts_nothing_never_holds_the_open_past_its_limit() {
    // Given - an engine whose backlog is full
    let (_listener, path, _waiting) = full_backlog();
    let exchanges = Arc::new(Exchanges::default());
    let (sink, _received, _) = heard();
    let limits = Limits {
        connect: Duration::from_millis(300),
        ..LIMITS
    };

    // When - a call is opened on it
    let started = Instant::now();
    let opened = exchanges.open(&path, "main", call(), sink, limits);

    // Then - it fails as no engine answering, within its limit
    assert!(started.elapsed() < Duration::from_secs(1));
    assert!(opened.is_err_and(|error| error.starts_with("no engine answers at ")));
    let _ = std::fs::remove_file(path);
}

#[test]
fn an_engine_that_reads_nothing_never_holds_the_write_past_its_limit() {
    // Given - an engine that accepts and never reads, and a request larger than a socket's buffer
    let (listener, path) = listening();
    let exchanges = Arc::new(Exchanges::default());
    let (sink, _received, _) = heard();
    let large =
        json!({ "v": 2, "id": "7", "call": "engine.read", "in": { "x": "x".repeat(900_000) } });
    let limits = Limits {
        write: Duration::from_millis(300),
        ..LIMITS
    };

    // When - the call is opened on it
    let started = Instant::now();
    let opened = exchanges.open(&path, "main", large, sink, limits);

    // Then - the write gives up within its limit, as no engine answering
    assert!(started.elapsed() < Duration::from_secs(2));
    assert!(opened.is_err_and(|error| error.starts_with("no engine answers at ")));
    drop(listener);
    let _ = std::fs::remove_file(path);
}

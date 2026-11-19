use std::io::{BufRead, BufReader, Write};
use std::os::unix::net::UnixListener;
use std::path::PathBuf;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Mutex;
use std::thread;
use std::time::Duration;

use serde_json::{json, Value};

use super::{changed, hello, next, Link, FIRST_BACKOFF, MAX_BACKOFF, RECHECK};

fn socket_path() -> PathBuf {
    static NEXT: AtomicUsize = AtomicUsize::new(0);
    let n = NEXT.fetch_add(1, Ordering::SeqCst);
    std::env::temp_dir().join(format!("window-engine-{}-{n}.sock", std::process::id()))
}

// value: an engine that reads one request line, hands it back on the channel, and writes `lines` in answer.
fn engine(lines: &'static str) -> (PathBuf, thread::JoinHandle<Value>) {
    let path = socket_path();
    let _ = std::fs::remove_file(&path);
    let listener = UnixListener::bind(&path).expect("a test socket binds");
    let serving = thread::spawn(move || {
        let (mut stream, _) = listener.accept().expect("the window connects");
        let mut request = String::new();
        BufReader::new(stream.try_clone().expect("the stream clones"))
            .read_line(&mut request)
            .expect("the request arrives");
        let _ = stream.write_all(lines.as_bytes());
        serde_json::from_str(&request).expect("the request is JSON")
    });
    (path, serving)
}

#[test]
fn a_hello_is_one_v2_call_and_returns_the_first_out_after_any_progress() {
    // Given - an engine that answers a progress, then its hello
    let (path, serving) = engine(
        "{\"v\":2,\"id\":\"hello\",\"progress\":{\"task\":\"t\"}}\n{\"v\":2,\"id\":\"hello\",\"out\":{\"pid\":7}}\n",
    );

    // When - the window says hello
    let answer = hello(&path);

    // Then - the request was engine.hello, and the answer is the out frame
    let request = serving.join().expect("the engine served");
    assert_eq!(
        request,
        json!({ "v": 2, "id": "hello", "call": "engine.hello" })
    );
    assert_eq!(
        answer.ok().and_then(|out| out.get("out").cloned()),
        Some(json!({ "pid": 7 }))
    );
    let _ = std::fs::remove_file(path);
}

#[test]
fn a_hello_refused_is_still_an_answer_and_one_unanswered_is_an_error() {
    // Given - an engine that refuses the hello, one that closes before answering, and no engine at all
    let (refusing, first) =
        engine("{\"v\":2,\"id\":\"hello\",\"error\":{\"message\":\"too old\"}}\n");
    let (closing, second) = engine("");

    // When - the window says hello to each
    let refused = hello(&refusing);
    let closed = hello(&closing);
    let absent = hello(&socket_path());

    // Then - the refusal is the engine's answer, the two others are reasons
    first.join().expect("the first engine served");
    second.join().expect("the second engine served");
    assert!(refused.is_ok_and(|answer| answer.get("error").is_some()));
    assert_eq!(
        closed,
        Err(String::from(
            "the engine closed the connection before it answered"
        ))
    );
    assert!(absent.is_err());
    let _ = std::fs::remove_file(refusing);
    let _ = std::fs::remove_file(closing);
}

#[test]
fn an_engine_that_goes_is_retried_after_a_doubling_backoff_and_its_return_resets_it() {
    // Given - the backoff at its start
    let mut backoff = FIRST_BACKOFF;
    let socket = || String::from("/run/engine.sock");

    // When - six hellos fail, then one answers, then one fails again
    let waits: Vec<u64> = (0..6)
        .map(|_| {
            next(Err(String::from("refused")), socket(), &mut backoff)
                .1
                .as_secs()
        })
        .collect();
    let (open, recheck) = next(Ok(json!({ "out": {} })), socket(), &mut backoff);
    let (absent, wait) = next(Err(String::from("refused")), socket(), &mut backoff);

    // Then - the waits double up to the cap, an answer is open and checked again later, and the backoff restarted
    assert_eq!(waits, vec![1, 2, 4, 8, 16, MAX_BACKOFF]);
    assert!(matches!(open, Link::Open { .. }));
    assert_eq!(recheck, RECHECK);
    assert_eq!(
        absent,
        Link::Absent {
            socket: socket(),
            reason: String::from("refused"),
            retry_in: 1000
        }
    );
    assert_eq!(wait, Duration::from_secs(1));
}

#[test]
fn a_link_is_emitted_when_it_changes_and_never_when_it_stays() {
    // Given - the link shown as connecting
    let shown = Mutex::new(Link::Connecting {
        socket: String::from("/run/engine.sock"),
    });
    let open = Link::Open {
        socket: String::from("/run/engine.sock"),
        answer: json!({ "out": { "pid": 7 } }),
    };

    // When - the same open link is heard twice
    let first = changed(&shown, &open);
    let second = changed(&shown, &open);

    // Then - the first is a change and is kept, the second is none
    assert!(first);
    assert!(!second);
    assert!(shown.lock().is_ok_and(|link| *link == open));
}

use std::io::{BufRead, BufReader, Write};
use std::os::unix::net::UnixListener;
use std::path::PathBuf;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, Instant};

use serde_json::{json, Value};

use super::{
    changed, hello, kept, told, Bridge, Failed, Link, Watch, ABSENT_AFTER, FIRST_BACKOFF,
    MAX_BACKOFF, RETRY,
};

fn refused() -> Failed {
    Failed {
        step: "connect",
        said: String::from("refused"),
    }
}

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
        Err(Failed {
            step: "read",
            said: String::from("the engine closed the connection before it answered"),
        })
    );
    assert!(absent.is_err_and(|failed| failed.step == "connect"));
    let _ = std::fs::remove_file(refusing);
    let _ = std::fs::remove_file(closing);
}

const SOCKET: &str = "/run/engine.sock";

fn open() -> Link {
    Link::Open {
        socket: String::from(SOCKET),
        answer: json!({ "out": {} }),
    }
}

// value: the supervisor's loop with a clock of its own: each hello is heard, the link it gives shown, and the wait it
// value: asks for added to the clock, so the instant each link is shown is read with no sleep; a rest with no end is
// value: cut at once.
fn played(
    watch: &mut Watch,
    mut shown: Link,
    heard: Vec<Result<Value, Failed>>,
) -> Vec<(u64, Link)> {
    let mut clock = Duration::ZERO;
    heard
        .into_iter()
        .map(|heard| {
            let (link, wait) = watch.next(heard, String::from(SOCKET), &shown);
            let at = clock.as_secs();
            clock += wait.unwrap_or(Duration::ZERO);
            shown = link.clone();
            (at, link)
        })
        .collect()
}

#[test]
fn a_single_failed_hello_keeps_the_link_and_the_third_in_a_row_declares_it_absent_10_s_after_the_first(
) {
    // Given - a link open, and an engine that refuses from now on
    let mut watch = Watch::default();

    // When - three hellos fail at once, one after another
    let heard = played(
        &mut watch,
        open(),
        vec![Err(refused()), Err(refused()), Err(refused())],
    );

    // Then - the first two leave the link open, 5 s apart, and the third, at 10 s, declares it absent
    assert_eq!(
        heard,
        vec![
            (0, open()),
            (RETRY.as_secs(), open()),
            (
                2 * RETRY.as_secs(),
                Link::Absent {
                    socket: String::from(SOCKET),
                    reason: String::from("refused"),
                    retry_in: FIRST_BACKOFF * 1000,
                }
            ),
        ]
    );
    assert_eq!(ABSENT_AFTER, 3);
    assert_eq!(2 * RETRY, Duration::from_secs(10));
}

#[test]
fn an_answer_between_two_failures_starts_the_row_again_and_a_link_still_connecting_stays_so() {
    // Given - a window still connecting
    let mut watch = Watch::default();
    let connecting = Link::Connecting {
        socket: String::from(SOCKET),
    };

    // When - two hellos fail, one answers, then two fail again
    let heard = played(
        &mut watch,
        connecting.clone(),
        vec![
            Err(refused()),
            Err(refused()),
            Ok(json!({ "out": {} })),
            Err(refused()),
            Err(refused()),
        ],
    );

    // Then - connecting until the answer, then open through the two failures that follow it
    let links: Vec<Link> = heard.into_iter().map(|(_, link)| link).collect();
    assert_eq!(
        links,
        vec![connecting.clone(), connecting, open(), open(), open()]
    );
    assert_eq!(watch.failed(), 2);
}

#[test]
fn a_declared_absence_is_retried_after_a_doubling_backoff_and_a_return_resets_it() {
    // Given - a link open
    let mut watch = Watch::default();

    // When - eight hellos fail, then one answers, then three fail again
    let mut heard = vec![Err(refused()); 8];
    heard.push(Ok(json!({ "out": {} })));
    heard.extend(vec![Err(refused()); 3]);
    let shown = played(&mut watch, open(), heard);

    // Then - 5 s twice, then the backoff doubles up to its cap; the answer at 71 s is checked again when its stream is
    // cut, and the next absence is declared 10 s after its first failure, the backoff at its start again
    let instants: Vec<u64> = shown.iter().map(|(at, _)| *at).collect();
    assert_eq!(instants, vec![0, 5, 10, 11, 13, 17, 25, 41, 71, 71, 76, 81]);
    assert_eq!(MAX_BACKOFF, 30);
    assert!(matches!(shown[8].1, Link::Open { .. }));
    assert_eq!(
        shown[11].1,
        Link::Absent {
            socket: String::from(SOCKET),
            reason: String::from("refused"),
            retry_in: 1000
        }
    );
}

#[test]
fn a_failed_hello_that_keeps_the_link_is_told_with_its_place_in_the_row() {
    // Given - a link open, and a refused connect
    let link = open();

    // When - the first failure of a row is told
    let line = kept(&link, &refused(), 1);

    // Then - the link it kept, what the hello heard, and its place in the row
    assert_eq!(
        line,
        "link open: hello failed at connect: refused (1 of 3 in a row, link kept)"
    );
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

#[test]
fn a_link_change_is_told_by_its_kind_and_what_the_hello_heard_never_its_answer_or_socket() {
    // Given - a link opened by an answer naming the engine's executable, and one lost to a refused connect
    let open = Link::Open {
        socket: String::from("/Users/you/.local/state/station/engine.sock"),
        answer: json!({ "out": { "executable": "/Users/you/.local/bin/station", "pid": 7 } }),
    };
    let absent = Link::Absent {
        socket: String::from("/Users/you/.local/state/station/engine.sock"),
        reason: String::from("refused"),
        retry_in: 1000,
    };

    // When - each change is told
    let lines = [told(&open, None), told(&absent, Some(&refused()))];

    // Then - the kind and the reason, with no path from the answer or the socket
    assert_eq!(
        lines,
        [
            "link open: answered",
            "link absent: hello failed at connect: refused"
        ]
    );
}

#[test]
fn an_answer_is_said_again_only_when_its_stream_is_cut_or_a_wake_comes_never_after_a_time() {
    // Given - a link that answers
    let mut watch = Watch::default();

    // When - its hello is heard
    let (link, wait) = watch.next(Ok(json!({ "out": {} })), String::from(SOCKET), &open());

    // Then - the link is open, and its rest has no end of its own
    assert_eq!((link, wait), (open(), None));
}

#[test]
fn a_rest_with_no_end_ends_on_a_cut_and_a_timed_rest_never_does() {
    // Given - a bridge whose held state stream is cut before its supervisor rests
    let bridge = Bridge::new();
    bridge.exchanges.alarm.cut();

    // When - it rests with no end, then a timed rest follows a second cut
    let untimed = bridge.rest(None);
    bridge.exchanges.alarm.cut();
    let started = Instant::now();
    let timed = bridge.rest(Some(Duration::from_millis(50)));

    // Then - the cut ended the first at once, unwoken; the second ran its whole wait, the cut ignored
    assert!(!untimed);
    assert!(!timed);
    assert!(started.elapsed() >= Duration::from_millis(50));
}

#[test]
fn a_wake_cuts_the_supervisors_rest_short_once() {
    // Given - a bridge whose supervisor rests a minute, and a core started meanwhile
    let bridge = Bridge::new();
    let waker = Arc::clone(&bridge);
    let started = Instant::now();
    let wakes = thread::spawn(move || {
        thread::sleep(Duration::from_millis(50));
        waker.wake();
    });

    // When - it rests, then rests again
    let first = bridge.rest(Some(Duration::from_secs(60)));
    let woke_after = started.elapsed();
    let second = bridge.rest(Some(Duration::from_millis(20)));
    let _ = wakes.join();

    // Then - the first rest ends on the wake, well before its minute, and the next one is a plain rest
    assert!(first);
    assert!(woke_after < Duration::from_secs(5));
    assert!(!second);
}

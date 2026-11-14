use std::collections::HashMap;
use std::io::{self, BufRead, BufReader, Read, Write};
use std::net::Shutdown;
use std::os::unix::net::UnixStream;
use std::path::Path;
use std::sync::atomic::{AtomicU32, Ordering};
use std::sync::{mpsc, Arc, Mutex};
use std::thread;
use std::time::Duration;

use serde::Serialize;
use serde_json::{json, Map, Value};

include!(concat!(env!("OUT_DIR"), "/bridged.rs"));

const API: &str = "station.api/v2";
const BAD_FRAME: &str = "call.bad-frame";
pub const NOT_BRIDGED: &str = "bridge.not-bridged";
pub const NOT_OPENED: &str = "bridge.not-opened";
const REQUEST_KEYS: [&str; 6] = ["v", "id", "call", "watch", "target", "in"];

#[derive(Debug, PartialEq)]
pub struct Refusal {
    pub code: &'static str,
    pub message: String,
}

impl Refusal {
    fn bad_frame(message: impl Into<String>) -> Self {
        Refusal {
            code: BAD_FRAME,
            message: message.into(),
        }
    }
}

#[derive(Clone, Debug, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum Message {
    Frame(Value),
    End(String),
}

pub trait Sink: Send + 'static {
    fn send(&self, message: Message) -> bool;
}

#[derive(Clone, Copy)]
pub struct Limits {
    pub connect: Duration,
    pub write: Duration,
    // value: the longest request line the engine reads (`remote/serve`'s scanner, 1 MiB).
    pub request: usize,
    // value: the longest answer line the bridge holds (`remote/api`'s client, 64 MiB).
    pub answer: usize,
}

pub const LIMITS: Limits = Limits {
    connect: Duration::from_secs(3),
    write: Duration::from_secs(3),
    request: 1 << 20,
    answer: 64 << 20,
};

fn wire_error(id: &str, kind: &str, code: &str, message: String) -> Value {
    json!({
        "v": 2,
        "id": id,
        "error": { "type": kind, "code": code, "subject": API, "message": message },
    })
}

fn named(frame: &Map<String, Value>, key: &str) -> bool {
    frame
        .get(key)
        .and_then(Value::as_str)
        .is_some_and(|name| !name.is_empty())
}

fn targeted(frame: &Map<String, Value>) -> bool {
    match frame.get("target") {
        None => true,
        Some(Value::Object(target)) => target
            .iter()
            .all(|(key, value)| (key == "name" || key == "on") && value.is_string()),
        Some(_) => false,
    }
}

fn name_of(frame: &Map<String, Value>) -> &str {
    frame
        .get("call")
        .or_else(|| frame.get("watch"))
        .and_then(Value::as_str)
        .unwrap_or_default()
}

// invariant: the envelope is checked as the engine checks it (`remote/serve`): v 2, a string id, exactly one of
// invariant: call and watch, a target of name and on; then the name, which is one of BRIDGED, the calls the window
// invariant: sends (`src/engine/bridged.ts`). Whether its `in` fits is the engine's to answer.
pub fn refusal(request: &Value, limit: usize) -> Option<Refusal> {
    let Value::Object(frame) = request else {
        return Some(Refusal::bad_frame("a request is one JSON object"));
    };
    if frame.get("v") != Some(&json!(2)) || !frame.get("id").is_some_and(Value::is_string) {
        return Some(Refusal::bad_frame("a request carries v 2 and a string id"));
    }
    if named(frame, "call") == named(frame, "watch") {
        return Some(Refusal::bad_frame(
            "a request names a call or a watch, never both",
        ));
    }
    if let Some(key) = frame
        .keys()
        .find(|key| !REQUEST_KEYS.contains(&key.as_str()))
    {
        return Some(Refusal::bad_frame(format!(
            "a request carries no field {key}"
        )));
    }
    if !targeted(frame) {
        return Some(Refusal::bad_frame(
            "a target names a machine and where it runs",
        ));
    }
    let length = serde_json::to_vec(request).map_or(usize::MAX, |line| line.len());
    if length > limit {
        return Some(Refusal::bad_frame(format!(
            "a request is at most {limit} bytes, this one {length}"
        )));
    }
    let name = name_of(frame);
    (!BRIDGED.contains(&name)).then(|| Refusal {
        code: NOT_BRIDGED,
        message: format!("the window does not send {name}"),
    })
}

fn ends(watch: bool, answer: &Value) -> bool {
    answer.get("error").is_some()
        || if watch {
            answer.get("done").is_some()
        } else {
            answer.get("out").is_some()
        }
}

pub fn connect_within(path: &Path, within: Duration) -> io::Result<UnixStream> {
    let (sent, connected) = mpsc::channel();
    let owned = path.to_path_buf();
    // hazard: a unix connect blocks while the engine's backlog is full and std gives it no timeout; the thread
    // hazard: left behind ends when the engine accepts or goes, and drops what it connected.
    thread::spawn(move || {
        let _ = sent.send(UnixStream::connect(owned));
    });
    match connected.recv_timeout(within) {
        Ok(stream) => stream,
        Err(_) => Err(io::Error::new(
            io::ErrorKind::TimedOut,
            format!("no connection within {} ms", within.as_millis()),
        )),
    }
}

pub fn write_line(stream: &mut UnixStream, value: &Value) -> io::Result<()> {
    let mut line = serde_json::to_vec(value)?;
    line.push(b'\n');
    stream.write_all(&line)
}

enum Line {
    Whole(Vec<u8>),
    Cut,
    TooLong,
    Closed,
}

fn line_of(reader: &mut impl BufRead, limit: usize) -> io::Result<Line> {
    let mut line = Vec::new();
    let read = reader
        .take(u64::try_from(limit).unwrap_or(u64::MAX).saturating_add(1))
        .read_until(b'\n', &mut line)?;
    if read == 0 {
        return Ok(Line::Closed);
    }
    if line.last() == Some(&b'\n') {
        line.pop();
        return Ok(Line::Whole(line));
    }
    Ok(if line.len() > limit {
        Line::TooLong
    } else {
        Line::Cut
    })
}

// invariant: one exchange's answers in order, until its last: a line that is no JSON or is longer than the limit
// invariant: ends it with the engine's own bad-frame error, a connection that ends mid-line or early ends it
// invariant: unanswered, and a sink that no longer listens ends it at once.
pub fn pump(reader: impl Read, request: &Value, sink: &impl Sink, limit: usize) -> String {
    let id = request
        .get("id")
        .and_then(Value::as_str)
        .unwrap_or_default();
    let watch = request.get("watch").is_some();
    let mut reader = BufReader::new(reader);
    loop {
        let refused = match line_of(&mut reader, limit) {
            Err(error) => return error.to_string(),
            Ok(Line::Closed | Line::Cut) => {
                return String::from("the engine closed the connection")
            }
            Ok(Line::TooLong) => format!("a line from the engine is longer than {limit} bytes"),
            Ok(Line::Whole(line)) if line.iter().all(u8::is_ascii_whitespace) => continue,
            Ok(Line::Whole(line)) => match serde_json::from_slice::<Value>(&line) {
                Ok(answer) => {
                    let last = ends(watch, &answer);
                    if !sink.send(Message::Frame(answer)) {
                        return String::from("the window stopped listening");
                    }
                    if last {
                        return String::from("answered");
                    }
                    continue;
                }
                Err(error) => format!("a line from the engine is no JSON: {error}"),
            },
        };
        sink.send(Message::Frame(wire_error(id, "failed", BAD_FRAME, refused)));
        return String::from("answered");
    }
}

struct Open {
    owner: String,
    id: String,
    stream: UnixStream,
}

// invariant: a cancel frame is exactly `{v: 2, id, cancel: true}` (`CancelFrame`, packages/api/schemas/frame.json).
fn cancel_id(frame: &Value) -> Option<&str> {
    let Value::Object(fields) = frame else {
        return None;
    };
    let shaped = fields.len() == 3
        && fields.get("v") == Some(&json!(2))
        && fields.get("cancel") == Some(&json!(true));
    if !shaped {
        return None;
    }
    fields.get("id").and_then(Value::as_str)
}

#[derive(Default)]
pub struct Exchanges {
    next: AtomicU32,
    open: Mutex<HashMap<u32, Open>>,
}

impl Exchanges {
    fn held(&self, handle: u32) -> Option<UnixStream> {
        self.open
            .lock()
            .ok()?
            .remove(&handle)
            .map(|open| open.stream)
    }

    // value: 0 names no exchange: a request the bridge refused, answered on the sink alone.
    pub fn open<S: Sink>(
        self: &Arc<Self>,
        path: &Path,
        owner: &str,
        request: Value,
        sink: S,
        limits: Limits,
    ) -> Result<u32, String> {
        if let Some(refused) = refusal(&request, limits.request) {
            let id = request
                .get("id")
                .and_then(Value::as_str)
                .ok_or(refused.message.clone())?;
            sink.send(Message::Frame(wire_error(
                id,
                "refused",
                refused.code,
                refused.message,
            )));
            sink.send(Message::End(String::from("answered")));
            return Ok(0);
        }
        let unanswered =
            |error: io::Error| format!("no engine answers at {}: {error}", path.display());
        let mut stream = connect_within(path, limits.connect).map_err(unanswered)?;
        stream
            .set_write_timeout(Some(limits.write))
            .map_err(unanswered)?;
        let reader = stream.try_clone().map_err(unanswered)?;
        write_line(&mut stream, &request).map_err(unanswered)?;
        let handle = self.next.fetch_add(1, Ordering::SeqCst) + 1;
        let id = request
            .get("id")
            .and_then(Value::as_str)
            .unwrap_or_default()
            .to_owned();
        if let Ok(mut open) = self.open.lock() {
            open.insert(
                handle,
                Open {
                    owner: owner.to_owned(),
                    id,
                    stream,
                },
            );
        }
        let exchanges = Arc::clone(self);
        thread::spawn(move || {
            let reason = pump(reader, &request, &sink, limits.answer);
            exchanges.close(handle);
            sink.send(Message::End(reason));
        });
        Ok(handle)
    }

    // invariant: a cancel reaches the engine only on the exchange it names, opened by the same webview, and only
    // invariant: for the id of the request that exchange carries; any other frame, id or exchange is refused unsent.
    pub fn cancel(&self, handle: u32, owner: &str, frame: &Value) -> Result<(), Refusal> {
        let Some(id) = cancel_id(frame) else {
            return Err(Refusal::bad_frame(
                "a cancel is v 2, a string id and cancel true",
            ));
        };
        let held = self.open.lock().ok().and_then(|open| {
            open.get(&handle)
                .filter(|held| held.owner == owner && held.id == id)
                .and_then(|held| held.stream.try_clone().ok())
        });
        let Some(mut stream) = held else {
            return Err(Refusal {
                code: NOT_OPENED,
                message: format!("the window has no request {id} open to cancel"),
            });
        };
        write_line(&mut stream, frame).map_err(|error| Refusal {
            code: NOT_OPENED,
            message: format!("the cancel of {id} was not sent: {error}"),
        })
    }

    pub fn close(&self, handle: u32) {
        if let Some(stream) = self.held(handle) {
            let _ = stream.shutdown(Shutdown::Both);
        }
    }

    // invariant: a webview that reloads or goes takes its exchanges with it: their sockets shut, their readers
    // invariant: wake on the end of their stream and stop, however quiet the watch they read.
    pub fn close_owner(&self, owner: &str) -> usize {
        let Ok(mut open) = self.open.lock() else {
            return 0;
        };
        let handles: Vec<u32> = open
            .iter()
            .filter(|(_, held)| held.owner == owner)
            .map(|(handle, _)| *handle)
            .collect();
        for handle in &handles {
            if let Some(held) = open.remove(handle) {
                let _ = held.stream.shutdown(Shutdown::Both);
            }
        }
        handles.len()
    }

    #[cfg(test)]
    pub fn count(&self) -> usize {
        self.open.lock().map_or(0, |open| open.len())
    }
}

#[cfg(test)]
#[path = "bridge_test.rs"]
mod tests;

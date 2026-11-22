use std::env;
use std::io::{BufRead, BufReader};
use std::net::Shutdown;
use std::path::{Path, PathBuf};
use std::sync::{Arc, Condvar, Mutex};
use std::thread;
use std::time::{Duration, SystemTime};

use serde::Serialize;
use serde_json::{json, Value};
use tauri::ipc::Channel;
use tauri::{AppHandle, Emitter, Manager, State, Webview};

use crate::bridge::{connect_within, write_line, Exchanges, Message, Sink, LIMITS};
use crate::local_core::{moved, Runs};

#[path = "link_log.rs"]
mod link_log;
use link_log::LinkLog;

const FIRST_BACKOFF: u64 = 1;
const MAX_BACKOFF: u64 = 30;
const RECHECK: Duration = Duration::from_secs(10);
const HELLO_WITHIN: Duration = Duration::from_secs(5);
const RETRY: Duration = Duration::from_secs(5);
const ABSENT_AFTER: u32 = 3;

#[derive(Clone, Debug, PartialEq, Serialize)]
#[serde(tag = "kind", rename_all = "lowercase")]
pub enum Link {
    Connecting {
        socket: String,
    },
    Open {
        socket: String,
        answer: Value,
    },
    Absent {
        socket: String,
        reason: String,
        #[serde(rename = "retryIn")]
        retry_in: u64,
    },
}

pub struct Bridge {
    pub exchanges: Arc<Exchanges>,
    link: Mutex<Link>,
    woken: Mutex<bool>,
    nudge: Condvar,
}

impl Bridge {
    pub fn new() -> Arc<Self> {
        Arc::new(Bridge {
            exchanges: Arc::new(Exchanges::default()),
            link: Mutex::new(Link::Connecting {
                socket: socket_path().display().to_string(),
            }),
            woken: Mutex::new(false),
            nudge: Condvar::new(),
        })
    }

    pub fn wake(&self) {
        if let Ok(mut woken) = self.woken.lock() {
            *woken = true;
            self.nudge.notify_all();
        }
    }

    pub fn shown(&self) -> Link {
        self.link
            .lock()
            .map(|link| link.clone())
            .unwrap_or(Link::Connecting {
                socket: socket_path().display().to_string(),
            })
    }

    pub fn rest(&self, wait: Duration) -> bool {
        let Ok(woken) = self.woken.lock() else {
            thread::sleep(wait);
            return false;
        };
        let (mut woken, _) = self
            .nudge
            .wait_timeout_while(woken, wait, |woken| !*woken)
            .unwrap_or_else(std::sync::PoisonError::into_inner);
        std::mem::replace(&mut *woken, false)
    }
}

impl Sink for Channel<Message> {
    fn send(&self, message: Message) -> bool {
        Channel::send(self, message).is_ok()
    }
}

pub fn socket_path() -> PathBuf {
    if let Some(path) = env::var_os("STATION_ENGINE_SOCKET").filter(|v| !v.is_empty()) {
        return PathBuf::from(path);
    }
    if let Some(state) = env::var_os("XDG_STATE_HOME").filter(|v| !v.is_empty()) {
        return PathBuf::from(state).join("station/engine.sock");
    }
    let home = env::var_os("HOME").unwrap_or_default();
    PathBuf::from(home).join(".local/state/station/engine.sock")
}

// invariant: every command runs off the main thread (Tauri 2 runs a command that is not async on it), so an
// invariant: engine that stalls never freezes the window.
#[tauri::command(async)]
pub fn engine_link(bridge: State<'_, Arc<Bridge>>) -> Link {
    bridge.shown()
}

#[tauri::command]
pub async fn engine_open(
    bridge: State<'_, Arc<Bridge>>,
    webview: Webview,
    request: Value,
    channel: Channel<Message>,
) -> Result<u32, String> {
    let exchanges = Arc::clone(&bridge.exchanges);
    let owner = webview.label().to_owned();
    tauri::async_runtime::spawn_blocking(move || {
        exchanges.open(&socket_path(), &owner, request, channel, LIMITS)
    })
    .await
    .map_err(|error| error.to_string())?
}

#[tauri::command(async)]
pub fn engine_cancel(
    bridge: State<'_, Arc<Bridge>>,
    webview: Webview,
    handle: u32,
    frame: Value,
) -> Result<(), String> {
    bridge
        .exchanges
        .cancel(handle, webview.label(), &frame)
        .map_err(|refused| format!("{}: {}", refused.code, refused.message))
}

#[tauri::command(async)]
pub fn engine_close(bridge: State<'_, Arc<Bridge>>, handle: u32) {
    bridge.exchanges.close(handle);
}

// value: a hello that failed: the step it failed at (connect, write, read, answer) and what that step said.
#[derive(Clone, Debug, PartialEq)]
pub struct Failed {
    pub step: &'static str,
    pub said: String,
}

fn at(step: &'static str) -> impl Fn(std::io::Error) -> Failed {
    move |error| Failed {
        step,
        said: error.to_string(),
    }
}

fn hello(path: &Path) -> Result<Value, Failed> {
    let mut stream = connect_within(path, LIMITS.connect).map_err(at("connect"))?;
    stream
        .set_read_timeout(Some(HELLO_WITHIN))
        .and_then(|()| stream.set_write_timeout(Some(LIMITS.write)))
        .map_err(at("connect"))?;
    let request = json!({ "v": 2, "id": "hello", "call": "engine.hello" });
    write_line(&mut stream, &request).map_err(at("write"))?;
    let reader = stream.try_clone().map_err(at("read"))?;
    for line in BufReader::new(reader).lines() {
        let line = line.map_err(at("read"))?;
        let answer: Value = serde_json::from_str(&line).map_err(|error| Failed {
            step: "answer",
            said: error.to_string(),
        })?;
        if answer.get("out").is_some() || answer.get("error").is_some() {
            let _ = stream.shutdown(Shutdown::Both);
            return Ok(answer);
        }
    }
    Err(Failed {
        step: "read",
        said: String::from("the engine closed the connection before it answered"),
    })
}

// value: the link log's line for a change: the link's kind, then what the hello that made it heard. Neither the
// value: hello's answer nor the socket is written: an answer carries the engine's executable path and identity.
fn told(link: &Link, failed: Option<&Failed>) -> String {
    let kind = match link {
        Link::Connecting { .. } => "connecting",
        Link::Open { .. } => "open",
        Link::Absent { .. } => "absent",
    };
    match failed {
        None => format!("link {kind}: answered"),
        Some(failed) => format!(
            "link {kind}: hello failed at {}: {}",
            failed.step, failed.said
        ),
    }
}

// value: the line for a failed hello that left the link as it was, counted in its row.
fn kept(link: &Link, failed: &Failed, in_a_row: u32) -> String {
    format!(
        "{} ({in_a_row} of {ABSENT_AFTER} in a row, link kept)",
        told(link, Some(failed))
    )
}

// value: what the supervisor carries from one hello to the next: the hellos failed in a row, and the backoff of a
// value: declared absence.
pub struct Watch {
    failed: u32,
    backoff: u64,
}

impl Default for Watch {
    fn default() -> Self {
        Watch {
            failed: 0,
            backoff: FIRST_BACKOFF,
        }
    }
}

impl Watch {
    pub fn woken(&mut self) {
        self.backoff = FIRST_BACKOFF;
    }

    pub fn failed(&self) -> u32 {
        self.failed
    }

    // invariant: an answer opens the link and is checked again after RECHECK. A failure leaves the link as shown and
    // invariant: is tried again after RETRY, until the ABSENT_AFTER-th in a row declares it absent; from there the wait
    // invariant: doubles from FIRST_BACKOFF up to MAX_BACKOFF, said in the link as `retryIn` milliseconds.
    pub fn next(
        &mut self,
        heard: Result<Value, Failed>,
        socket: String,
        shown: &Link,
    ) -> (Link, Duration) {
        let reason = match heard {
            Ok(answer) => {
                self.failed = 0;
                self.backoff = FIRST_BACKOFF;
                return (Link::Open { socket, answer }, RECHECK);
            }
            Err(Failed { said, .. }) => said,
        };
        self.failed = self.failed.saturating_add(1);
        if self.failed < ABSENT_AFTER {
            return (shown.clone(), RETRY);
        }
        let wait = self.backoff;
        self.backoff = (wait * 2).min(MAX_BACKOFF);
        (
            Link::Absent {
                socket,
                reason,
                retry_in: wait * 1000,
            },
            Duration::from_secs(wait),
        )
    }
}

fn changed(shown: &Mutex<Link>, link: &Link) -> bool {
    shown
        .lock()
        .map(|mut shown| {
            let changed = *shown != *link;
            *shown = link.clone();
            changed
        })
        .unwrap_or(false)
}

// invariant: the first engine to answer after the window's launch is judged once for the B to A move, on a thread
// invariant: of its own, so the link is supervised meanwhile; a move that ran wakes the link.
fn judged_once(app: &AppHandle, bridge: &Arc<Bridge>, runs: &Arc<Runs>, answer: &Value) {
    let (log, bridge, runs, answer) = (
        LinkLog::in_folder(app.path().app_log_dir().ok()),
        Arc::clone(bridge),
        Arc::clone(runs),
        answer.clone(),
    );
    thread::spawn(move || {
        let said = match moved(&runs, &answer) {
            None => return,
            Some(Ok(ended)) if ended.code == 0 => {
                bridge.wake();
                String::from("moved the bare engine into Spwn.app")
            }
            Some(Ok(ended)) => format!("the move into Spwn.app exited {}", ended.code),
            Some(Err(refusal)) => {
                format!("the move into Spwn.app did not run: {}", refusal.message)
            }
        };
        log.write(SystemTime::now(), &said);
    });
}

pub fn supervise(app: AppHandle, bridge: Arc<Bridge>, runs: Arc<Runs>) {
    let log = LinkLog::in_folder(app.path().app_log_dir().ok());
    thread::spawn(move || {
        let mut watch = Watch::default();
        let mut judged = false;
        loop {
            let path = socket_path();
            let socket = path.display().to_string();
            let heard = hello(&path);
            if let (false, Ok(answer)) = (judged, heard.as_ref()) {
                judged = true;
                judged_once(&app, &bridge, &runs, answer);
            }
            let failed = heard.as_ref().err().cloned();
            let (link, wait) = watch.next(heard, socket, &bridge.shown());
            if changed(&bridge.link, &link) {
                log.write(SystemTime::now(), &told(&link, failed.as_ref()));
                let _ = app.emit("engine://link", &link);
            } else if let Some(failed) = failed.as_ref().filter(|_| watch.failed() < ABSENT_AFTER) {
                log.write(SystemTime::now(), &kept(&link, failed, watch.failed()));
            }
            if bridge.rest(wait) {
                watch.woken();
            }
        }
    });
}

#[cfg(test)]
#[path = "engine_test.rs"]
mod tests;

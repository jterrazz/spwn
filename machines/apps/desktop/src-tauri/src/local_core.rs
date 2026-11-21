use std::env;
use std::ffi::{CStr, OsStr, OsString};
use std::io::{BufRead, BufReader, Read};
use std::os::unix::ffi::OsStringExt;
use std::os::unix::process::{CommandExt, ExitStatusExt};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::mpsc;
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, Instant};

use serde::Serialize;
use tauri::ipc::Channel;
use tauri::State;

use crate::engine::Bridge;
use crate::opener::Refusal;
use crate::station::{bundled_of, station_of, Where, INSTALL_LINE};

pub const SEARCH_PATH: &str = "/usr/bin:/bin:/usr/sbin:/sbin";
pub const FALLBACK_SHELL: &str = "/bin/zsh";
pub const BUSY: &str = "core.busy";
pub const NOT_RUN: &str = "core.not-run";
pub const FAKED_BY: &str = "STATION_WINDOW_CORE";

const PASSED_THROUGH: [&str; 3] = ["USER", "LOGNAME", "TMPDIR"];
const READERS_WITHIN: Duration = Duration::from_secs(2);

#[derive(Clone, Copy, Debug, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum Stream {
    Out,
    Err,
}

#[derive(Clone, Debug, PartialEq, Serialize)]
pub struct Line {
    pub stream: Stream,
    pub text: String,
}

#[derive(Debug, PartialEq, Serialize)]
pub struct Ended {
    pub code: i32,
    pub cancelled: bool,
    pub lines: usize,
}

struct Counted {
    heard: Arc<dyn Heard>,
    count: AtomicUsize,
}

impl Heard for Counted {
    fn heard(&self, line: Line) {
        self.count.fetch_add(1, Ordering::SeqCst);
        self.heard.heard(line);
    }
}

#[derive(Debug, PartialEq)]
pub struct Plan {
    pub program: PathBuf,
    pub args: Vec<OsString>,
    pub env: Vec<(String, OsString)>,
}

pub trait Heard: Send + Sync {
    fn heard(&self, line: Line);
}

impl Heard for Channel<Line> {
    fn heard(&self, line: Line) {
        let _ = self.send(line);
    }
}

fn refused(code: &'static str, message: impl Into<String>) -> Refusal {
    Refusal {
        code,
        message: message.into(),
    }
}

pub fn home() -> PathBuf {
    env::var_os("HOME")
        .filter(|home| !home.is_empty())
        .map_or_else(
            || {
                passwd_field(|entry| entry.pw_dir)
                    .unwrap_or_default()
                    .into()
            },
            PathBuf::from,
        )
}

fn passwd_field(field: fn(&libc::passwd) -> *mut libc::c_char) -> Option<OsString> {
    let mut entry: libc::passwd = unsafe { std::mem::zeroed() };
    let mut buffer = vec![0 as libc::c_char; 16 * 1024];
    let mut found: *mut libc::passwd = std::ptr::null_mut();
    let code = unsafe {
        libc::getpwuid_r(
            libc::getuid(),
            &mut entry,
            buffer.as_mut_ptr(),
            buffer.len(),
            &mut found,
        )
    };
    let pointer = field(&entry);
    if code != 0 || found.is_null() || pointer.is_null() {
        return None;
    }
    let value = unsafe { CStr::from_ptr(pointer) };
    Some(OsString::from_vec(value.to_bytes().to_vec())).filter(|value| !value.is_empty())
}

pub fn login_shell() -> OsString {
    passwd_field(|entry| entry.pw_shell)
        .or_else(|| env::var_os("SHELL").filter(|shell| !shell.is_empty()))
        .unwrap_or_else(|| OsString::from(FALLBACK_SHELL))
}

pub fn environment(
    home: &Path,
    shell: &OsStr,
    from: impl Fn(&str) -> Option<OsString>,
) -> Vec<(String, OsString)> {
    let mut env = vec![
        (String::from("HOME"), home.as_os_str().to_owned()),
        (String::from("PATH"), OsString::from(SEARCH_PATH)),
        (String::from("SHELL"), shell.to_owned()),
    ];
    for name in PASSED_THROUGH {
        if let Some(value) = from(name).filter(|value| !value.is_empty()) {
            env.push((String::from(name), value));
        }
    }
    env
}

pub fn piped(line: &str, env: Vec<(String, OsString)>) -> Plan {
    Plan {
        program: PathBuf::from("/bin/bash"),
        args: ["-o", "pipefail", "-c", line].map(OsString::from).to_vec(),
        env,
    }
}

fn faked_or_refused(faked: Option<&Path>, what: &str) -> Result<Option<PathBuf>, Refusal> {
    match faked {
        Some(dir) => Ok(Some(dir.to_path_buf())),
        None if cfg!(debug_assertions) => Err(refused(
            NOT_RUN,
            format!("a debug build runs {what} only under {FAKED_BY}"),
        )),
        None => Ok(None),
    }
}

pub fn install_plan(faked: Option<&Path>, env: Vec<(String, OsString)>) -> Result<Plan, Refusal> {
    Ok(match faked_or_refused(faked, "no installer")? {
        Some(dir) => Plan {
            program: PathBuf::from("/bin/sh"),
            args: vec![dir.join("install.sh").into_os_string()],
            env,
        },
        None => piped(INSTALL_LINE, env),
    })
}

pub fn start_plan(
    faked: Option<&Path>,
    station: Option<PathBuf>,
    env: Vec<(String, OsString)>,
) -> Result<Plan, Refusal> {
    faked_or_refused(faked, "no engine")?;
    let program = station.ok_or_else(|| {
        refused(
            NOT_RUN,
            "station is neither where the engine said, nor in its record, nor in the install folder",
        )
    })?;
    Ok(Plan {
        program,
        args: ["engine", "start"].map(OsString::from).to_vec(),
        env,
    })
}

fn text_of(mut bytes: Vec<u8>) -> String {
    while bytes
        .last()
        .is_some_and(|last| *last == b'\n' || *last == b'\r')
    {
        bytes.pop();
    }
    String::from_utf8_lossy(&bytes).into_owned()
}

fn reader(
    from: Option<impl Read + Send + 'static>,
    stream: Stream,
    heard: Arc<dyn Heard>,
    done: mpsc::Sender<()>,
) {
    thread::spawn(move || {
        if let Some(from) = from {
            let mut lines = BufReader::new(from);
            let mut bytes = Vec::new();
            while lines
                .read_until(b'\n', &mut bytes)
                .is_ok_and(|read| read > 0)
            {
                heard.heard(Line {
                    stream,
                    text: text_of(std::mem::take(&mut bytes)),
                });
            }
        }
        let _ = done.send(());
    });
}

struct Running {
    group: i32,
    cancelled: bool,
}

#[derive(Default)]
pub struct Runs {
    running: Mutex<Option<Running>>,
}

impl Runs {
    pub fn run(&self, plan: &Plan, heard: Arc<dyn Heard>) -> Result<Ended, Refusal> {
        let mut child = {
            let mut running = self
                .running
                .lock()
                .map_err(|_| refused(NOT_RUN, "the window lost track of its runs"))?;
            if running.is_some() {
                return Err(refused(
                    BUSY,
                    "the window runs one install or start at a time",
                ));
            }
            let child = Command::new(&plan.program)
                .args(&plan.args)
                .env_clear()
                .envs(plan.env.iter().map(|(name, value)| (name, value)))
                .stdin(Stdio::null())
                .stdout(Stdio::piped())
                .stderr(Stdio::piped())
                .process_group(0)
                .spawn()
                .map_err(|error| {
                    refused(
                        NOT_RUN,
                        format!("{} could not run: {error}", plan.program.display()),
                    )
                })?;
            *running = Some(Running {
                group: i32::try_from(child.id()).unwrap_or(0),
                cancelled: false,
            });
            child
        };
        let counted = Arc::new(Counted {
            heard,
            count: AtomicUsize::new(0),
        });
        let (done, finished) = mpsc::channel();
        let out: Arc<dyn Heard> = counted.clone();
        let err: Arc<dyn Heard> = counted.clone();
        reader(child.stdout.take(), Stream::Out, out, done.clone());
        reader(child.stderr.take(), Stream::Err, err, done);
        let status = child.wait();
        let until = Instant::now() + READERS_WITHIN;
        for _ in 0..2 {
            let left = until.saturating_duration_since(Instant::now());
            if finished.recv_timeout(left).is_err() {
                break;
            }
        }
        let cancelled = self
            .running
            .lock()
            .map(|mut running| running.take().is_some_and(|running| running.cancelled))
            .unwrap_or(false);
        let status = status.map_err(|error| refused(NOT_RUN, error.to_string()))?;
        let code = status
            .code()
            .or_else(|| status.signal().map(|signal| 128 + signal))
            .unwrap_or(-1);
        Ok(Ended {
            code,
            cancelled,
            lines: counted.count.load(Ordering::SeqCst),
        })
    }

    pub fn cancel(&self) -> bool {
        let Ok(mut running) = self.running.lock() else {
            return false;
        };
        match running.as_mut() {
            Some(run) if run.group > 0 => {
                run.cancelled = true;
                unsafe { libc::kill(-run.group, libc::SIGTERM) == 0 }
            }
            _ => false,
        }
    }
}

#[cfg(debug_assertions)]
fn faked_dir() -> Option<PathBuf> {
    env::var_os(FAKED_BY)
        .filter(|dir| !dir.is_empty())
        .map(PathBuf::from)
}

#[cfg(not(debug_assertions))]
fn faked_dir() -> Option<PathBuf> {
    None
}

fn child_env() -> Vec<(String, OsString)> {
    environment(&home(), &login_shell(), |name: &str| env::var_os(name))
}

fn station(faked: Option<&Path>, said: Option<&Path>) -> Option<PathBuf> {
    let bundled = env::current_exe().ok().and_then(|exe| bundled_of(&exe));
    station_of(Where {
        faked,
        said,
        bundled: bundled.as_deref(),
        home: &home(),
        xdg_state: env::var_os("XDG_STATE_HOME"),
    })
}

async fn ran(runs: Arc<Runs>, plan: Plan, channel: Channel<Line>) -> Result<Ended, Refusal> {
    tauri::async_runtime::spawn_blocking(move || runs.run(&plan, Arc::new(channel)))
        .await
        .map_err(|error| refused(NOT_RUN, error.to_string()))?
}

#[tauri::command(async)]
pub fn core_present() -> bool {
    station(faked_dir().as_deref(), None).is_some()
}

#[tauri::command]
pub async fn core_install(
    runs: State<'_, Arc<Runs>>,
    channel: Channel<Line>,
) -> Result<Ended, Refusal> {
    let plan = install_plan(faked_dir().as_deref(), child_env())?;
    ran(Arc::clone(&runs), plan, channel).await
}

#[tauri::command]
pub async fn core_start(
    runs: State<'_, Arc<Runs>>,
    bridge: State<'_, Arc<Bridge>>,
    channel: Channel<Line>,
    executable: Option<String>,
) -> Result<Ended, Refusal> {
    let faked = faked_dir();
    let said = executable.map(PathBuf::from);
    let found = station(faked.as_deref(), said.as_deref());
    let plan = start_plan(faked.as_deref(), found, child_env())?;
    let ended = ran(Arc::clone(&runs), plan, channel).await?;
    if ended.code == 0 {
        bridge.wake();
    }
    Ok(ended)
}

#[tauri::command(async)]
pub fn core_cancel(runs: State<'_, Arc<Runs>>) -> bool {
    runs.cancel()
}

#[cfg(test)]
#[path = "local_core_test.rs"]
mod tests;

use std::env;
use std::fs::{self, DirBuilder, OpenOptions};
use std::io::{self, Write};
use std::os::unix::fs::{DirBuilderExt, OpenOptionsExt, PermissionsExt};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};
use std::sync::atomic::{AtomicU32, Ordering};
use std::time::{SystemTime, UNIX_EPOCH};

use serde::Serialize;

pub const NOT_A_WEB_ADDRESS: &str = "opener.not-a-web-address";
pub const NOT_ONE_LINE: &str = "opener.not-one-line";
pub const NOT_OPENED: &str = "opener.not-opened";
pub const NOT_A_SETTINGS_PANE: &str = "opener.not-a-settings-pane";

// value: the longest address and command line the window opens; an engine's are a few dozen characters.
pub const URL_LIMIT: usize = 2048;
pub const COMMAND_LIMIT: usize = 1024;

// value: the System Settings panes the window opens, whole as the engine sends them: the engine's `catalog.Panes`,
// value: which a test of the architecture holds equal to this list. Nothing else reaches `open` as a pane.
pub const PANES: &[&str] = &[
    "x-apple.systempreferences:com.apple.settings.PrivacySecurity.extension?FileVault",
    "x-apple.systempreferences:com.apple.Network-Settings.extension?Firewall",
    "x-apple.systempreferences:com.apple.settings.PrivacySecurity.extension?Security",
    "x-apple.systempreferences:com.apple.settings.PrivacySecurity.extension",
    "x-apple.systempreferences:com.apple.Software-Update-Settings.extension",
    "x-apple.systempreferences:com.apple.Lock-Screen-Settings.extension",
    "x-apple.systempreferences:com.apple.Sharing-Settings.extension",
    "x-apple.systempreferences:com.apple.Time-Machine-Settings.extension",
];

const FOLDER: &str = "spwn-window-open";
const LINE_END: &str = "SPWN_LINE";
const SUFFIX: &str = ".command";

#[derive(Debug, PartialEq, Serialize)]
pub struct Refusal {
    pub code: &'static str,
    pub message: String,
}

fn refused(code: &'static str, message: impl Into<String>) -> Refusal {
    Refusal {
        code,
        message: message.into(),
    }
}

// invariant: an address opens only when it is one http or https URL written `<scheme>://<host>`, whole as sent: no
// invariant: space, no control character and no slash for a host, each of which a URL parser forgives and a browser
// invariant: would then read otherwise.
pub fn web_address(url: &str) -> Result<&str, Refusal> {
    let refuse = |why: &str| refused(NOT_A_WEB_ADDRESS, format!("the window opens no {why}"));
    if url.is_empty() || url.len() > URL_LIMIT {
        return Err(refuse(&format!(
            "address that is empty or longer than {URL_LIMIT} bytes"
        )));
    }
    if url.chars().any(|c| c.is_control() || c.is_whitespace()) {
        return Err(refuse("address holding a space or a control character"));
    }
    let parsed = tauri::Url::parse(url).map_err(|_| refuse("text that is not a web address"))?;
    if !matches!(parsed.scheme(), "http" | "https") {
        return Err(refuse(&format!("{} address", parsed.scheme())));
    }
    let written = url
        .strip_prefix(parsed.scheme())
        .and_then(|rest| rest.strip_prefix("://"))
        .and_then(|rest| rest.chars().next());
    if parsed.host_str().is_none_or(str::is_empty)
        || written.is_none_or(|first| first == '/' || first == '\\')
    {
        return Err(refuse("address that names no host"));
    }
    Ok(url)
}

// invariant: a command opens only as one line, the engine's, run verbatim: a line break, a NUL or any other control
// invariant: character would end it early or hide a second command in it.
pub fn one_line(command: &str) -> Result<&str, Refusal> {
    if command.trim().is_empty() || command.len() > COMMAND_LIMIT {
        return Err(refused(
            NOT_ONE_LINE,
            format!(
                "the window runs no command that is empty or longer than {COMMAND_LIMIT} bytes"
            ),
        ));
    }
    if command.chars().any(char::is_control) {
        return Err(refused(
            NOT_ONE_LINE,
            "the window runs no command holding a line break or a control character",
        ));
    }
    Ok(command)
}

// invariant: a pane opens only when it is, whole and byte for byte, one of PANES: a URL the engine did not name, a
// invariant: well-formed pane of another id included, never reaches `open`.
pub fn settings_pane(url: &str) -> Result<&str, Refusal> {
    if PANES.contains(&url) {
        Ok(url)
    } else {
        Err(refused(
            NOT_A_SETTINGS_PANE,
            "the window opens no System Settings pane it was not told of",
        ))
    }
}

// value: the script a terminal runs: it removes itself, then runs the engine's line as it came.
pub fn script_of(command: &str) -> String {
    format!("#!/bin/sh\nrm -f -- \"$0\"\n{command}\n")
}

// invariant: the script of a line that is the owner's to run: it prints what Spwn filled in, then waits for Return
// invariant: before it runs the line, so closing the window runs nothing; the line is printed by a quoted here-document,
// invariant: never quoted into a command.
pub fn waiting_script_of(command: &str) -> String {
    format!(
        "#!/bin/sh\nrm -f -- \"$0\"\ncat <<'{LINE_END}'\nSpwn filled in this line for you. Nothing runs until you press Return:\n\n  {command}\n\n{LINE_END}\nprintf 'Press Return to run it, or close this window. '\nread -r _answer || exit 0\n{command}\n"
    )
}

pub fn folder() -> PathBuf {
    env::temp_dir().join(FOLDER)
}

// invariant: the folder is a folder, never a link, and 0700: only its owner may change its mode, so a folder of
// invariant: another account fails here, and no other account can swap the script a terminal runs.
fn own_folder(dir: &Path) -> io::Result<()> {
    match DirBuilder::new().mode(0o700).create(dir) {
        Ok(()) => {}
        Err(error) if error.kind() == io::ErrorKind::AlreadyExists => {}
        Err(error) => return Err(error),
    }
    if !fs::symlink_metadata(dir)?.is_dir() {
        return Err(io::Error::new(
            io::ErrorKind::PermissionDenied,
            format!("{} is not a folder", dir.display()),
        ));
    }
    fs::set_permissions(dir, fs::Permissions::from_mode(0o700))
}

pub fn write_script(dir: &Path, script: &str) -> io::Result<PathBuf> {
    static NEXT: AtomicU32 = AtomicU32::new(0);
    own_folder(dir)?;
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_or(0, |since| since.as_nanos());
    let n = NEXT.fetch_add(1, Ordering::SeqCst);
    let path = dir.join(format!("open-{}-{nanos}-{n}{SUFFIX}", std::process::id()));
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o700)
        .open(&path)?;
    file.write_all(script.as_bytes())?;
    Ok(path)
}

// value: the scripts a terminal never ran, left by a window that ended before it did; removed when the window starts.
pub fn sweep(dir: &Path) -> usize {
    let Ok(entries) = fs::read_dir(dir) else {
        return 0;
    };
    entries
        .flatten()
        .filter(|entry| entry.file_name().to_string_lossy().ends_with(SUFFIX))
        .filter(|entry| fs::remove_file(entry.path()).is_ok())
        .count()
}

fn ran(program: &str, args: &[&std::ffi::OsStr]) -> Result<(), Refusal> {
    let status = Command::new(program)
        .args(args)
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .status()
        .map_err(|error| refused(NOT_OPENED, format!("{program} could not run: {error}")))?;
    if status.success() {
        Ok(())
    } else {
        Err(refused(
            NOT_OPENED,
            format!("{program} ended with {status}"),
        ))
    }
}

#[cfg(target_os = "macos")]
fn launch_url(url: &str) -> Result<(), Refusal> {
    ran("/usr/bin/open", &[url.as_ref()])
}

#[cfg(target_os = "macos")]
fn launch_script(script: &Path) -> Result<(), Refusal> {
    ran(
        "/usr/bin/open",
        &["-a".as_ref(), "Terminal".as_ref(), script.as_os_str()],
    )
}

#[cfg(target_os = "linux")]
fn launch_url(url: &str) -> Result<(), Refusal> {
    ran("xdg-open", &[url.as_ref()])
}

#[cfg(target_os = "linux")]
fn launch_script(script: &Path) -> Result<(), Refusal> {
    ran("x-terminal-emulator", &["-e".as_ref(), script.as_os_str()])
}

// hazard: a debug build alone reads STATION_WINDOW_OPENED, so a bench session can see what the window would open
// hazard: without a browser or a terminal; a release build has no such path, the function compiled out.
#[cfg(debug_assertions)]
fn faked(what: &str, value: &str) -> Option<Result<(), Refusal>> {
    let file = env::var_os("STATION_WINDOW_OPENED").filter(|path| !path.is_empty())?;
    let wrote = OpenOptions::new()
        .create(true)
        .append(true)
        .open(file)
        .and_then(|mut out| writeln!(out, "{what}\t{value}"));
    Some(wrote.map_err(|error| refused(NOT_OPENED, error.to_string())))
}

#[cfg(not(debug_assertions))]
fn faked(_: &str, _: &str) -> Option<Result<(), Refusal>> {
    None
}

pub fn open_url_now(url: &str) -> Result<(), Refusal> {
    let url = web_address(url)?;
    faked("url", url).unwrap_or_else(|| launch_url(url))
}

pub fn open_settings_now(url: &str) -> Result<(), Refusal> {
    let url = settings_pane(url)?;
    faked("settings", url).unwrap_or_else(|| launch_url(url))
}

fn run_in_terminal(what: &str, command: &str, script: &str) -> Result<(), Refusal> {
    let script = write_script(&folder(), script).map_err(|error| {
        refused(
            NOT_OPENED,
            format!("the script could not be written: {error}"),
        )
    })?;
    let shown = script.display().to_string();
    faked(what, &format!("{shown}\t{command}")).unwrap_or_else(|| {
        launch_script(&script).inspect_err(|_| {
            let _ = fs::remove_file(&script);
        })
    })
}

pub fn open_in_terminal_now(command: &str) -> Result<(), Refusal> {
    let command = one_line(command)?;
    run_in_terminal("terminal", command, &script_of(command))
}

pub fn open_fix_in_terminal_now(command: &str) -> Result<(), Refusal> {
    let command = one_line(command)?;
    run_in_terminal("fix-line", command, &waiting_script_of(command))
}

#[tauri::command(async)]
pub fn open_url(url: String) -> Result<(), Refusal> {
    open_url_now(&url)
}

#[tauri::command(async)]
pub fn open_in_terminal(command: String) -> Result<(), Refusal> {
    open_in_terminal_now(&command)
}

#[tauri::command(async)]
pub fn open_settings(url: String) -> Result<(), Refusal> {
    open_settings_now(&url)
}

#[tauri::command(async)]
pub fn open_fix_in_terminal(command: String) -> Result<(), Refusal> {
    open_fix_in_terminal_now(&command)
}

#[cfg(test)]
#[path = "opener_test.rs"]
mod tests;

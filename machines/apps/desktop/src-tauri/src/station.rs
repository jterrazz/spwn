use std::ffi::OsString;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};

include!(concat!(env!("OUT_DIR"), "/install.rs"));

const RECORD_VERSION: u64 = 1;

pub fn usable(path: &Path) -> bool {
    path.is_absolute()
        && fs::metadata(path)
            .is_ok_and(|meta| meta.is_file() && meta.permissions().mode() & 0o111 != 0)
}

pub fn under_home(dir: &str, home: &Path) -> PathBuf {
    dir.strip_prefix("~/")
        .map_or_else(|| PathBuf::from(dir), |rest| home.join(rest))
}

pub fn state_root(home: &Path, xdg_state: Option<OsString>) -> PathBuf {
    xdg_state
        .map(PathBuf::from)
        .filter(|root| root.is_absolute())
        .unwrap_or_else(|| home.join(".local/state"))
}

pub fn recorded(record: &Path) -> Option<PathBuf> {
    let text = fs::read_to_string(record).ok()?;
    let value: serde_json::Value = serde_json::from_str(&text).ok()?;
    if value.get("v").and_then(serde_json::Value::as_u64) != Some(RECORD_VERSION) {
        return None;
    }
    value
        .get("executable")
        .and_then(serde_json::Value::as_str)
        .map(PathBuf::from)
}

pub struct Where<'a> {
    pub faked: Option<&'a Path>,
    pub said: Option<&'a Path>,
    pub bundled: Option<&'a Path>,
    pub home: &'a Path,
    pub xdg_state: Option<OsString>,
}

// invariant: a window inside Spwn.app runs the engine its bundle carries before any other, the one a running engine
// invariant: names included, so the Mac stays in mode A (ADR-073).
pub fn station_of(at: Where<'_>) -> Option<PathBuf> {
    if let Some(dir) = at.faked {
        return Some(dir.join(CORE_BINARY)).filter(|path| usable(path));
    }
    let state = state_root(at.home, at.xdg_state);
    [
        at.bundled.map(Path::to_path_buf),
        at.said.map(Path::to_path_buf),
        recorded(&state.join(RECORD_PATH)),
        recorded(&state.join(INSTALLED_PATH)),
        Some(under_home(INSTALL_DIR, at.home).join(CORE_BINARY)),
    ]
    .into_iter()
    .flatten()
    .find(|path| usable(path))
}

// value: the engine Spwn.app carries beside the window's own executable
// (ADR-073, mode A); none when the window runs outside a bundle.
pub fn bundled_of(executable: &Path) -> Option<PathBuf> {
    let bundle = executable.parent()?.parent()?.parent()?;
    bundle
        .extension()
        .is_some_and(|extension| extension == "app")
        .then(|| bundle.join(BUNDLED_ENGINE))
}

// value: whether an engine's executable, as its hello names it, is one a Spwn.app carries, whichever copy.
pub fn carried(executable: &Path) -> bool {
    executable.ends_with(BUNDLED_ENGINE)
        && executable
            .ancestors()
            .nth(Path::new(BUNDLED_ENGINE).components().count())
            .and_then(Path::extension)
            .is_some_and(|extension| extension == "app")
}

#[cfg(test)]
#[path = "station_test.rs"]
mod tests;

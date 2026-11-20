use std::sync::Mutex;
use std::time::Duration;

use serde::Serialize;
use tauri::ipc::Channel;
use tauri::plugin::TauriPlugin;
use tauri::{AppHandle, Runtime, State};
use tauri_plugin_updater::{Config, Error, Update, Updater, UpdaterExt};

use crate::opener::Refusal;

// invariant: the one platform key of the window's latest.json; scripts/window-update.sh reads it from this line
pub const WINDOW_TARGET: &str = "darwin-aarch64-window";
pub const UNREACHABLE: &str = "window.update-unreachable";
pub const UNPUBLISHED: &str = "window.update-unpublished";
pub const UNSIGNED: &str = "window.update-unsigned";
pub const INSTALL: &str = "window.update-install";
pub const GONE: &str = "window.update-gone";
pub const BUSY: &str = "window.update-busy";

// invariant: a debug build and any build off macOS never reach the network for an update
pub const CHECKS: bool = cfg!(target_os = "macos") && !cfg!(debug_assertions);
const WITHIN: Duration = Duration::from_secs(20);

#[derive(Debug, PartialEq, Serialize)]
#[serde(tag = "kind", rename_all = "lowercase")]
pub enum Checked {
    Unsupported,
    Current {
        current: String,
    },
    Out {
        current: String,
        version: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        date: Option<String>,
    },
}

#[derive(Clone, Debug, PartialEq, Serialize)]
#[serde(untagged)]
pub enum Progress {
    Downloaded {
        downloaded: u64,
        #[serde(skip_serializing_if = "Option::is_none")]
        total: Option<u64>,
    },
    Installed {
        installed: bool,
    },
}

enum Held<T> {
    Empty,
    Ready(T),
    Running,
}

pub struct Slot<T> {
    held: Mutex<Held<T>>,
}

impl<T> Default for Slot<T> {
    fn default() -> Self {
        Self {
            held: Mutex::new(Held::Empty),
        }
    }
}

impl<T> Slot<T> {
    fn with<A>(&self, act: impl FnOnce(&mut Held<T>) -> Result<A, Refusal>) -> Result<A, Refusal> {
        let mut held = self
            .held
            .lock()
            .map_err(|_| refused(INSTALL, "the window lost track of its update"))?;
        act(&mut held)
    }

    pub fn keep(&self, found: Option<T>) -> Result<(), Refusal> {
        self.with(|held| match held {
            Held::Running => Err(refused(BUSY, "the window installs its update")),
            _ => {
                *held = found.map_or(Held::Empty, Held::Ready);
                Ok(())
            }
        })
    }

    pub fn begin(&self) -> Result<T, Refusal> {
        self.with(|held| match std::mem::replace(held, Held::Running) {
            Held::Ready(found) => Ok(found),
            Held::Running => Err(refused(BUSY, "the window installs its update")),
            Held::Empty => {
                *held = Held::Empty;
                Err(refused(GONE, "the window holds no update to install"))
            }
        })
    }

    pub fn end(&self, back: Option<T>) {
        let _ = self.with(|held| {
            *held = back.map_or(Held::Empty, Held::Ready);
            Ok(())
        });
    }
}

pub type Pending = Slot<Update>;

fn refused(code: &'static str, message: impl Into<String>) -> Refusal {
    Refusal {
        code,
        message: message.into(),
    }
}

pub fn code_of(error: &Error) -> &'static str {
    match error {
        Error::ReleaseNotFound
        | Error::TargetNotFound(_)
        | Error::TargetsNotFound(_)
        | Error::Serialization(_)
        | Error::Semver(_) => UNPUBLISHED,
        Error::Minisign(_)
        | Error::Base64(_)
        | Error::SignatureUtf8(_)
        | Error::SignedVersionMismatch { .. }
        | Error::MissingSignedVersion => UNSIGNED,
        Error::Io(_)
        | Error::FailedToDetermineExtractPath
        | Error::TempDirNotOnSameMountPoint
        | Error::PreviousAppNotRestored(_)
        | Error::BinaryNotFoundInArchive
        | Error::TempDirNotFound
        | Error::AuthenticationFailed
        | Error::DebInstallFailed
        | Error::PackageInstallFailed
        | Error::InvalidUpdaterFormat
        | Error::Tauri(_) => INSTALL,
        _ => UNREACHABLE,
    }
}

fn refusal(error: Error) -> Refusal {
    refused(code_of(&error), error.to_string())
}

pub fn plugin<R: Runtime>() -> TauriPlugin<R, Config> {
    tauri_plugin_updater::Builder::new()
        .target(WINDOW_TARGET)
        .build()
}

fn out_of(update: &Update) -> Checked {
    Checked::Out {
        current: update.current_version.clone(),
        version: update.version.clone(),
        date: update
            .raw_json
            .get("pub_date")
            .and_then(serde_json::Value::as_str)
            .map(str::to_owned),
    }
}

pub async fn check_with(
    updater: Result<Updater, Error>,
    current: String,
    pending: &Pending,
) -> Result<Checked, Refusal> {
    let found = updater.map_err(refusal)?.check().await.map_err(refusal)?;
    let checked = found.as_ref().map_or(Checked::Current { current }, out_of);
    pending.keep(found)?;
    Ok(checked)
}

pub async fn install_with(pending: &Pending, heard: impl Fn(Progress)) -> Result<(), Refusal> {
    let update = pending.begin()?;
    let mut downloaded = 0u64;
    let installed = update
        .download_and_install(
            |chunk, total| {
                downloaded += chunk as u64;
                heard(Progress::Downloaded { downloaded, total });
            },
            || {},
        )
        .await;
    match installed {
        Ok(()) => {
            pending.end(None);
            heard(Progress::Installed { installed: true });
            Ok(())
        }
        Err(error) => {
            pending.end(Some(update));
            Err(refusal(error))
        }
    }
}

#[tauri::command]
pub async fn window_update_check<R: Runtime>(
    app: AppHandle<R>,
    pending: State<'_, Pending>,
) -> Result<Checked, Refusal> {
    if !CHECKS {
        return Ok(Checked::Unsupported);
    }
    let current = app.package_info().version.to_string();
    let updater = app.updater_builder().timeout(WITHIN).build();
    check_with(updater, current, &pending).await
}

#[tauri::command]
pub async fn window_update_install<R: Runtime>(
    app: AppHandle<R>,
    pending: State<'_, Pending>,
    progress: Channel<Progress>,
) -> Result<(), Refusal> {
    install_with(&pending, |step| {
        let _ = progress.send(step);
    })
    .await?;
    app.restart()
}

#[cfg(test)]
#[path = "updates_test.rs"]
mod tests;

#[cfg(test)]
#[path = "updates_rehearsal_test.rs"]
mod rehearsal;

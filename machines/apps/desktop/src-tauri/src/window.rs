use std::sync::atomic::{AtomicBool, Ordering};
use std::thread;
use std::time::Duration;

use tauri::{AppHandle, Manager, State, WebviewWindow};

const MAIN: &str = "main";

// value: the shell's own cap, counted from its setup: the page shows the window at its first data or at 800 ms of its
// value: load (src/application/use-cases/launch.ts), and past this the shell shows it whatever the page did, so a page
// value: that never ran, threw before its cap, or was refused its show never leaves the window hidden.
pub const HARD_CAP: Duration = Duration::from_millis(1500);

#[derive(Default)]
pub struct Shown(AtomicBool);

impl Shown {
    pub fn claim(&self) -> bool {
        !self.0.swap(true, Ordering::SeqCst)
    }
}

pub fn capped(wait: impl FnOnce(Duration), claim: impl FnOnce() -> bool, show: impl FnOnce()) {
    wait(HARD_CAP);
    if claim() {
        show();
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Dock {
    Shown,
    Hidden,
}

// value: the Dock icon follows the window, except while the menu bar item is hidden too: then it stays, or a
// value: running Spwn would have no way back to its window
pub fn dock(window_shown: bool, item_shown: bool) -> Dock {
    if window_shown || !item_shown {
        Dock::Shown
    } else {
        Dock::Hidden
    }
}

#[cfg(target_os = "macos")]
fn follow(app: &AppHandle, window_shown: bool) {
    let policy = match dock(window_shown, crate::menu_bar::panel().shown()) {
        Dock::Shown => tauri::ActivationPolicy::Regular,
        Dock::Hidden => tauri::ActivationPolicy::Accessory,
    };
    let _ = app.set_activation_policy(policy);
}

#[cfg(not(target_os = "macos"))]
fn follow(_app: &AppHandle, _window_shown: bool) {}

fn reveal(window: &WebviewWindow) {
    follow(window.app_handle(), true);
    let _ = window.show();
}

// invariant: closing Spwn's window hides it, the page and its state kept, so the next show draws it unchanged and
// invariant: the process lives on with its menu bar
#[cfg(target_os = "macos")]
pub fn close_requested(window: &tauri::Window, api: &tauri::CloseRequestApi) {
    if window.label() != MAIN {
        return;
    }
    api.prevent_close();
    let _ = window.hide();
    follow(window.app_handle(), false);
}

#[tauri::command]
pub fn window_ready(window: WebviewWindow, shown: State<'_, Shown>) {
    if shown.claim() {
        reveal(&window);
    }
}

pub fn backstop(app: AppHandle) {
    thread::spawn(move || {
        capped(
            thread::sleep,
            || app.state::<Shown>().claim(),
            || {
                if let Some(window) = app.get_webview_window(MAIN) {
                    reveal(&window);
                }
            },
        );
    });
}

// value: the Dock's click on an app with no window on screen, the window still hidden or minimized among them.
#[cfg(target_os = "macos")]
pub fn reopened(app: &AppHandle) {
    app.state::<Shown>().claim();
    if let Some(window) = app.get_webview_window(MAIN) {
        let _ = window.unminimize();
        reveal(&window);
        let _ = window.set_focus();
    }
}

#[cfg(test)]
#[path = "window_test.rs"]
mod tests;

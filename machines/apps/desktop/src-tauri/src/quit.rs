use tauri::{AppHandle, Runtime};

// invariant: the id of the app menu's own "Quit Spwn", on Cmd-Q, which reaches `requested` instead of AppKit's
// invariant: terminate:, so every quit takes this one route
pub const MENU_ID: &str = "spwn-quit";

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Reason {
    User,
    Logout,
    Restart,
    Shutdown,
}

impl Reason {
    // value: the reason as the menu bar's C entry points number it, any other number read as the owner's own quit
    pub fn of(code: i32) -> Self {
        match code {
            1 => Self::Logout,
            2 => Self::Restart,
            3 => Self::Shutdown,
            _ => Self::User,
        }
    }

    pub fn word(self) -> &'static str {
        match self {
            Self::User => "user",
            Self::Logout => "logout",
            Self::Restart => "restart",
            Self::Shutdown => "shutdown",
        }
    }
}

pub fn requested<R: Runtime>(app: &AppHandle<R>, reason: Reason) {
    eprintln!("spwn-window: quit ({})", reason.word());
    app.exit(0);
}

#[cfg(test)]
#[path = "quit_test.rs"]
mod tests;

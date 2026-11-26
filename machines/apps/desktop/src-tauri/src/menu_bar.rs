use serde::Serialize;

use crate::opener::Refusal;

// invariant: the bundle id of the menu bar Spwn.app nests at Contents/Library/LoginItems/Station.app; `make window`
// invariant: checks the nested bundle carries it
pub const MENU_BAR: &str = "com.jterrazz.station.menubar";
pub const NOT_SWITCHED: &str = "menubar.not-switched";

// invariant: a debug build and any build off macOS never touch a login item
pub const SWITCHES: bool = cfg!(target_os = "macos") && !cfg!(debug_assertions);

#[derive(Clone, Copy, Debug, PartialEq)]
pub enum Status {
    Enabled,
    NotRegistered,
    RequiresApproval,
    NotFound,
}

#[derive(Debug, PartialEq, Serialize)]
#[serde(tag = "kind", rename_all = "lowercase")]
pub enum Read {
    Shown,
    Hidden,
    Approval,
    Unsupported,
}

pub trait LoginItem {
    fn status(&self) -> Status;
    fn register(&self) -> Result<(), String>;
    fn unregister(&self) -> Result<(), String>;
}

pub fn read(item: &dyn LoginItem) -> Read {
    match item.status() {
        Status::Enabled => Read::Shown,
        Status::NotRegistered => Read::Hidden,
        Status::RequiresApproval => Read::Approval,
        Status::NotFound => Read::Unsupported,
    }
}

// value: the switch's press: the login item registered or unregistered, then its status read back, so the window
// value: draws what macOS holds; one already as asked is left alone.
pub fn show(item: &dyn LoginItem, shown: bool) -> Result<Read, Refusal> {
    let held = read(item);
    if held == Read::Unsupported || (held == Read::Hidden) != shown {
        return Ok(held);
    }
    let done = if shown {
        item.register()
    } else {
        item.unregister()
    };
    done.map_err(|message| Refusal {
        code: NOT_SWITCHED,
        message,
    })?;
    Ok(read(item))
}

#[cfg(target_os = "macos")]
mod service {
    use objc2_foundation::NSString;
    use objc2_service_management::{SMAppService, SMAppServiceStatus};

    use super::{LoginItem, Status, MENU_BAR};

    pub struct Nested;

    fn service() -> objc2::rc::Retained<SMAppService> {
        unsafe { SMAppService::loginItemServiceWithIdentifier(&NSString::from_str(MENU_BAR)) }
    }

    fn said(error: &objc2_foundation::NSError) -> String {
        format!(
            "{} {}: {}",
            error.domain(),
            error.code(),
            error.localizedDescription()
        )
    }

    impl LoginItem for Nested {
        fn status(&self) -> Status {
            match unsafe { service().status() } {
                SMAppServiceStatus::Enabled => Status::Enabled,
                SMAppServiceStatus::RequiresApproval => Status::RequiresApproval,
                SMAppServiceStatus::NotRegistered => Status::NotRegistered,
                _ => Status::NotFound,
            }
        }

        fn register(&self) -> Result<(), String> {
            unsafe { service().registerAndReturnError() }.map_err(|error| said(&error))
        }

        fn unregister(&self) -> Result<(), String> {
            unsafe { service().unregisterAndReturnError() }.map_err(|error| said(&error))
        }
    }
}

// invariant: the menu bar item Spwn's own process draws, when it is built with the `menu-bar` feature; a build
// invariant: without it draws none, and its item reads hidden
pub trait Panel {
    fn shown(&self) -> bool;
}

#[cfg(all(target_os = "macos", feature = "menu-bar"))]
mod native {
    use std::ffi::{c_char, c_void};
    use std::sync::OnceLock;

    use tauri::AppHandle;

    use super::Panel;
    use crate::quit::{self, Reason};
    use crate::window;

    static APP: OnceLock<AppHandle> = OnceLock::new();

    // invariant: the C entry points of libSpwnMenuBar.dylib (apps/menubar/sources/spwn-menu-bar/Shell.swift), its
    // invariant: one Rust-side owner
    extern "C" {
        fn spwn_menu_bar_start(
            ctx: *mut c_void,
            open_window: extern "C" fn(*mut c_void, *const c_char),
            quit: extern "C" fn(*mut c_void, i32),
        );
        fn spwn_menu_bar_shown() -> bool;
    }

    extern "C" fn open_window(_ctx: *mut c_void, _machine: *const c_char) {
        if let Some(app) = APP.get() {
            window::reopened(app);
        }
    }

    extern "C" fn quit(_ctx: *mut c_void, reason: i32) {
        if let Some(app) = APP.get() {
            quit::requested(app, Reason::of(reason));
        }
    }

    pub struct Native;

    impl Native {
        pub fn start(app: AppHandle) {
            if APP.set(app).is_ok() {
                unsafe { spwn_menu_bar_start(std::ptr::null_mut(), open_window, quit) }
            }
        }
    }

    impl Panel for Native {
        fn shown(&self) -> bool {
            unsafe { spwn_menu_bar_shown() }
        }
    }
}

#[cfg(all(target_os = "macos", feature = "menu-bar"))]
pub fn panel() -> Box<dyn Panel> {
    Box::new(native::Native)
}

#[cfg(not(all(target_os = "macos", feature = "menu-bar")))]
pub fn panel() -> Box<dyn Panel> {
    Box::new(Absent)
}

#[cfg(all(target_os = "macos", feature = "menu-bar"))]
pub fn start(app: &tauri::AppHandle) {
    native::Native::start(app.clone());
}

#[cfg(not(all(target_os = "macos", feature = "menu-bar")))]
pub fn start(_app: &tauri::AppHandle) {}

struct Absent;

impl Panel for Absent {
    fn shown(&self) -> bool {
        false
    }
}

impl LoginItem for Absent {
    fn status(&self) -> Status {
        Status::NotFound
    }

    fn register(&self) -> Result<(), String> {
        Err(String::from("this window has no menu bar to switch"))
    }

    fn unregister(&self) -> Result<(), String> {
        Err(String::from("this window has no menu bar to switch"))
    }
}

#[cfg(target_os = "macos")]
fn nested() -> Box<dyn LoginItem> {
    if SWITCHES {
        Box::new(service::Nested)
    } else {
        Box::new(Absent)
    }
}

#[cfg(not(target_os = "macos"))]
fn nested() -> Box<dyn LoginItem> {
    Box::new(Absent)
}

#[tauri::command(async)]
pub fn menu_bar_read() -> Read {
    read(nested().as_ref())
}

#[tauri::command(async)]
pub fn menu_bar_show(shown: bool) -> Result<Read, Refusal> {
    show(nested().as_ref(), shown)
}

#[cfg(test)]
#[path = "menu_bar_test.rs"]
mod tests;

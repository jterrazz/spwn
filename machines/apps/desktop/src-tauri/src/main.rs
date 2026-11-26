#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod bridge;
mod engine;
mod local_core;
mod menu_bar;
mod opener;
mod quit;
mod station;
mod updates;
mod window;

use std::sync::Arc;

use tauri::webview::PageLoadEvent;
use tauri::{AppHandle, RunEvent, WindowEvent};

// invariant: on a Mac, Spwn ends only by a quit it is asked: closing its last window never ends the run loop
#[cfg(target_os = "macos")]
fn on_run(app: &AppHandle, event: RunEvent) {
    match event {
        RunEvent::Reopen {
            has_visible_windows: false,
            ..
        } => window::reopened(app),
        RunEvent::ExitRequested {
            code: None, api, ..
        } => api.prevent_exit(),
        _ => {}
    }
}

#[cfg(not(target_os = "macos"))]
fn on_run(_app: &AppHandle, _event: RunEvent) {}

// value: the default app menu, its app submenu ending in Spwn's own "Quit Spwn" on Cmd-Q
#[cfg(target_os = "macos")]
fn app_menu(app: &AppHandle) -> tauri::Result<tauri::menu::Menu<tauri::Wry>> {
    use tauri::menu::{Menu, MenuItem, PredefinedMenuItem, Submenu};

    let menu = Menu::default(app)?;
    let own = Submenu::with_items(
        app,
        app.package_info().name.clone(),
        true,
        &[
            &PredefinedMenuItem::about(app, None, None)?,
            &PredefinedMenuItem::separator(app)?,
            &PredefinedMenuItem::services(app, None)?,
            &PredefinedMenuItem::separator(app)?,
            &PredefinedMenuItem::hide(app, None)?,
            &PredefinedMenuItem::hide_others(app, None)?,
            &PredefinedMenuItem::separator(app)?,
            &MenuItem::with_id(app, quit::MENU_ID, "Quit Spwn", true, Some("CmdOrCtrl+Q"))?,
        ],
    )?;
    menu.remove_at(0)?;
    menu.prepend(&own)?;
    Ok(menu)
}

#[cfg(target_os = "macos")]
fn with_menu(builder: tauri::Builder<tauri::Wry>) -> tauri::Builder<tauri::Wry> {
    builder.menu(app_menu).on_menu_event(|app, event| {
        if event.id().as_ref() == quit::MENU_ID {
            quit::requested(app, quit::Reason::User);
        }
    })
}

#[cfg(not(target_os = "macos"))]
fn with_menu(builder: tauri::Builder<tauri::Wry>) -> tauri::Builder<tauri::Wry> {
    builder
}

fn main() {
    opener::sweep(&opener::folder());
    let bridge = engine::Bridge::new();
    let runs = Arc::new(local_core::Runs::default());
    let loading = bridge.exchanges.clone();
    let closing = bridge.exchanges.clone();
    with_menu(tauri::Builder::default())
        .plugin(updates::plugin())
        .manage(bridge.clone())
        .manage(Arc::clone(&runs))
        .manage(updates::Pending::default())
        .manage(window::Shown::default())
        .invoke_handler(tauri::generate_handler![
            engine::engine_link,
            engine::engine_open,
            engine::engine_cancel,
            engine::engine_close,
            opener::open_url,
            opener::open_in_terminal,
            local_core::core_present,
            local_core::core_install,
            local_core::core_start,
            local_core::core_cancel,
            updates::window_update_check,
            updates::window_update_install,
            menu_bar::menu_bar_read,
            menu_bar::menu_bar_show,
            window::window_ready
        ])
        .on_page_load(move |webview, payload| {
            if payload.event() == PageLoadEvent::Started {
                loading.close_owner(webview.label());
            }
        })
        .on_window_event(move |window, event| match event {
            WindowEvent::Destroyed => {
                closing.close_owner(window.label());
            }
            #[cfg(target_os = "macos")]
            WindowEvent::CloseRequested { api, .. } => window::close_requested(window, api),
            _ => {}
        })
        .setup(move |app| {
            menu_bar::start(app.handle());
            window::backstop(app.handle().clone());
            engine::supervise(app.handle().clone(), bridge, runs);
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("the window could not start")
        .run(on_run);
}

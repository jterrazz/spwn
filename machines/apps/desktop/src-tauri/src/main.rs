#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod bridge;
mod engine;
mod local_core;
mod opener;
mod station;
mod updates;

use std::sync::Arc;

use tauri::webview::PageLoadEvent;
use tauri::WindowEvent;

fn main() {
    opener::sweep(&opener::folder());
    let bridge = engine::Bridge::new();
    let loading = bridge.exchanges.clone();
    let closing = bridge.exchanges.clone();
    tauri::Builder::default()
        .plugin(updates::plugin())
        .manage(bridge.clone())
        .manage(Arc::new(local_core::Runs::default()))
        .manage(updates::Pending::default())
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
            updates::window_update_install
        ])
        .on_page_load(move |webview, payload| {
            if payload.event() == PageLoadEvent::Started {
                loading.close_owner(webview.label());
            }
        })
        .on_window_event(move |window, event| {
            if matches!(event, WindowEvent::Destroyed) {
                closing.close_owner(window.label());
            }
        })
        .setup(move |app| {
            engine::supervise(app.handle().clone(), bridge);
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("the window could not start");
}

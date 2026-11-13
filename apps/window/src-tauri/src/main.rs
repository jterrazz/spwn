#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod bridge;
mod engine;

use tauri::webview::PageLoadEvent;
use tauri::WindowEvent;

fn main() {
    let bridge = engine::Bridge::new();
    let loading = bridge.exchanges.clone();
    let closing = bridge.exchanges.clone();
    tauri::Builder::default()
        .manage(bridge.clone())
        .invoke_handler(tauri::generate_handler![
            engine::engine_link,
            engine::engine_open,
            engine::engine_close
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

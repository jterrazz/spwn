use std::process::{Command, Stdio};
use std::sync::Mutex;
use tauri::Manager;

struct ApiProcess(Mutex<Option<std::process::Child>>);

#[tauri::command]
fn get_api_port(state: tauri::State<'_, ApiPort>) -> u16 {
    state.0
}

struct ApiPort(u16);

pub fn run() {
    // Pick a random available port for the Go API
    let api_port = portpicker::pick_unused_port().expect("No free port available");

    // Find spwn binary — check PATH first, then common locations
    let spwn_bin = which_spwn();

    // Start the Go API server as a child process
    let child = Command::new(&spwn_bin)
        .args(["dash", "start", "--port", &api_port.to_string()])
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn();

    let api_child = match child {
        Ok(c) => {
            println!("[spwn] API server starting on port {api_port}");
            Some(c)
        }
        Err(e) => {
            eprintln!("[spwn] Failed to start API server: {e}");
            eprintln!("[spwn] Looked for binary at: {spwn_bin}");
            eprintln!("[spwn] Install spwn first: https://spwn.sh");
            None
        }
    };

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_process::init())
        .manage(ApiPort(api_port))
        .manage(ApiProcess(Mutex::new(api_child)))
        .invoke_handler(tauri::generate_handler![get_api_port])
        .setup(move |app| {
            // Give the API server a moment to start
            std::thread::sleep(std::time::Duration::from_secs(2));

            // Set the window URL to include the API port
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.set_title(&format!("⬡ spwn Observatory — API on port {api_port}"));
            }

            println!("[spwn] Observatory ready");
            println!("[spwn] API: http://localhost:{api_port}");
            Ok(())
        })
        .on_window_event(|window, event| {
            // Gracefully stop the API server when the window closes
            if let tauri::WindowEvent::Destroyed = event {
                if let Some(state) = window.try_state::<ApiProcess>() {
                    if let Ok(mut guard) = state.0.lock() {
                        if let Some(ref mut child) = *guard {
                            println!("[spwn] Stopping API server...");
                            let _ = child.kill();
                            let _ = child.wait();
                            println!("[spwn] API server stopped");
                        }
                    }
                }
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

fn which_spwn() -> String {
    // Check PATH
    if let Ok(output) = Command::new("which").arg("spwn").output() {
        if output.status.success() {
            let path = String::from_utf8_lossy(&output.stdout).trim().to_string();
            if !path.is_empty() {
                return path;
            }
        }
    }

    // Common locations
    let home = std::env::var("HOME").unwrap_or_default();
    let candidates = [
        format!("{home}/.local/bin/spwn"),
        "/usr/local/bin/spwn".to_string(),
        format!("{home}/go/bin/spwn"),
    ];

    for c in &candidates {
        if std::path::Path::new(c).exists() {
            return c.clone();
        }
    }

    // Fallback — hope it's in PATH
    "spwn".to_string()
}

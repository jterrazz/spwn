use std::process::{Command, Stdio};
use std::sync::Mutex;
use tauri::Manager;

struct ApiProcess(Mutex<Option<std::process::Child>>);

#[tauri::command]
fn get_api_port(state: tauri::State<'_, ApiPort>) -> u16 {
    state.0
}

struct ApiPort(u16);

// Traffic light position inside the sidebar glass panel
#[cfg(target_os = "macos")]
const TRAFFIC_LIGHT_X: f64 = 24.0;
#[cfg(target_os = "macos")]
const TRAFFIC_LIGHT_Y: f64 = 42.0;

#[cfg(target_os = "macos")]
fn position_traffic_lights(ns_window: cocoa::base::id) {
    use cocoa::appkit::{NSView, NSWindow, NSWindowButton};
    use cocoa::foundation::NSRect;
    use objc::{msg_send, sel, sel_impl};

    unsafe {
        let close = ns_window.standardWindowButton_(NSWindowButton::NSWindowCloseButton);
        let miniaturize = ns_window.standardWindowButton_(NSWindowButton::NSWindowMiniaturizeButton);
        let zoom = ns_window.standardWindowButton_(NSWindowButton::NSWindowZoomButton);

        let title_bar_container_view = close.superview().superview();

        let close_rect: NSRect = msg_send![close, frame];
        let button_height = close_rect.size.height;

        let title_bar_frame_height = button_height + TRAFFIC_LIGHT_Y;
        let mut title_bar_rect = NSView::frame(title_bar_container_view);
        title_bar_rect.size.height = title_bar_frame_height;
        title_bar_rect.origin.y = NSView::frame(ns_window).size.height - title_bar_frame_height;
        let _: () = msg_send![title_bar_container_view, setFrame: title_bar_rect];

        let buttons = [close, miniaturize, zoom];
        let space_between = 20.0;

        for (i, button) in buttons.iter().enumerate() {
            let mut rect: NSRect = NSView::frame(*button);
            rect.origin.x = TRAFFIC_LIGHT_X + (i as f64 * space_between);
            rect.origin.y = (title_bar_frame_height - button_height) / 2.0;
            button.setFrameOrigin(rect.origin);
        }
    }
}

pub fn run() {
    // Pick a random available port for the Go API
    let api_port = portpicker::pick_unused_port().expect("No free port available");

    // On macOS, GUI apps launched by Finder / launchd inherit a
    // sanitized PATH ("/usr/bin:/bin:/usr/sbin:/sbin") that does NOT
    // include /opt/homebrew/bin or /usr/local/bin where Docker Desktop
    // and Homebrew install their binaries. Augment PATH before we
    // spawn the spwn child so every `exec.Command("docker", ...)` call
    // inside the Go process resolves correctly. The Go side also
    // re-runs the same augmentation at process start (defense in
    // depth — if a user launches spwn directly via Spotlight, the
    // GUI-launched spwn binary still fixes its own PATH).
    let augmented_path = augmented_path();

    // Find spwn binary — check PATH first, then common locations
    let spwn_bin = which_spwn(&augmented_path);

    // Run migrations before starting the API server.
    // This ensures the user's ~/.spwn data is up-to-date even if they
    // haven't used the CLI directly since updating the app.
    match Command::new(&spwn_bin)
        .env("PATH", &augmented_path)
        .args(["version"])
        .output()
    {
        Ok(output) if output.status.success() => {
            println!("[spwn] Migrations applied (via CLI pre-run hook)");
        }
        Ok(output) => {
            let stderr = String::from_utf8_lossy(&output.stderr);
            eprintln!("[spwn] Migration pre-flight warning: {stderr}");
        }
        Err(e) => {
            eprintln!("[spwn] Could not run migrations: {e}");
        }
    }

    // Start the Go API server as a child process
    let child = Command::new(&spwn_bin)
        .env("PATH", &augmented_path)
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
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_dialog::init())
        .manage(ApiPort(api_port))
        .manage(ApiProcess(Mutex::new(api_child)))
        .invoke_handler(tauri::generate_handler![get_api_port])
        .setup(move |app| {
            // Give the API server a moment to start
            std::thread::sleep(std::time::Duration::from_secs(2));

            if let Some(window) = app.get_webview_window("main") {
                // Add tauri class to html element for native app styling
                let _ = window.eval("document.documentElement.classList.add('tauri')");

                // Position traffic lights AFTER everything else (set_title resets them)
                #[cfg(target_os = "macos")]
                {
                    let win = window.clone();
                    std::thread::spawn(move || {
                        std::thread::sleep(std::time::Duration::from_millis(500));
                        let win2 = win.clone();
                        let _ = win.run_on_main_thread(move || {
                            let ns_win = win2.ns_window().expect("Failed to get NS window") as cocoa::base::id;
                            position_traffic_lights(ns_win);
                        });
                    });
                }
            }

            println!("[spwn] Observatory ready");
            println!("[spwn] API: http://localhost:{api_port}");
            Ok(())
        })
        .on_window_event(|window, event| {
            match event {
                // Reapply traffic light position on resize, move, and focus changes
                // (moving between screens with different DPI resets them)
                #[cfg(target_os = "macos")]
                tauri::WindowEvent::Resized(..)
                | tauri::WindowEvent::Moved(..)
                | tauri::WindowEvent::Focused(true) => {
                    let ns_win = window.ns_window().expect("Failed to get NS window") as cocoa::base::id;
                    position_traffic_lights(ns_win);
                }
                // Gracefully stop the API server when the window closes
                tauri::WindowEvent::Destroyed => {
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
                _ => {}
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

/// Return PATH with well-known Homebrew / Docker Desktop locations
/// prepended, de-duplicated. Matches the Go-side
/// `foundation.EnsureDockerFriendlyPATH` — keep in sync.
fn augmented_path() -> String {
    let current = std::env::var("PATH").unwrap_or_default();
    let extras: &[&str] = if cfg!(target_os = "macos") {
        &[
            "/opt/homebrew/bin",
            "/opt/homebrew/sbin",
            "/usr/local/bin",
            "/usr/local/sbin",
            "/Applications/Docker.app/Contents/Resources/bin",
        ]
    } else if cfg!(target_os = "linux") {
        &["/usr/local/bin", "/usr/local/sbin", "/snap/bin"]
    } else {
        &[]
    };

    let mut seen: std::collections::HashSet<&str> = std::collections::HashSet::new();
    let mut out: Vec<String> = Vec::new();
    for extra in extras {
        if seen.insert(*extra) {
            out.push((*extra).to_string());
        }
    }
    for p in current.split(':') {
        if !p.is_empty() && seen.insert(p) {
            out.push(p.to_string());
        }
    }
    out.join(":")
}

fn which_spwn(augmented_path: &str) -> String {
    // Check PATH using the augmented PATH so /usr/local/bin/spwn and
    // ~/.local/bin/spwn are actually reachable by `which` when the
    // GUI-inherited PATH doesn't include them.
    if let Ok(output) = Command::new("which")
        .arg("spwn")
        .env("PATH", augmented_path)
        .output()
    {
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
        "/opt/homebrew/bin/spwn".to_string(),
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

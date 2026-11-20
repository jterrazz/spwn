use std::collections::HashMap;
use std::fs;
use std::io::{BufRead, BufReader, Write};
use std::net::{TcpListener, TcpStream};
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};
use std::thread;

use serde_json::{json, Value};
use tauri::test::MockRuntime;
use tauri::{App, Url};
use tauri_plugin_updater::UpdaterExt;

use super::{
    check_with, install_with, plugin, Checked, Pending, Progress, UNPUBLISHED, UNSIGNED,
    WINDOW_TARGET,
};

// value: the folder holds A/Spwn.app, the installed window at INSTALLED, then the release's Spwn.app.tar.gz and
// value: latest.json, and pubkey: the public half of the key that signed them, as plugins.updater.pubkey carries it
const FOLDER: &str = "SPWN_WINDOW_REHEARSAL";
const INSTALLED: &str = "0.0.1-rehearsal.1";

struct Rehearsal {
    installed: PathBuf,
    version: String,
    origin: String,
    app: App<MockRuntime>,
}

fn read(dir: &Path, name: &str) -> Vec<u8> {
    fs::read(dir.join(name)).unwrap_or_else(|error| panic!("{name} is unreadable: {error}"))
}

fn pointed(manifest: &Value, target: &str, url: String) -> Vec<u8> {
    let mut platform = manifest["platforms"][WINDOW_TARGET].clone();
    platform["url"] = json!(url);
    let mut manifest = manifest.clone();
    manifest["platforms"] = json!({ target: platform });
    serde_json::to_vec(&manifest).expect("the manifest serializes")
}

fn answer(stream: &TcpStream, files: &HashMap<String, Vec<u8>>) {
    let mut lines = BufReader::new(stream);
    let mut request = String::new();
    let _ = lines.read_line(&mut request);
    let mut header = String::new();
    while lines.read_line(&mut header).is_ok_and(|read| read > 0) && !header.trim().is_empty() {
        header.clear();
    }
    let path = request.split(' ').nth(1).unwrap_or_default();
    let (status, body) = files
        .get(path)
        .map_or(("404 Not Found", &[][..]), |body| ("200 OK", &body[..]));
    let head = format!(
        "HTTP/1.1 {status}\r\nContent-Length: {}\r\nConnection: close\r\n\r\n",
        body.len()
    );
    let mut stream = stream;
    let _ = stream.write_all(head.as_bytes());
    let _ = stream.write_all(body);
}

fn served(dir: &Path, manifest: &Value) -> String {
    let listener = TcpListener::bind("127.0.0.1:0").expect("a local port is free");
    let origin = format!(
        "http://{}",
        listener.local_addr().expect("the port is known")
    );
    let archive = read(dir, "Spwn.app.tar.gz");
    let mut tampered = archive.clone();
    if let Some(last) = tampered.last_mut() {
        *last ^= 0xff;
    }
    let good = format!("{origin}/Spwn.app.tar.gz");
    let files = HashMap::from([
        (
            String::from("/good.json"),
            pointed(manifest, WINDOW_TARGET, good.clone()),
        ),
        (
            String::from("/tampered.json"),
            pointed(manifest, WINDOW_TARGET, format!("{origin}/tampered.tar.gz")),
        ),
        (
            String::from("/foreign.json"),
            pointed(manifest, "darwin-aarch64", good),
        ),
        (String::from("/Spwn.app.tar.gz"), archive),
        (String::from("/tampered.tar.gz"), tampered),
    ]);
    thread::spawn(move || {
        for stream in listener.incoming().flatten() {
            answer(&stream, &files);
        }
    });
    origin
}

fn mock(pubkey: &str) -> App<MockRuntime> {
    let mut context = tauri::test::mock_context(tauri::test::noop_assets());
    context.config_mut().plugins.0.insert(
        String::from("updater"),
        json!({
            "pubkey": pubkey,
            "requireSignedVersion": true,
            "dangerousInsecureTransportProtocol": true,
        }),
    );
    context.package_info_mut().version = INSTALLED.parse().expect("INSTALLED is semver");
    tauri::test::mock_builder()
        .plugin(plugin())
        .build(context)
        .expect("the mock app builds with the window's updater")
}

fn rehearsal() -> Rehearsal {
    let dir = std::env::var_os(FOLDER)
        .map(PathBuf::from)
        .unwrap_or_else(|| panic!("{FOLDER} names the rehearsal's folder"));
    let manifest: Value =
        serde_json::from_slice(&read(&dir, "latest.json")).expect("latest.json is JSON");
    let pubkey = String::from_utf8(read(&dir, "pubkey")).expect("pubkey is text");
    Rehearsal {
        installed: dir.join("A/Spwn.app"),
        version: manifest["version"].as_str().unwrap_or_default().to_owned(),
        origin: served(&dir, &manifest),
        app: mock(pubkey.trim()),
    }
}

impl Rehearsal {
    fn checked(&self, manifest: &str, pending: &Pending) -> Result<Checked, &'static str> {
        let endpoint = Url::parse(&format!("{}/{manifest}", self.origin)).expect("a URL");
        let binary = self.installed.join("Contents/MacOS/spwn-window");
        let updater = self
            .app
            .updater_builder()
            .endpoints(vec![endpoint])
            .and_then(|builder| builder.executable_path(binary).build());
        tauri::async_runtime::block_on(check_with(updater, String::from(INSTALLED), pending))
            .map_err(|refusal| refusal.code)
    }

    fn out(&self) -> Result<Checked, &'static str> {
        Ok(Checked::Out {
            current: String::from(INSTALLED),
            version: self.version.clone(),
            date: None,
        })
    }

    fn plist(&self) -> Vec<u8> {
        read(&self.installed, "Contents/Info.plist")
    }
}

fn without_date(checked: Result<Checked, &'static str>) -> Result<Checked, &'static str> {
    checked.map(|checked| match checked {
        Checked::Out {
            current, version, ..
        } => Checked::Out {
            current,
            version,
            date: None,
        },
        other => other,
    })
}

fn installed(pending: &Pending) -> (Result<(), &'static str>, Vec<Progress>) {
    let heard = Arc::new(Mutex::new(Vec::new()));
    let kept = Arc::clone(&heard);
    let result = tauri::async_runtime::block_on(install_with(pending, move |step| {
        if let Ok(mut steps) = kept.lock() {
            steps.push(step);
        }
    }))
    .map_err(|refusal| refusal.code);
    let steps = heard.lock().map(|steps| steps.clone()).unwrap_or_default();
    (result, steps)
}

#[test]
#[ignore = "a rehearsal over SPWN_WINDOW_REHEARSAL's folder"]
fn rehearsal_a_tampered_archive_is_refused_as_unsigned_and_the_installed_app_stays() {
    let rehearsal = rehearsal();
    let before = rehearsal.plist();
    let pending = Pending::default();
    assert_eq!(
        without_date(rehearsal.checked("tampered.json", &pending)),
        rehearsal.out()
    );
    assert_eq!(installed(&pending).0, Err(UNSIGNED));
    assert_eq!(rehearsal.plist(), before, "the installed app is untouched");
}

#[test]
#[ignore = "a rehearsal over SPWN_WINDOW_REHEARSAL's folder"]
fn rehearsal_a_release_for_the_old_desktop_apps_alone_is_refused_as_unpublished() {
    let rehearsal = rehearsal();
    let pending = Pending::default();
    assert_eq!(
        rehearsal.checked("foreign.json", &pending),
        Err(UNPUBLISHED)
    );
}

#[cfg(target_os = "macos")]
#[test]
#[ignore = "a rehearsal over SPWN_WINDOW_REHEARSAL's folder"]
fn rehearsal_the_installed_app_is_swapped_for_the_release_in_place() {
    use std::process::Command;
    let rehearsal = rehearsal();
    let pending = Pending::default();
    assert_eq!(
        without_date(rehearsal.checked("good.json", &pending)),
        rehearsal.out()
    );
    let (result, steps) = installed(&pending);
    assert_eq!(result, Ok(()));
    assert_eq!(steps.last(), Some(&Progress::Installed { installed: true }));
    let plist = rehearsal.installed.join("Contents/Info.plist");
    let shown = Command::new("plutil")
        .args(["-extract", "CFBundleShortVersionString", "raw"])
        .arg(&plist)
        .output()
        .expect("plutil runs");
    assert_eq!(
        String::from_utf8_lossy(&shown.stdout).trim(),
        rehearsal.version
    );
    let verified = Command::new("codesign")
        .args(["--verify", "--strict"])
        .arg(&rehearsal.installed)
        .status()
        .expect("codesign runs");
    assert!(verified.success(), "the swapped app verifies strict");
}

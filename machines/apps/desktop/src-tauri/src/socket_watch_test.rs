use std::fs;
use std::os::unix::net::UnixListener;
use std::path::PathBuf;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::mpsc::{self, Receiver};
use std::thread;
use std::time::Duration;

use super::watch;

const SOON: Duration = Duration::from_secs(2);
const QUIET: Duration = Duration::from_millis(200);

fn folder() -> PathBuf {
    static NEXT: AtomicUsize = AtomicUsize::new(0);
    let n = NEXT.fetch_add(1, Ordering::SeqCst);
    let folder =
        std::env::temp_dir().join(format!("window-socket-watch-{}-{n}", std::process::id()));
    let _ = fs::remove_dir_all(&folder);
    fs::create_dir_all(&folder).expect("the folder is made");
    folder
}

// value: a socket bound at `engine.sock` in a folder of its own, watched; each change the watch tells is a message.
fn watched() -> (PathBuf, UnixListener, Receiver<()>, Receiver<()>) {
    let folder = folder();
    let path = folder.join("engine.sock");
    let listener = UnixListener::bind(&path).expect("the engine's socket binds");
    let (told, changes) = mpsc::channel();
    let handle = watch(&path, move || {
        let _ = told.send(());
    })
    .expect("the folder is watched");
    let (gone, ended) = mpsc::channel();
    thread::spawn(move || {
        let _ = handle.join();
        let _ = gone.send(());
    });
    (path, listener, changes, ended)
}

#[test]
fn a_socket_bound_anew_at_the_path_is_told() {
    // Given - an engine's socket, watched
    let (path, listener, changes, _) = watched();

    // When - another engine takes its place
    drop(listener);
    fs::remove_file(&path).expect("the old socket goes");
    let _next = UnixListener::bind(&path).expect("the new socket binds");

    // Then - the change is told
    assert!(changes.recv_timeout(SOON).is_ok());
    let _ = fs::remove_dir_all(path.parent().expect("a folder"));
}

#[test]
fn a_socket_removed_from_the_path_is_told() {
    // Given - an engine's socket, watched
    let (path, listener, changes, _) = watched();

    // When - the engine goes and takes its socket with it
    drop(listener);
    fs::remove_file(&path).expect("the socket goes");

    // Then - the change is told
    assert!(changes.recv_timeout(SOON).is_ok());
    let _ = fs::remove_dir_all(path.parent().expect("a folder"));
}

#[test]
fn a_file_written_beside_the_socket_is_never_told() {
    // Given - an engine's socket, watched
    let (path, _listener, changes, _) = watched();

    // When - another file of its folder is written and removed
    let beside = path.with_file_name("engine.json");
    fs::write(&beside, "{}").expect("the file is written");
    fs::remove_file(&beside).expect("the file is removed");

    // Then - nothing is told
    assert!(changes.recv_timeout(QUIET).is_err());
    let _ = fs::remove_dir_all(path.parent().expect("a folder"));
}

#[test]
fn the_watch_ends_when_its_folder_goes() {
    // Given - an engine's socket, watched
    let (path, listener, _, ended) = watched();

    // When - its whole folder is removed
    drop(listener);
    fs::remove_dir_all(path.parent().expect("a folder")).expect("the folder goes");

    // Then - the watch's thread ends
    assert!(ended.recv_timeout(SOON).is_ok());
}

#[test]
fn a_path_whose_folder_does_not_exist_is_not_watched() {
    // Given - a path in no folder
    let path = std::env::temp_dir().join(format!(
        "window-socket-watch-none-{}/engine.sock",
        std::process::id()
    ));

    // When - it is watched
    let watched = watch(&path, || {});

    // Then - the watch is refused
    assert!(watched.is_err());
}

use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::PathBuf;
use std::sync::atomic::{AtomicUsize, Ordering};

use super::{
    one_line, script_of, sweep, web_address, write_script, COMMAND_LIMIT, NOT_A_WEB_ADDRESS,
    NOT_ONE_LINE, URL_LIMIT,
};

fn scratch() -> PathBuf {
    static NEXT: AtomicUsize = AtomicUsize::new(0);
    let n = NEXT.fetch_add(1, Ordering::SeqCst);
    let dir = std::env::temp_dir().join(format!("window-opener-{}-{n}", std::process::id()));
    let _ = fs::remove_dir_all(&dir);
    dir
}

#[test]
fn a_web_address_opens_whole_as_sent() {
    for url in [
        "http://localhost:3000",
        "https://api.example.ts.net/",
        "http://192.168.64.5:8080/path?q=1#top",
    ] {
        assert_eq!(web_address(url), Ok(url));
    }
}

#[test]
fn an_address_of_another_scheme_is_refused() {
    for url in [
        "file:///etc/passwd",
        "javascript:alert(1)",
        "vscode://open?x=1",
        "ssh://linux-1",
        "data:text/html,hi",
        "mailto:a@b.c",
    ] {
        let refused = web_address(url).expect_err(url);
        assert_eq!(refused.code, NOT_A_WEB_ADDRESS, "{url}");
    }
}

#[test]
fn an_address_that_is_no_url_or_names_no_host_is_refused() {
    let long = format!("https://a.b/{}", "x".repeat(URL_LIMIT));
    for url in [
        "",
        "localhost:3000",
        "http://",
        "https://exa mple.com",
        "https://example.com/\n",
        " https://example.com",
        "https://example.com/\u{0}",
        "http:///path",
        "http:/example.com",
        "http:\\\\example.com",
        "HTTPS://example.com",
        long.as_str(),
    ] {
        let refused = web_address(url).expect_err(url);
        assert_eq!(refused.code, NOT_A_WEB_ADDRESS, "{url:?}");
    }
}

#[test]
fn a_command_runs_when_it_is_one_line() {
    for command in [
        "ssh admin@infra",
        "npm login",
        "station accounts share claude --with linux-1",
    ] {
        assert_eq!(one_line(command), Ok(command));
    }
}

#[test]
fn a_command_on_two_lines_or_holding_a_control_character_is_refused() {
    let long = "x".repeat(COMMAND_LIMIT + 1);
    for command in [
        "",
        "   ",
        "ssh infra\nrm -rf ~",
        "ssh infra\rrm -rf ~",
        "ssh infra\u{0}",
        "ssh \u{1b}[2Jinfra",
        "ssh\tinfra",
        long.as_str(),
    ] {
        let refused = one_line(command).expect_err(command);
        assert_eq!(refused.code, NOT_ONE_LINE, "{command:?}");
    }
}

#[test]
fn the_script_removes_itself_then_runs_the_command_verbatim() {
    assert_eq!(
        script_of("ssh admin@infra"),
        "#!/bin/sh\nrm -f -- \"$0\"\nssh admin@infra\n"
    );
}

#[test]
fn the_script_is_written_executable_by_its_owner_alone_in_a_folder_of_its_own() {
    let dir = scratch();
    let path = write_script(&dir, "npm login").expect("the script is written");
    assert_eq!(path.parent(), Some(dir.as_path()));
    assert!(path.to_string_lossy().ends_with(".command"));
    assert_eq!(
        fs::read_to_string(&path).expect("it reads"),
        script_of("npm login")
    );
    let mode = |p: &PathBuf| fs::metadata(p).expect("it exists").permissions().mode() & 0o777;
    assert_eq!(mode(&path), 0o700);
    assert_eq!(mode(&dir), 0o700);
    let other = write_script(&dir, "npm login").expect("a second one is written");
    assert_ne!(path, other);
    assert_eq!(sweep(&dir), 2);
    assert_eq!(sweep(&dir), 0);
    fs::remove_dir_all(&dir).expect("the folder goes");
}

#[test]
fn a_folder_that_is_a_link_is_refused() {
    let dir = scratch();
    let real = scratch();
    fs::create_dir_all(&real).expect("the real folder");
    std::os::unix::fs::symlink(&real, &dir).expect("the link");
    assert!(write_script(&dir, "npm login").is_err());
    assert_eq!(fs::read_dir(&real).expect("it reads").count(), 0);
    fs::remove_file(&dir).expect("the link goes");
    fs::remove_dir_all(&real).expect("the folder goes");
}

#[test]
fn a_refused_value_is_never_written() {
    let held = || fs::read_dir(super::folder()).map_or(0, Iterator::count);
    let before = held();
    let refused = super::open_in_terminal_now("ssh infra\nrm -rf ~").expect_err("two lines");
    assert_eq!(refused.code, NOT_ONE_LINE);
    let refused = super::open_url_now("file:///etc/passwd").expect_err("a file");
    assert_eq!(refused.code, NOT_A_WEB_ADDRESS);
    assert_eq!(held(), before);
}

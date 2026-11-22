use std::ffi::OsString;
use std::fs;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, Instant};

use serde_json::json;

use super::{
    environment, install_plan, move_plan, piped, start_plan, Ended, Heard, Line, Plan, Runs,
    Stream, BUSY, INSTALL_LINE, NOT_RUN, SEARCH_PATH,
};

fn scratch() -> PathBuf {
    static NEXT: AtomicUsize = AtomicUsize::new(0);
    let n = NEXT.fetch_add(1, Ordering::SeqCst);
    let dir = std::env::temp_dir().join(format!("window-core-{}-{n}", std::process::id()));
    let _ = fs::remove_dir_all(&dir);
    fs::create_dir_all(&dir).expect("the scratch folder is made");
    dir
}

#[derive(Default)]
struct Kept(Mutex<Vec<Line>>);

impl Heard for Kept {
    fn heard(&self, line: Line) {
        if let Ok(mut lines) = self.0.lock() {
            lines.push(line);
        }
    }
}

impl Kept {
    fn of(&self, stream: Stream) -> Vec<String> {
        self.0.lock().map_or_else(
            |_| Vec::new(),
            |lines| {
                lines
                    .iter()
                    .filter(|line| line.stream == stream)
                    .map(|line| line.text.clone())
                    .collect()
            },
        )
    }
}

fn shell(script: &str) -> Plan {
    Plan {
        program: PathBuf::from("/bin/sh"),
        args: vec![OsString::from("-c"), OsString::from(script)],
        env: environment(Path::new("/nowhere"), "/bin/zsh".as_ref(), |_| None),
    }
}

#[test]
fn the_child_gets_home_a_minimal_path_the_login_shell_and_nothing_else() {
    // Given - a parent environment holding a token, a user and an empty TMPDIR
    let parent = |name: &str| match name {
        "USER" => Some(OsString::from("someone")),
        "TMPDIR" => Some(OsString::new()),
        "GITHUB_TOKEN" => Some(OsString::from("not-passed")),
        _ => None,
    };

    // When - the child's environment is made, then printed by env
    let env = environment(Path::new("/Users/someone"), "/bin/bash".as_ref(), parent);
    let kept = Arc::new(Kept::default());
    let plan = Plan {
        program: PathBuf::from("/usr/bin/env"),
        args: Vec::new(),
        env,
    };
    let ended = Runs::default().run(&plan, kept.clone());

    // Then - exactly HOME, PATH, SHELL and USER reach it
    assert_eq!(
        ended,
        Ok(Ended {
            code: 0,
            cancelled: false,
            lines: 4
        })
    );
    let mut printed = kept.of(Stream::Out);
    printed.sort();
    assert_eq!(
        printed,
        [
            String::from("HOME=/Users/someone"),
            format!("PATH={SEARCH_PATH}"),
            String::from("SHELL=/bin/bash"),
            String::from("USER=someone"),
        ]
    );
}

#[test]
fn the_engine_gets_the_folders_and_the_machine_kind_the_window_was_started_with() {
    // Given - a parent environment holding the four XDG folders, the kind pin and a spec-only root
    let parent = |name: &str| match name {
        "XDG_CONFIG_HOME" => Some(OsString::from("/Users/someone/config")),
        "XDG_DATA_HOME" => Some(OsString::from("/Users/someone/data")),
        "XDG_STATE_HOME" => Some(OsString::from("/Users/someone/state")),
        "XDG_CACHE_HOME" => Some(OsString::from("/Users/someone/cache")),
        "STATION_MACHINE_KIND" => Some(OsString::from("mac")),
        "STATION_SYSTEM_ROOT" => Some(OsString::from("/not-passed")),
        _ => None,
    };

    // When - the child's environment is made
    let mut env = environment(Path::new("/Users/someone"), "/bin/zsh".as_ref(), parent);
    env.sort();

    // Then - the folders and the kind reach it beside HOME, PATH and SHELL, and the root does not
    assert_eq!(
        env,
        [
            ("HOME", "/Users/someone"),
            ("PATH", SEARCH_PATH),
            ("SHELL", "/bin/zsh"),
            ("STATION_MACHINE_KIND", "mac"),
            ("XDG_CACHE_HOME", "/Users/someone/cache"),
            ("XDG_CONFIG_HOME", "/Users/someone/config"),
            ("XDG_DATA_HOME", "/Users/someone/data"),
            ("XDG_STATE_HOME", "/Users/someone/state"),
        ]
        .map(|(name, value)| (String::from(name), OsString::from(value)))
    );
}

#[test]
fn each_line_is_streamed_as_sent_on_its_stream_then_the_exit_code() {
    // Given - a run that writes two lines out, one err and a last line with no line break
    let kept = Arc::new(Kept::default());
    let plan = shell("echo 'Detecting the newest release...'; echo 'Version: v1' ; echo 'warned' >&2; printf '  ✓ spwn v1'");

    // When - it runs
    let ended = Runs::default().run(&plan, kept.clone());

    // Then - every line is heard whole, in order, and the run ends 0
    assert_eq!(
        ended,
        Ok(Ended {
            code: 0,
            cancelled: false,
            lines: 4
        })
    );
    assert_eq!(
        kept.of(Stream::Out),
        [
            "Detecting the newest release...",
            "Version: v1",
            "  ✓ spwn v1"
        ]
    );
    assert_eq!(kept.of(Stream::Err), ["warned"]);
}

#[test]
fn a_failing_run_ends_on_its_code_with_its_error_line() {
    // Given - a run that says an error and exits 1, as the installer does
    let kept = Arc::new(Kept::default());
    let plan = shell(
        "echo 'Error: could not download station-darwin-arm64, nothing installed' >&2; exit 1",
    );

    // When - it runs
    let ended = Runs::default().run(&plan, kept.clone());

    // Then - the code is 1 and the line is heard as sent
    assert_eq!(
        ended,
        Ok(Ended {
            code: 1,
            cancelled: false,
            lines: 1
        })
    );
    assert_eq!(
        kept.of(Stream::Err),
        ["Error: could not download station-darwin-arm64, nothing installed"]
    );
}

#[test]
fn a_pipe_whose_first_command_fails_fails_the_run() {
    // Given - the plan the installer runs by, over a pipe whose download fails and whose shell succeeds
    let kept = Arc::new(Kept::default());
    let plan = piped(
        "echo 'curl: (6) Could not resolve host' >&2; false | cat",
        environment(Path::new("/nowhere"), "/bin/zsh".as_ref(), |_| None),
    );

    // When - it runs
    let ended = Runs::default().run(&plan, kept.clone());

    // Then - the run fails, as a bare `curl | bash` would not
    assert_eq!(
        ended,
        Ok(Ended {
            code: 1,
            cancelled: false,
            lines: 1
        })
    );
    assert_eq!(kept.of(Stream::Err), ["curl: (6) Could not resolve host"]);
}

#[test]
fn the_install_runs_the_public_line_under_pipefail_and_a_debug_build_never_runs_it_unfaked() {
    // Given - no fake, then a fake folder
    let env = environment(Path::new("/nowhere"), "/bin/zsh".as_ref(), |_| None);

    // When - the plans are made
    let unfaked = install_plan(None, env.clone());
    let faked = install_plan(Some(Path::new("/tmp/fake-core")), env.clone());
    let public = piped(INSTALL_LINE, env.clone());
    let start = start_plan(
        Some(Path::new("/tmp/fake-core")),
        Some(PathBuf::from("/tmp/fake-core/station")),
        env.clone(),
    );
    let nowhere = start_plan(Some(Path::new("/tmp/fake-core")), None, env.clone());

    // Then - a debug build refuses the real installer, runs the fake's script, and starts the fake's station
    assert_eq!(unfaked.map_err(|refusal| refusal.code), Err(NOT_RUN));
    assert_eq!(INSTALL_LINE, "curl -fsSL https://spwn.sh/install.sh | bash");
    assert_eq!(public.program, PathBuf::from("/bin/bash"));
    assert_eq!(
        public.args,
        ["-o", "pipefail", "-c", INSTALL_LINE].map(OsString::from)
    );
    assert_eq!(
        faked.map(|plan| (plan.program, plan.args)),
        Ok((
            PathBuf::from("/bin/sh"),
            vec![OsString::from("/tmp/fake-core/install.sh")]
        ))
    );
    assert_eq!(
        start.map(|plan| (plan.program, plan.args)),
        Ok((
            PathBuf::from("/tmp/fake-core/station"),
            ["engine", "start"].map(OsString::from).to_vec()
        ))
    );
    assert_eq!(nowhere.map_err(|refusal| refusal.code), Err(NOT_RUN));
    assert_eq!(
        start_plan(None, Some(PathBuf::from("/x/station")), env).map_err(|refusal| refusal.code),
        Err(NOT_RUN)
    );
}

fn alive(pid: i32) -> bool {
    unsafe { libc::kill(pid, 0) == 0 }
}

fn waits_for(what: impl Fn() -> bool) -> bool {
    let until = Instant::now() + Duration::from_secs(5);
    while Instant::now() < until {
        if what() {
            return true;
        }
        thread::sleep(Duration::from_millis(20));
    }
    false
}

#[test]
fn a_cancel_ends_the_whole_group_and_a_second_run_meanwhile_is_refused() {
    // Given - a run whose shell starts a long child and waits on it, its pid written down
    let dir = scratch();
    let pid_file = dir.join("child.pid");
    let plan = shell(&format!(
        "sleep 30 & echo $! > {}; echo started; wait",
        pid_file.display()
    ));
    let runs = Arc::new(Runs::default());
    let kept = Arc::new(Kept::default());
    let running = Arc::clone(&runs);
    let heard = kept.clone();
    let run = thread::spawn(move || running.run(&plan, heard));
    assert!(waits_for(|| kept.of(Stream::Out) == ["started"]));

    // When - a second run is asked, then the first is cancelled
    let second = runs.run(&shell("echo never"), Arc::new(Kept::default()));
    let cancelled = runs.cancel();
    let ended = run.join().expect("the run returns");

    // Then - the second was refused, the first ends cancelled, and its child is gone with it
    assert_eq!(second.map_err(|refusal| refusal.code), Err(BUSY));
    assert!(cancelled);
    assert_eq!(
        ended,
        Ok(Ended {
            code: 143,
            cancelled: true,
            lines: 1
        })
    );
    let child: i32 = fs::read_to_string(&pid_file)
        .expect("the child's pid is written")
        .trim()
        .parse()
        .expect("it is a pid");
    assert!(waits_for(|| !alive(child)));
    assert!(!runs.cancel());
}

fn carried(dir: &Path) -> PathBuf {
    let engine = dir.join("Spwn.app/Contents/MacOS/Spwn Engine");
    fs::create_dir_all(engine.parent().expect("the engine has a folder")).expect("it is made");
    fs::write(&engine, "#!/bin/sh\n").expect("the engine is written");
    fs::set_permissions(&engine, std::os::unix::fs::PermissionsExt::from_mode(0o755))
        .expect("it is executable");
    engine
}

fn answered(executable: &str) -> serde_json::Value {
    json!({ "v": 2, "id": "hello", "out": { "executable": executable } })
}

#[test]
fn a_bare_engine_answering_a_bundled_window_is_moved_by_the_bundles_engine_start() {
    // Given - a window whose Spwn.app carries its engine, and a bare engine answering its hello
    let dir = scratch();
    let bundled = carried(&dir);
    let env = environment(Path::new("/Users/you"), "/bin/zsh".as_ref(), |_| None);

    // When - the window judges the hello
    let plan = move_plan(
        Some(&bundled),
        &answered("/Users/you/.local/bin/station"),
        env.clone(),
    );

    // Then - the bundle's own engine runs `engine start`, which moves the Mac to mode A
    assert_eq!(
        plan,
        Some(Plan {
            program: bundled,
            args: vec![OsString::from("engine"), OsString::from("start")],
            env,
        })
    );
}

#[test]
fn an_engine_inside_a_bundle_an_unnamed_one_or_a_window_with_no_engine_moves_nothing() {
    // Given - a bundled window, and hellos from the bundle's engine, another copy's, and one naming no executable
    let dir = scratch();
    let bundled = carried(&dir);
    let env = || environment(Path::new("/Users/you"), "/bin/zsh".as_ref(), |_| None);
    let own = bundled.to_string_lossy().into_owned();

    // When / Then - none of them asks for a move
    for answer in [
        answered(&own),
        answered("/Applications/Spwn.app/Contents/MacOS/Spwn Engine"),
        json!({ "v": 2, "id": "hello", "out": {} }),
        json!({ "v": 2, "id": "hello", "error": { "code": "engine.down" } }),
    ] {
        assert_eq!(move_plan(Some(&bundled), &answer, env()), None, "{answer}");
    }

    // When / Then - a window outside a bundle, or whose bundle lost its engine, never moves a bare engine
    let bare = answered("/Users/you/.local/bin/station");
    assert_eq!(move_plan(None, &bare, env()), None);
    assert_eq!(
        move_plan(
            Some(&dir.join("Gone.app/Contents/MacOS/Spwn Engine")),
            &bare,
            env()
        ),
        None
    );
}

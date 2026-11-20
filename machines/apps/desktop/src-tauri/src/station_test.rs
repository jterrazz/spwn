use std::ffi::OsString;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicUsize, Ordering};

use super::{
    recorded, state_root, station_of, under_home, usable, Where, CORE_BINARY, INSTALLED_PATH,
    INSTALL_DIR, INSTALL_LINE, RECORD_PATH,
};

fn scratch() -> PathBuf {
    static NEXT: AtomicUsize = AtomicUsize::new(0);
    let n = NEXT.fetch_add(1, Ordering::SeqCst);
    let dir = std::env::temp_dir().join(format!("window-station-{}-{n}", std::process::id()));
    let _ = fs::remove_dir_all(&dir);
    fs::create_dir_all(&dir).expect("the scratch folder is made");
    dir
}

fn executable(path: &Path, mode: u32) {
    fs::create_dir_all(path.parent().expect("a file has a folder")).expect("its folder is made");
    fs::write(path, "#!/bin/sh\n").expect("the file is written");
    fs::set_permissions(path, fs::Permissions::from_mode(mode)).expect("its mode is set");
}

fn written(file: &Path, text: &str) {
    fs::create_dir_all(file.parent().expect("the record has a folder")).expect("it is made");
    fs::write(file, text).expect("the record is written");
}

fn record_at(file: &Path, executable: &Path) {
    let text = format!(
        "{{\"v\":1,\"executable\":{:?},\"version\":\"v0.21.0\",\"at\":\"2026-11-19T09:00:00Z\"}}",
        executable.display().to_string()
    );
    written(file, &text);
}

fn record(home: &Path, executable: &Path) {
    record_at(&home.join(".local/state").join(RECORD_PATH), executable);
}

fn installed_record(home: &Path) -> PathBuf {
    home.join(".local/state").join(INSTALLED_PATH)
}

fn looked(home: &Path, said: Option<&Path>) -> Option<PathBuf> {
    station_of(Where {
        faked: None,
        said,
        home,
        xdg_state: None,
    })
}

#[test]
fn the_install_constants_are_the_packages_install_json() {
    // Given - the constants build.rs read from packages/api/install.json

    // When - they are read

    // Then - they are the public installer's line, folder and binary, the engine's record and the installer's
    assert_eq!(INSTALL_LINE, "curl -fsSL https://spwn.sh/install.sh | bash");
    assert_eq!(INSTALL_DIR, "~/.local/bin");
    assert_eq!(CORE_BINARY, "station");
    assert_eq!(RECORD_PATH, "station/engine.json");
    assert_eq!(INSTALLED_PATH, "station/installed.json");
}

#[test]
fn only_an_executable_file_at_an_absolute_path_counts() {
    // Given - one executable station, one not executable, a folder by that name, a relative path
    let [found, plain, folder] = [scratch(), scratch(), scratch()];
    executable(&found.join("station"), 0o755);
    executable(&plain.join("station"), 0o644);
    fs::create_dir(folder.join("station")).expect("the folder is made");

    // When - each is looked at
    let seen = [
        found.join("station"),
        plain.join("station"),
        folder.join("station"),
        PathBuf::from("station"),
    ]
    .map(|path| usable(&path));

    // Then - only the executable file counts
    assert_eq!(seen, [true, false, false, false]);
}

#[test]
fn the_folder_and_the_state_root_are_under_home_never_a_path_lookup() {
    // Given - a home
    let home = Path::new("/Users/someone");

    // When - the install folder and the state root are asked, with and without XDG_STATE_HOME
    let folder = under_home(INSTALL_DIR, home);
    let plain = state_root(home, None);
    let relative = state_root(home, Some(OsString::from("state")));
    let xdg = state_root(home, Some(OsString::from("/var/state")));

    // Then - home's .local/bin, and the state root as the record's contract says
    assert_eq!(folder, PathBuf::from("/Users/someone/.local/bin"));
    assert_eq!(plain, PathBuf::from("/Users/someone/.local/state"));
    assert_eq!(relative, PathBuf::from("/Users/someone/.local/state"));
    assert_eq!(xdg, PathBuf::from("/var/state"));
}

#[test]
fn the_engine_said_wins_then_its_record_then_the_install_record_then_the_install_folder() {
    // Given - a home with a station in each of the four places
    let home = scratch();
    let said = home.join("said/station");
    let kept = home.join("kept/station");
    let landed = home.join("landed/station");
    let installed = home.join(".local/bin/station");
    for path in [&said, &kept, &landed, &installed] {
        executable(path, 0o755);
    }
    record(&home, &kept);
    record_at(&installed_record(&home), &landed);

    // When - station is looked for with what the engine said, then without, then with each record gone
    let first = looked(&home, Some(&said));
    let second = looked(&home, None);
    fs::remove_file(home.join(".local/state").join(RECORD_PATH)).expect("the record goes");
    let third = looked(&home, None);
    fs::remove_file(installed_record(&home)).expect("the install record goes");
    let fourth = looked(&home, None);

    // Then - each place in that order
    assert_eq!(first, Some(said));
    assert_eq!(second, Some(kept));
    assert_eq!(third, Some(landed));
    assert_eq!(fourth, Some(installed));
}

#[test]
fn the_install_record_is_read_under_xdg_state_home_when_it_is_absolute() {
    // Given - a state root of its own holding the install record, and no engine record
    let [home, state] = [scratch(), scratch()];
    let landed = home.join("landed/station");
    executable(&landed, 0o755);
    record_at(&state.join(INSTALLED_PATH), &landed);

    // When - station is looked for with XDG_STATE_HOME naming that root
    let found = station_of(Where {
        faked: None,
        said: None,
        home: &home,
        xdg_state: Some(state.into_os_string()),
    });

    // Then - it is the station the install record names
    assert_eq!(found, Some(landed));
}

#[test]
fn an_install_record_naming_no_usable_station_falls_to_the_install_folder() {
    // Given - a station in the install folder, and install records naming a relative path or a gone one
    let home = scratch();
    let installed = home.join(".local/bin/station");
    executable(&installed, 0o755);
    let records = [
        "{\"v\":1,\"executable\":\"station\",\"version\":\"v0.21.0\",\"at\":\"2026-11-19T09:00:00Z\"}"
            .to_owned(),
        format!(
            "{{\"v\":1,\"executable\":{:?},\"version\":\"v0.21.0\",\"at\":\"2026-11-19T09:00:00Z\"}}",
            home.join("gone/station").display().to_string()
        ),
    ];

    // When - station is looked for over each record
    let seen = records.map(|text| {
        written(&installed_record(&home), &text);
        looked(&home, None)
    });

    // Then - both times the install folder's
    assert_eq!(seen, [Some(installed.clone()), Some(installed)]);
}

#[test]
fn an_install_record_missing_malformed_of_another_version_or_unreadable_is_ignored() {
    // Given - a station in the install folder, another one an install record of version 2 names
    let home = scratch();
    let landed = home.join("landed/station");
    let installed = home.join(".local/bin/station");
    for path in [&landed, &installed] {
        executable(path, 0o755);
    }
    let file = installed_record(&home);
    let newer = format!(
        "{{\"v\":2,\"executable\":{:?},\"version\":\"v0.22.0\",\"at\":\"2026-11-19T09:00:00Z\"}}",
        landed.display().to_string()
    );

    // When - station is looked for with no record, a malformed one, one of version 2, then a folder no read opens
    let missing = looked(&home, None);
    written(&file, "{\"v\":1,\"executable\":");
    let malformed = looked(&home, None);
    written(&file, &newer);
    let other = looked(&home, None);
    fs::remove_file(&file).expect("the record goes");
    fs::create_dir(&file).expect("a folder takes its place");
    let unreadable = looked(&home, None);

    // Then - each time the install folder's
    assert_eq!(
        [missing, malformed, other, unreadable],
        [
            Some(installed.clone()),
            Some(installed.clone()),
            Some(installed.clone()),
            Some(installed)
        ]
    );
}

#[test]
fn a_place_holding_no_station_is_passed_over_and_none_found_is_none() {
    // Given - the engine names a path with nothing there, and the record names a file not executable
    let home = scratch();
    let kept = home.join("kept/station");
    executable(&kept, 0o644);
    record(&home, &kept);

    // When - station is looked for
    let found = looked(&home, Some(&home.join("gone/station")));

    // Then - nothing is found
    assert_eq!(found, None);
}

#[test]
fn a_record_of_another_version_or_shape_is_not_read() {
    // Given - three records: another version, no executable, not JSON
    let home = scratch();
    let file = home.join("engine.json");

    // When - each is read
    let seen = [
        "{\"v\":2,\"executable\":\"/x/station\"}",
        "{\"v\":1}",
        "not json",
    ]
    .map(|text| {
        fs::write(&file, text).expect("the record is written");
        recorded(&file)
    });

    // Then - none gives a path
    assert_eq!(seen, [None, None, None]);
}

#[test]
fn a_record_of_version_2_or_unreadable_is_ignored_for_the_install_folder() {
    // Given - a station in the install folder, another one a record of version 2 names
    let home = scratch();
    let kept = home.join("kept/station");
    let installed = home.join(".local/bin/station");
    for path in [&kept, &installed] {
        executable(path, 0o755);
    }
    let file = home.join(".local/state").join(RECORD_PATH);
    fs::create_dir_all(file.parent().expect("the record has a folder")).expect("it is made");
    let text = format!(
        "{{\"v\":2,\"executable\":{:?},\"version\":\"v0.22.0\",\"at\":\"2026-11-19T09:00:00Z\"}}",
        kept.display().to_string()
    );
    fs::write(&file, text).expect("the record is written");

    // When - station is looked for, then again once the record is a folder no read opens
    let newer = looked(&home, None);
    fs::remove_file(&file).expect("the record goes");
    fs::create_dir(&file).expect("a folder takes its place");
    let unreadable = looked(&home, None);

    // Then - both times the install folder's
    assert_eq!(newer, Some(installed.clone()));
    assert_eq!(unreadable, Some(installed));
}

#[test]
fn a_fake_folder_is_the_one_place_looked_at() {
    // Given - a fake folder holding station, and an engine naming another that exists
    let [fake, other] = [scratch(), scratch()];
    executable(&fake.join("station"), 0o755);
    executable(&other.join("station"), 0o755);

    // When - station is looked for under the fake
    let found = station_of(Where {
        faked: Some(&fake),
        said: Some(&other.join("station")),
        home: &other,
        xdg_state: None,
    });

    // Then - it is the fake's station, never what the engine said
    assert_eq!(found, Some(fake.join("station")));
}

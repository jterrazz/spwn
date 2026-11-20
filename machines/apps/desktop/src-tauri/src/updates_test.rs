use std::path::PathBuf;

use serde_json::json;
use tauri_plugin_updater::Error;

use super::{
    code_of, install_with, Checked, Pending, Progress, Slot, BUSY, GONE, INSTALL, UNPUBLISHED,
    UNREACHABLE, UNSIGNED,
};

#[cfg(any(debug_assertions, not(target_os = "macos")))]
#[test]
fn a_debug_build_or_one_off_macos_answers_unsupported_and_builds_no_updater() {
    use super::{window_update_check, CHECKS};
    use tauri::Manager;
    const { assert!(!CHECKS, "a debug build, or one off macOS, never checks") };
    let app = tauri::test::mock_builder()
        .manage(Pending::default())
        .build(tauri::test::mock_context(tauri::test::noop_assets()))
        .expect("the mock app builds");
    let checked = tauri::async_runtime::block_on(window_update_check(
        app.handle().clone(),
        app.state::<Pending>(),
    ));
    assert_eq!(checked, Ok(Checked::Unsupported));
}

#[test]
fn each_error_of_the_updater_is_one_code_of_the_window() {
    let cases = [
        (
            Error::Network(String::from("connection refused")),
            UNREACHABLE,
        ),
        (Error::EmptyEndpoints, UNREACHABLE),
        (Error::InsecureTransportProtocol, UNREACHABLE),
        (Error::ReleaseNotFound, UNPUBLISHED),
        (
            Error::TargetNotFound(String::from("darwin-aarch64-window")),
            UNPUBLISHED,
        ),
        (
            Error::TargetsNotFound(vec![String::from("darwin-aarch64")]),
            UNPUBLISHED,
        ),
        (
            Error::Serialization(serde_json::from_str::<u8>("{").unwrap_err()),
            UNPUBLISHED,
        ),
        (Error::SignatureUtf8(String::from("not base64")), UNSIGNED),
        (Error::MissingSignedVersion, UNSIGNED),
        (
            Error::SignedVersionMismatch {
                signed: String::from("0.21.0"),
                announced: String::from("0.22.0"),
            },
            UNSIGNED,
        ),
        (
            Error::Io(std::io::Error::other("read-only volume")),
            INSTALL,
        ),
        (
            Error::PreviousAppNotRestored(PathBuf::from("/tmp/old")),
            INSTALL,
        ),
        (Error::AuthenticationFailed, INSTALL),
        (Error::TempDirNotFound, INSTALL),
        (Error::BinaryNotFoundInArchive, INSTALL),
    ];
    for (error, code) in cases {
        assert_eq!(code_of(&error), code, "{error}");
    }
}

#[test]
fn an_install_with_nothing_pending_is_refused_as_gone() {
    let pending = Pending::default();
    let installed = tauri::async_runtime::block_on(install_with(&pending, |step| {
        panic!("nothing was pending, yet it heard {step:?}")
    }));
    assert_eq!(installed.map_err(|refusal| refusal.code), Err(GONE));
}

#[test]
fn a_slot_hands_its_update_out_once_and_takes_it_back_after_a_failure() {
    let slot = Slot::<u32>::default();
    slot.keep(Some(22))
        .expect("an idle slot keeps what was found");
    assert_eq!(slot.begin(), Ok(22));
    assert_eq!(slot.begin().map_err(|refusal| refusal.code), Err(BUSY));
    assert_eq!(
        slot.keep(Some(23)).map_err(|refusal| refusal.code),
        Err(BUSY)
    );
    slot.end(Some(22));
    assert_eq!(slot.begin(), Ok(22));
    slot.end(None);
    assert_eq!(slot.begin().map_err(|refusal| refusal.code), Err(GONE));
    assert_eq!(slot.begin().map_err(|refusal| refusal.code), Err(GONE));
}

#[test]
fn a_check_finding_nothing_newer_empties_the_slot() {
    let slot = Slot::<u32>::default();
    slot.keep(Some(22))
        .expect("an idle slot keeps what was found");
    slot.keep(None).expect("an idle slot forgets");
    assert_eq!(slot.begin().map_err(|refusal| refusal.code), Err(GONE));
}

#[test]
fn the_answers_keep_the_shapes_the_webview_parses() {
    let shapes = [
        (
            serde_json::to_value(Checked::Unsupported),
            json!({"kind": "unsupported"}),
        ),
        (
            serde_json::to_value(Checked::Current {
                current: String::from("0.21.0"),
            }),
            json!({"kind": "current", "current": "0.21.0"}),
        ),
        (
            serde_json::to_value(Checked::Out {
                current: String::from("0.21.0"),
                version: String::from("0.22.0"),
                date: Some(String::from("2026-11-20T10:00:00Z")),
            }),
            json!({"kind": "out", "current": "0.21.0", "version": "0.22.0", "date": "2026-11-20T10:00:00Z"}),
        ),
        (
            serde_json::to_value(Checked::Out {
                current: String::from("0.21.0"),
                version: String::from("0.22.0"),
                date: None,
            }),
            json!({"kind": "out", "current": "0.21.0", "version": "0.22.0"}),
        ),
        (
            serde_json::to_value(Progress::Downloaded {
                downloaded: 512,
                total: Some(2048),
            }),
            json!({"downloaded": 512, "total": 2048}),
        ),
        (
            serde_json::to_value(Progress::Downloaded {
                downloaded: 512,
                total: None,
            }),
            json!({"downloaded": 512}),
        ),
        (
            serde_json::to_value(Progress::Installed { installed: true }),
            json!({"installed": true}),
        ),
    ];
    for (shape, wanted) in shapes {
        assert_eq!(shape.expect("an answer serializes"), wanted);
    }
}

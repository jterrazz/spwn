use std::fs;
use std::path::PathBuf;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::time::{Duration, UNIX_EPOCH};

use super::{stamp, LinkLog, KEPT_BYTES};

fn folder() -> PathBuf {
    static NEXT: AtomicUsize = AtomicUsize::new(0);
    let n = NEXT.fetch_add(1, Ordering::SeqCst);
    let folder = std::env::temp_dir().join(format!("window-link-log-{}-{n}", std::process::id()));
    let _ = fs::remove_dir_all(&folder);
    folder
}

#[test]
fn a_stamp_is_the_utc_instant_to_the_millisecond() {
    // Given - the epoch, a leap day and a morning of 2026
    let epoch = UNIX_EPOCH;
    let leap = UNIX_EPOCH + Duration::from_secs(951_782_400);
    let morning = UNIX_EPOCH + Duration::from_millis(1_795_257_723_120);

    // When - each is stamped
    let stamps = [stamp(epoch), stamp(leap), stamp(morning)];

    // Then - each is its ISO 8601 instant in UTC
    assert_eq!(
        stamps,
        [
            "1970-01-01T00:00:00.000Z",
            "2000-02-29T00:00:00.000Z",
            "2026-11-21T10:42:03.120Z"
        ]
    );
}

#[test]
fn a_log_appends_one_stamped_line_per_write_in_its_folder() {
    // Given - a log in a folder that does not exist yet
    let folder = folder();
    let log = LinkLog::in_folder(Some(folder.clone()));

    // When - two lines are written
    log.write(UNIX_EPOCH, "link open: answered");
    log.write(
        UNIX_EPOCH + Duration::from_secs(1),
        "link absent: hello failed at connect: refused",
    );

    // Then - the folder holds link.log, the two lines in order
    let written = fs::read_to_string(folder.join("link.log")).expect("the log was written");
    assert_eq!(
        written,
        "1970-01-01T00:00:00.000Z link open: answered\n1970-01-01T00:00:01.000Z link absent: hello failed at connect: refused\n"
    );
    let _ = fs::remove_dir_all(folder);
}

#[test]
fn a_full_log_is_kept_once_aside_and_a_log_with_no_folder_writes_nothing() {
    // Given - a log already at its size, and a log the shell gave no folder
    let folder = folder();
    fs::create_dir_all(&folder).expect("the folder is made");
    let full = vec![b'x'; usize::try_from(KEPT_BYTES).expect("the size fits")];
    fs::write(folder.join("link.log"), full).expect("the full log is written");
    let nowhere = LinkLog::in_folder(None);

    // When - each writes a line
    LinkLog::in_folder(Some(folder.clone())).write(UNIX_EPOCH, "link open: answered");
    nowhere.write(UNIX_EPOCH, "link open: answered");

    // Then - the full log moved to link.log.1, and link.log starts again on the new line
    let aside = fs::metadata(folder.join("link.log.1")).expect("the old log is kept");
    assert_eq!(aside.len(), KEPT_BYTES);
    let written = fs::read_to_string(folder.join("link.log")).expect("the log was written");
    assert_eq!(written, "1970-01-01T00:00:00.000Z link open: answered\n");
    let _ = fs::remove_dir_all(folder);
}

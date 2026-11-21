use std::fs::{self, OpenOptions};
use std::io::Write;
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

const KEPT_BYTES: u64 = 1 << 20;

pub struct LinkLog {
    file: Option<PathBuf>,
}

impl LinkLog {
    pub fn in_folder(folder: Option<PathBuf>) -> Self {
        LinkLog {
            file: folder.map(|folder| folder.join("link.log")),
        }
    }

    // invariant: a line the log cannot write is dropped: the log never stops the link it tells.
    pub fn write(&self, at: SystemTime, said: &str) {
        let Some(file) = &self.file else {
            return;
        };
        if let Some(folder) = file.parent() {
            let _ = fs::create_dir_all(folder);
        }
        rotated(file);
        if let Ok(mut out) = OpenOptions::new().create(true).append(true).open(file) {
            let _ = writeln!(out, "{} {said}", stamp(at));
        }
    }
}

fn rotated(file: &Path) {
    if fs::metadata(file).is_ok_and(|meta| meta.len() >= KEPT_BYTES) {
        let _ = fs::rename(file, file.with_extension("log.1"));
    }
}

pub fn stamp(at: SystemTime) -> String {
    let since = at.duration_since(UNIX_EPOCH).unwrap_or_default();
    let seconds = since.as_secs();
    let (year, month, day) = civil(seconds / 86_400);
    let time = seconds % 86_400;
    format!(
        "{year:04}-{month:02}-{day:02}T{:02}:{:02}:{:02}.{:03}Z",
        time / 3600,
        time % 3600 / 60,
        time % 60,
        since.subsec_millis()
    )
}

// value: the Gregorian date of a day counted from 1970-01-01, by Howard Hinnant's `civil_from_days`.
fn civil(days: u64) -> (u64, u64, u64) {
    let shifted = days + 719_468;
    let era = shifted / 146_097;
    let of_era = shifted % 146_097;
    let year_of_era = (of_era - of_era / 1460 + of_era / 36_524 - of_era / 146_096) / 365;
    let of_year = of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100);
    let shifted_month = (5 * of_year + 2) / 153;
    let day = of_year - (153 * shifted_month + 2) / 5 + 1;
    let month = if shifted_month < 10 {
        shifted_month + 3
    } else {
        shifted_month - 9
    };
    let year = year_of_era + era * 400 + u64::from(month <= 2);
    (year, month, day)
}

#[cfg(test)]
#[path = "link_log_test.rs"]
mod tests;

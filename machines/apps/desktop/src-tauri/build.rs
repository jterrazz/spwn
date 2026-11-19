use std::env;
use std::fs;
use std::path::Path;

const BRIDGED: &str = "../src/domain/bridged.ts";
const CORE: &str = "../src/domain/core.ts";

fn bridged() -> Vec<String> {
    let source = fs::read_to_string(BRIDGED).expect("the bridged list is readable");
    let list = source
        .split_once("BRIDGED = [")
        .and_then(|(_, rest)| rest.split_once(']'))
        .map(|(list, _)| list)
        .expect("bridged.ts holds `BRIDGED = [ ... ]`");
    let names: Vec<String> = list
        .lines()
        .map(str::trim)
        .filter(|line| !line.is_empty())
        .map(|line| {
            line.trim_end_matches(',')
                .strip_prefix('\'')
                .and_then(|name| name.strip_suffix('\''))
                .filter(|name| {
                    name.chars()
                        .all(|c| c.is_ascii_lowercase() || c == '.' || c == '-')
                })
                .unwrap_or_else(|| panic!("bridged.ts names one call a line, not {line}"))
                .to_owned()
        })
        .collect();
    assert!(!names.is_empty(), "bridged.ts names no call");
    names
}

fn install_line() -> String {
    let source = fs::read_to_string(CORE).expect("the core's lines are readable");
    let line = source
        .split_once("INSTALL_LINE = '")
        .and_then(|(_, rest)| rest.split_once('\''))
        .map(|(line, _)| line)
        .expect("core.ts holds `INSTALL_LINE = '...'`");
    assert!(
        !line.is_empty() && !line.chars().any(char::is_control),
        "the install line is one line"
    );
    line.to_owned()
}

fn main() {
    println!("cargo:rerun-if-changed={BRIDGED}");
    println!("cargo:rerun-if-changed={CORE}");
    let names = bridged();
    let dir = env::var("OUT_DIR").expect("cargo sets OUT_DIR");
    let out = Path::new(&dir).join("bridged.rs");
    let quoted: Vec<String> = names.iter().map(|name| format!("{name:?}")).collect();
    fs::write(
        out,
        format!("pub const BRIDGED: &[&str] = &[{}];\n", quoted.join(", ")),
    )
    .expect("the bridged list is written");
    fs::write(
        Path::new(&dir).join("install_line.rs"),
        format!("pub const INSTALL_LINE: &str = {:?};\n", install_line()),
    )
    .expect("the install line is written");
    tauri_build::build();
}

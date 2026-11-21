use std::env;
use std::fs;
use std::path::Path;

const BRIDGED: &str = "../src/domain/bridged.ts";
const INSTALL: &str = "../../../packages/api/install.json";

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

fn one_line(install: &serde_json::Value, pointer: &str) -> String {
    let value = install
        .pointer(pointer)
        .and_then(serde_json::Value::as_str)
        .unwrap_or_else(|| panic!("install.json holds a string at {pointer}"));
    assert!(
        !value.is_empty() && !value.chars().any(char::is_control),
        "install.json's {pointer} is one line"
    );
    value.to_owned()
}

fn install() -> String {
    let source = fs::read_to_string(INSTALL).expect("install.json is readable");
    let install: serde_json::Value = serde_json::from_str(&source).expect("install.json is JSON");
    for record in ["/record/root", "/installed/root"] {
        assert_eq!(
            one_line(&install, record),
            "state",
            "install.json's {record} is the state root"
        );
    }
    [
        ("INSTALL_LINE", "/line"),
        ("INSTALL_DIR", "/dir"),
        ("CORE_BINARY", "/binary"),
        ("BUNDLED_ENGINE", "/bundled"),
        ("RECORD_PATH", "/record/path"),
        ("INSTALLED_PATH", "/installed/path"),
    ]
    .map(|(name, pointer)| {
        format!(
            "pub const {name}: &str = {:?};\n",
            one_line(&install, pointer)
        )
    })
    .concat()
}

fn main() {
    println!("cargo:rerun-if-changed={BRIDGED}");
    println!("cargo:rerun-if-changed={INSTALL}");
    let names = bridged();
    let dir = env::var("OUT_DIR").expect("cargo sets OUT_DIR");
    let out = Path::new(&dir).join("bridged.rs");
    let quoted: Vec<String> = names.iter().map(|name| format!("{name:?}")).collect();
    fs::write(
        out,
        format!("pub const BRIDGED: &[&str] = &[{}];\n", quoted.join(", ")),
    )
    .expect("the bridged list is written");
    fs::write(Path::new(&dir).join("install.rs"), install())
        .expect("the install constants are written");
    tauri_build::build();
}

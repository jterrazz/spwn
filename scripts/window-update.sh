#!/usr/bin/env bash

set -euo pipefail

usage() {
    cat >&2 <<'EOF'
usage: window-update.sh <step> <paths...>

  archive <app> <dir>     pack the stapled Spwn.app into <dir>/Spwn.app.tar.gz, then judge the copy it holds: codesign strict, and on a notarized run stapler and Gatekeeper
  sign <archive>          sign the archive with the Tauri key into <archive>.sig, bound to the app's version, then verify it
  verify <archive>        check <archive>.sig against plugins.updater.pubkey of the window's tauri.conf.json, and that it is bound to the app's version
  manifest <app> <dir>    write the updater manifest the window's endpoint names, from the app, its archive and its signature in <dir>

env:
  TAURI_SIGNING_PRIVATE_KEY           the private half of the window's updater key, read by `tauri signer sign` itself
  TAURI_SIGNING_PRIVATE_KEY_PASSWORD  its password, read by `tauri signer sign` itself; unset means none
  APPLE_SIGNING_IDENTITY=-            archive judges the copy by codesign alone: an ad hoc app carries no ticket
  MACOS_NOTARIZE=skip                 the same, for a run that skipped the notarization
EOF
    exit 2
}

fail() {
    printf '::error::%s\n' "$*" >&2
    exit 1
}

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
machines=$root/machines
config=$machines/apps/desktop/src-tauri/tauri.conf.json
updates=$machines/apps/desktop/src-tauri/src/updates.rs
bundle=Spwn.app
packed=Spwn.app.tar.gz

scratch() {
    local dir
    dir=$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/window-update.XXXXXX")
    chmod 700 "$dir"
    printf '%s' "$dir"
}

version_of() {
    plutil -extract CFBundleShortVersionString raw "$1/Contents/Info.plist" ||
        fail "$1 carries no CFBundleShortVersionString"
}

app_of() {
    local app=$1
    [ -f "$app/Contents/Info.plist" ] || fail "$app is not an app bundle"
    [ "$(basename "$app")" = "$bundle" ] || fail "$app is not $bundle, the bundle the updater swaps"
}

# value: the version of the app an archive holds, read from its own Info.plist
packed_version() {
    local archive=$1 dir version
    dir=$(scratch)
    tar -xzf "$archive" -C "$dir" "$bundle/Contents/Info.plist" || {
        rm -rf "$dir"
        fail "$archive holds no $bundle/Contents/Info.plist"
    }
    version=$(version_of "$dir/$bundle")
    rm -rf "$dir"
    printf '%s' "$version"
}

# invariant: the platform key has one owner, WINDOW_TARGET in updates.rs, and the window looks up no other
target_of() {
    local lines=() line
    while IFS= read -r line; do lines+=("$line"); done < <(grep -E '^pub const WINDOW_TARGET: &str = "[a-z0-9-]+";$' "$updates")
    [ ${#lines[@]} -eq 1 ] || fail "$updates holds ${#lines[@]} lines declaring WINDOW_TARGET, not one"
    [[ ${lines[0]} =~ \"([a-z0-9-]+)\" ]] || fail "WINDOW_TARGET is not a quoted name"
    printf '%s' "${BASH_REMATCH[1]}"
}

# invariant: the window reads one endpoint, the latest release's manifest, so a release's assets sit beside it under its tag
endpoint_of() {
    local endpoint
    endpoint=$(jq -r '.plugins.updater.endpoints | if length == 1 then .[0] else "" end' "$config")
    [[ $endpoint =~ ^https://github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+/releases/latest/download/[A-Za-z0-9_.-]+\.json$ ]] ||
        fail "plugins.updater.endpoints of $config is not one https://github.com/<owner>/<repo>/releases/latest/download/<file>.json"
    printf '%s' "$endpoint"
}

archive() {
    [ $# -eq 2 ] || usage
    local app=$1 dir=$2 copy entry
    app_of "$app"
    mkdir -p "$dir"
    COPYFILE_DISABLE=1 tar --no-mac-metadata --no-xattrs -czf "$dir/$packed" -C "$(dirname "$app")" "$bundle"
    while IFS= read -r entry; do
        [[ $entry == "$bundle"/* || $entry == "$bundle" ]] || fail "archive: $packed holds $entry, outside $bundle/"
        [[ /$entry != */._* ]] || fail "archive: $packed holds the AppleDouble file $entry"
    done < <(tar -tzf "$dir/$packed")
    copy=$(scratch)
    tar -xzf "$dir/$packed" -C "$copy"
    codesign --verify --strict --verbose=2 "$copy/$bundle"
    if [ "${APPLE_SIGNING_IDENTITY:-}" != - ] && [ "${MACOS_NOTARIZE:-}" != skip ]; then
        xcrun stapler validate "$copy/$bundle"
        spctl --assess --type execute --verbose=2 "$copy/$bundle"
    fi
    rm -rf "$copy"
    echo "archive: $dir/$packed holds $bundle $(version_of "$app"), and the copy it holds verifies"
}

sign() {
    { set +x; } 2>/dev/null
    [ $# -eq 1 ] || usage
    local archive version
    [ -f "$1" ] || fail "sign: $1 does not exist"
    archive=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
    [ -n "${TAURI_SIGNING_PRIVATE_KEY:-}" ] || fail "sign: TAURI_SIGNING_PRIVATE_KEY is not set (docs/guides/release.md, \"The Tauri updater key\")"
    version=$(packed_version "$archive")
    rm -f "$archive.sig"
    (cd "$machines" && npx --no-install tauri signer sign --app-version "$version" "$archive" </dev/null >/dev/null) ||
        fail "sign: tauri signer could not sign $archive with TAURI_SIGNING_PRIVATE_KEY"
    [ -s "$archive.sig" ] || fail "sign: tauri signer wrote no $archive.sig"
    verify "$archive"
    echo "sign: $archive.sig signs $bundle $version"
}

verify() {
    [ $# -eq 1 ] || usage
    local archive=$1 version dir pubkey key comment
    [ -f "$archive" ] || fail "verify: $archive does not exist"
    [ -f "$archive.sig" ] || fail "verify: $archive.sig does not exist"
    command -v rsign >/dev/null || fail "verify: rsign is not on PATH (cargo install rsign2 --version 0.6.7 --locked)"
    dir=$(scratch)
    pubkey=$(jq -r '.plugins.updater.pubkey // ""' "$config")
    [ -n "$pubkey" ] || fail "verify: $config holds no plugins.updater.pubkey"
    base64 --decode <<< "$pubkey" > "$dir/key.pub" || fail "verify: plugins.updater.pubkey is not base64"
    key=$(sed -n 2p "$dir/key.pub")
    base64 --decode < "$archive.sig" > "$dir/sig.minisig" || fail "verify: $archive.sig is not base64"
    rsign verify -q -P "$key" -x "$dir/sig.minisig" "$archive" ||
        fail "the updater signature of $packed does not verify against plugins.updater.pubkey of tauri.conf.json: TAURI_SIGNING_PRIVATE_KEY is not the private half of that key (docs/guides/release.md, \"The Tauri updater key\")"
    comment=$(sed -n 3p "$dir/sig.minisig")
    rm -rf "$dir"
    version=$(packed_version "$archive")
    [[ $'\t'${comment#trusted comment: }$'\t' == *$'\t'"version:$version"$'\t'* ]] ||
        fail "verify: $archive.sig is not bound to version $version, which requireSignedVersion requires"
    echo "verify: $archive.sig verifies against plugins.updater.pubkey, bound to $version"
}

manifest() {
    [ $# -eq 2 ] || usage
    local app=$1 dir=$2 version endpoint target url file
    app_of "$app"
    version=$(version_of "$app")
    [ -f "$dir/$packed.sig" ] || fail "manifest: $dir/$packed.sig does not exist; run sign first"
    [ "$(packed_version "$dir/$packed")" = "$version" ] || fail "manifest: $dir/$packed holds another version than $app's $version"
    endpoint=$(endpoint_of)
    target=$(target_of)
    url=${endpoint%/latest/download/*}/download/v$version/$packed
    file=$dir/$(basename "$endpoint")
    jq -n \
        --arg version "$version" \
        --arg notes "Spwn v$version: ${endpoint%/latest/download/*}/tag/v$version" \
        --arg date "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --arg target "$target" \
        --arg url "$url" \
        --rawfile signature "$dir/$packed.sig" \
        '{version: $version, notes: $notes, pub_date: $date, platforms: {($target): {url: $url, signature: $signature}}}' > "$file"
    echo "manifest: $file offers $version to $target, at $url"
}

[ $# -gt 0 ] || usage
step=$1
shift
case "$step" in
    archive | sign | verify | manifest) "$step" "$@" ;;
    *) usage ;;
esac

#!/usr/bin/env bash

set -euo pipefail

secrets=(APPLE_CERTIFICATE APPLE_CERTIFICATE_PASSWORD APPLE_SIGNING_IDENTITY APPLE_ID APPLE_PASSWORD APPLE_TEAM_ID TAURI_SIGNING_PRIVATE_KEY)

usage() {
    cat >&2 <<'EOF'
usage: sign-macos.sh <step> [paths...]

  require                 decide whether this run signs: every secret set, or none on a ref Infisical does not trust
  keychain                import APPLE_CERTIFICATE into a keychain of this run, and check APPLE_SIGNING_IDENTITY is in it, of APPLE_TEAM_ID
  sign <path>...          codesign each binary or .app with the hardened runtime and a timestamp, then verify it
  notarize <file>...      submit each .zip or .dmg to Apple's notary service and wait for its verdict
  staple <path>...        staple the notarization ticket to each .app or .dmg, then let Gatekeeper judge it
  binary <target> <path>  GoReleaser's post-build hook: on a signed run, sign a darwin binary and notarize it inside a zip that is not shipped
  cleanup                 delete the keychain and the files of this run

env:
  the secrets require checks: the six APPLE_* ones and TAURI_SIGNING_PRIVATE_KEY, which signs the window's update
  (scripts/window-update.sh); TAURI_SIGNING_PRIVATE_KEY_PASSWORD is not checked, since a key may have none
  APPLE_SIGNING_IDENTITY  the Developer ID Application identity, its name or its SHA-1; `-` signs ad hoc, with no keychain and no timestamp
  MACOS_NOTARIZE=skip     notarize and staple print what they skip and succeed: the dry run of a signing
  MACOS_SIGN=true         `binary` signs; any other value passes the binary through untouched
EOF
    exit 2
}

fail() {
    printf '::error::%s\n' "$*" >&2
    exit 1
}

# value: a step of a workflow passes what it found to the next through these files; run by hand, they print instead
publish() {
    local file=$1 name=$2 value=$3
    if [ -n "${!file:-}" ]; then
        printf '%s=%s\n' "$name" "$value" >> "${!file}"
    else
        printf '%s %s=%s\n' "$file" "$name" "$value"
    fi
}

adhoc() {
    [ "${APPLE_SIGNING_IDENTITY:-}" = - ]
}

skipped() {
    [ "${MACOS_NOTARIZE:-}" = skip ]
}

# invariant: Infisical gives a token to main and to a v* tag alone, so those two refs read the secrets and must find all of them; any other ref reads none
trusted() {
    case "${GITHUB_REF:-}" in refs/heads/main | refs/tags/v*) return 0 ;; *) return 1 ;; esac
}

require() {
    local missing=() name
    for name in "${secrets[@]}"; do
        [ -n "${!name:-}" ] || missing+=("$name")
    done
    if [ ${#missing[@]} -eq 0 ]; then
        echo "signing: every secret is set, so this run signs and notarizes the macOS artefacts"
        publish GITHUB_OUTPUT signed true
        return
    fi
    if trusted; then
        fail "${GITHUB_REF} reads the signing secrets from Infisical, and these are not set: ${missing[*]} (docs/guides/release.md, \"Signing and notarization\")"
    fi
    if [ ${#missing[@]} -lt ${#secrets[@]} ]; then
        fail "the signing secrets are set in part, and these are missing: ${missing[*]}"
    fi
    echo "::notice::no signing secret is set: ${GITHUB_REF:-this run} is neither main nor a v* tag, reads nothing from Infisical, and builds the macOS artefacts unsigned"
    publish GITHUB_OUTPUT signed false
}

keychain() {
    local name
    for name in APPLE_CERTIFICATE APPLE_CERTIFICATE_PASSWORD APPLE_SIGNING_IDENTITY APPLE_TEAM_ID; do
        [ -n "${!name:-}" ] || fail "keychain: $name is not set"
    done
    adhoc && fail "keychain: APPLE_SIGNING_IDENTITY is -, which signs ad hoc and needs no keychain"
    local dir chain password
    dir=$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/signing.XXXXXX")
    chmod 700 "$dir"
    chain="$dir/signing.keychain-db"
    password=$(openssl rand -hex 24)
    echo "::add-mask::$password"
    publish GITHUB_ENV MACOS_SIGN_DIR "$dir"
    publish GITHUB_ENV MACOS_SIGN_KEYCHAIN "$chain"

    (umask 077 && printf '%s' "$APPLE_CERTIFICATE" | base64 --decode > "$dir/certificate.p12") ||
        fail "keychain: APPLE_CERTIFICATE is not base64"
    security create-keychain -p "$password" "$chain"
    security set-keychain-settings -lut 21600 "$chain"
    security unlock-keychain -p "$password" "$chain"
    security import "$dir/certificate.p12" -k "$chain" -f pkcs12 -P "$APPLE_CERTIFICATE_PASSWORD" -T /usr/bin/codesign >/dev/null ||
        fail "keychain: APPLE_CERTIFICATE does not open with APPLE_CERTIFICATE_PASSWORD"
    rm -f "$dir/certificate.p12"
    # invariant: codesign reads the key without a prompt only once the partition list names it; a CI runner has nobody to click Allow
    security set-key-partition-list -S apple-tool:,apple:,codesign: -s -k "$password" "$chain" >/dev/null
    local chains=()
    while IFS= read -r name; do
        name=${name#"${name%%[![:space:]]*}"}
        chains+=("${name//\"/}")
    done < <(security list-keychains -d user)
    security list-keychains -d user -s "$chain" "${chains[@]}"

    team_of_identity "$(security find-identity -v -p codesigning "$chain")"
    echo "keychain: the Developer ID Application identity APPLE_SIGNING_IDENTITY names is imported, of team APPLE_TEAM_ID"
}

# value: reads `security find-identity` lines, `  1) <SHA-1> "<name>"`, and never prints one: the identity's name carries the owner's name
team_of_identity() {
    local listing=$1 line hash name found=
    while IFS= read -r line; do
        [[ $line =~ ^[[:space:]]*[0-9]+\)[[:space:]]+([0-9A-F]{40})[[:space:]]+\"(.*)\"$ ]] || continue
        hash=${BASH_REMATCH[1]}
        name=${BASH_REMATCH[2]}
        if [ "$APPLE_SIGNING_IDENTITY" = "$hash" ] || [ "$APPLE_SIGNING_IDENTITY" = "$name" ]; then
            found=$name
            break
        fi
    done <<< "$listing"
    [ -n "$found" ] || fail "keychain: APPLE_CERTIFICATE holds no valid codesigning identity APPLE_SIGNING_IDENTITY names, with its private key"
    [[ $found == "Developer ID Application: "* ]] || fail "keychain: APPLE_SIGNING_IDENTITY is not a Developer ID Application identity"
    [[ $found == *" ($APPLE_TEAM_ID)" ]] || fail "keychain: APPLE_SIGNING_IDENTITY belongs to another team than APPLE_TEAM_ID"
}

codesign_one() {
    local path=$1 args=(--force --options runtime --sign "$APPLE_SIGNING_IDENTITY")
    if adhoc; then
        args+=(--timestamp=none)
    else
        args+=(--timestamp --keychain "$MACOS_SIGN_KEYCHAIN")
    fi
    codesign "${args[@]}" "$path"
}

sign() {
    [ $# -gt 0 ] || usage
    [ -n "${APPLE_SIGNING_IDENTITY:-}" ] || fail "sign: APPLE_SIGNING_IDENTITY is not set; set it to - to sign ad hoc"
    if ! adhoc; then
        [ -n "${MACOS_SIGN_KEYCHAIN:-}" ] || fail "sign: MACOS_SIGN_KEYCHAIN is not set; run the keychain step first"
        [ -n "${APPLE_TEAM_ID:-}" ] || fail "sign: APPLE_TEAM_ID is not set"
    fi
    local path nested details
    for path in "$@"; do
        [ -e "$path" ] || fail "sign: $path does not exist"
        if [ -d "$path" ]; then
            [ -f "$path/Contents/Info.plist" ] || fail "sign: $path is a folder but not an app bundle"
            # invariant: a bundle is sealed inside out, every nested piece of code signed before the bundle that holds it
            while IFS= read -r -d '' nested; do
                codesign_one "$nested"
            done < <(find "$path/Contents" -depth \( -name '*.framework' -o -name '*.dylib' -o -name '*.app' -o -name '*.xpc' -o -name '*.appex' \) -print0)
        fi
        codesign_one "$path"
        codesign --verify --strict --verbose=2 "$path"
        details=$(codesign --display --verbose=2 "$path" 2>&1)
        grep -Eq '^CodeDirectory .*flags=0x[0-9a-f]+\([^)]*runtime' <<< "$details" || fail "sign: $path is signed without the hardened runtime"
        if ! adhoc; then
            grep -qx "TeamIdentifier=$APPLE_TEAM_ID" <<< "$details" || fail "sign: $path is signed by another team than APPLE_TEAM_ID"
            grep -q '^Timestamp=' <<< "$details" || fail "sign: $path carries no secure timestamp"
        fi
        echo "sign: $path signed with the hardened runtime and verified strict"
    done
}

notarize() {
    [ $# -gt 0 ] || usage
    local file
    if skipped; then
        for file in "$@"; do echo "notarize: skipped $file, MACOS_NOTARIZE=skip"; done
        return
    fi
    local name
    for name in APPLE_ID APPLE_PASSWORD APPLE_TEAM_ID; do
        [ -n "${!name:-}" ] || fail "notarize: $name is not set"
    done
    local answer id status
    for file in "$@"; do
        case "$file" in *.zip | *.dmg | *.pkg) ;; *) fail "notarize: $file is not a .zip, .dmg or .pkg, the only shapes the notary service takes" ;; esac
        # hazard: notarytool takes the app-specific password on its command line alone, so it is in the runner's process list while it runs; the runner is the job's own, and the log masks it
        answer=$(xcrun notarytool submit "$file" --apple-id "$APPLE_ID" --password "$APPLE_PASSWORD" --team-id "$APPLE_TEAM_ID" --wait --timeout 30m --output-format json) ||
            fail "notarize: notarytool could not submit $file: $answer"
        id=$(plutil -extract id raw -o - - <<< "$answer")
        status=$(plutil -extract status raw -o - - <<< "$answer")
        if [ "$status" != Accepted ]; then
            xcrun notarytool log "$id" --apple-id "$APPLE_ID" --password "$APPLE_PASSWORD" --team-id "$APPLE_TEAM_ID" >&2 || true
            fail "notarize: Apple answered $status for $file (submission $id)"
        fi
        echo "notarize: $file accepted (submission $id)"
    done
}

staple() {
    [ $# -gt 0 ] || usage
    local path
    for path in "$@"; do
        if skipped; then
            echo "staple: skipped $path, MACOS_NOTARIZE=skip"
            continue
        fi
        xcrun stapler staple "$path"
        xcrun stapler validate "$path"
        case "$path" in
            *.app) spctl --assess --type execute --verbose=2 "$path" ;;
            *.dmg) spctl --assess --type open --context context:primary-signature --verbose=2 "$path" ;;
        esac
        echo "staple: $path stapled, and Gatekeeper accepts it"
    done
}

binary() {
    [ $# -eq 2 ] || usage
    local target=$1 path=$2
    if [ "${MACOS_SIGN:-}" != true ] || [[ $target != darwin_* ]]; then
        echo "binary: $target left as built, MACOS_SIGN is not true or the target is not darwin"
        return
    fi
    sign "$path"
    local dir
    dir=$(mktemp -d "${MACOS_SIGN_DIR:-${RUNNER_TEMP:-${TMPDIR:-/tmp}}}/notary.XXXXXX")
    mkdir "$dir/$target"
    cp "$path" "$dir/$target/"
    ditto -c -k --keepParent "$dir/$target" "$dir/$target.zip"
    notarize "$dir/$target.zip"
    rm -rf "$dir"
}

cleanup() {
    if [ -n "${MACOS_SIGN_KEYCHAIN:-}" ] && [ -e "$MACOS_SIGN_KEYCHAIN" ]; then
        security delete-keychain "$MACOS_SIGN_KEYCHAIN"
    fi
    if [ -n "${MACOS_SIGN_DIR:-}" ]; then
        rm -rf "$MACOS_SIGN_DIR"
    fi
    echo "cleanup: the keychain and the files of this run are deleted"
}

[ $# -gt 0 ] || usage
step=$1
shift
case "$step" in
    require | keychain | sign | notarize | staple | binary | cleanup) "$step" "$@" ;;
    *) usage ;;
esac

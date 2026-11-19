#!/usr/bin/env bash

set -euo pipefail

secrets=(MACOS_CERT_P12_BASE64 MACOS_CERT_PASSWORD APPLE_API_KEY_P8_BASE64 APPLE_API_KEY_ID APPLE_API_ISSUER_ID APPLE_TEAM_ID)

usage() {
    cat >&2 <<'EOF'
usage: sign-macos.sh <step> [paths...]

  require             decide whether this run signs: every secret set, or none and not a tag
  keychain            import MACOS_CERT_P12_BASE64 into a keychain of this run, and write the notary key
  sign <path>...      codesign each binary or .app with the hardened runtime and a timestamp, then verify it
  notarize <file>...  submit each .zip or .dmg to Apple's notary service and wait for its verdict
  staple <path>...    staple the notarization ticket to each .app or .dmg, then let Gatekeeper judge it
  cleanup             delete the keychain and the key files

env:
  MACOS_SIGN_IDENTITY  the identity `keychain` found; `-` signs ad hoc, with no keychain and no timestamp
  MACOS_NOTARIZE=skip  notarize and staple print what they skip and succeed: the dry run of a signing
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
    [ "${MACOS_SIGN_IDENTITY:-}" = - ]
}

skipped() {
    [ "${MACOS_NOTARIZE:-}" = skip ]
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
    if [ "${GITHUB_REF_TYPE:-}" = tag ]; then
        fail "a release tag ships signed and notarized macOS artefacts, and these secrets are not set: ${missing[*]} (docs/guides/release.md, \"Signing and notarization\")"
    fi
    if [ ${#missing[@]} -lt ${#secrets[@]} ]; then
        fail "the signing secrets are set in part, and these are missing: ${missing[*]}"
    fi
    echo "::notice::no signing secret is set: this run, not a tag, builds the macOS artefacts unsigned"
    publish GITHUB_OUTPUT signed false
}

keychain() {
    local name
    for name in MACOS_CERT_P12_BASE64 MACOS_CERT_PASSWORD APPLE_API_KEY_P8_BASE64; do
        [ -n "${!name:-}" ] || fail "keychain: $name is not set"
    done
    local dir chain password identity
    dir=$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/signing.XXXXXX")
    chmod 700 "$dir"
    chain="$dir/signing.keychain-db"
    password=$(openssl rand -hex 24)
    echo "::add-mask::$password"
    publish GITHUB_ENV MACOS_SIGN_DIR "$dir"
    publish GITHUB_ENV MACOS_SIGN_KEYCHAIN "$chain"

    (umask 077 && printf '%s' "$MACOS_CERT_P12_BASE64" | base64 --decode > "$dir/certificate.p12") ||
        fail "keychain: MACOS_CERT_P12_BASE64 is not base64"
    security create-keychain -p "$password" "$chain"
    security set-keychain-settings -lut 21600 "$chain"
    security unlock-keychain -p "$password" "$chain"
    security import "$dir/certificate.p12" -k "$chain" -f pkcs12 -P "$MACOS_CERT_PASSWORD" -T /usr/bin/codesign >/dev/null ||
        fail "keychain: MACOS_CERT_P12_BASE64 does not open with MACOS_CERT_PASSWORD"
    rm -f "$dir/certificate.p12"
    # invariant: codesign reads the key without a prompt only once the partition list names it; a CI runner has nobody to click Allow
    security set-key-partition-list -S apple-tool:,apple:,codesign: -s -k "$password" "$chain" >/dev/null
    local chains=()
    while IFS= read -r name; do
        name=${name#"${name%%[![:space:]]*}"}
        chains+=("${name//\"/}")
    done < <(security list-keychains -d user)
    security list-keychains -d user -s "$chain" "${chains[@]}"

    identity=$(security find-identity -v -p codesigning "$chain" | awk '/"Developer ID Application: / { print $2; exit }')
    [ -n "$identity" ] || fail "keychain: MACOS_CERT_P12_BASE64 holds no valid Developer ID Application identity with its private key"
    echo "::add-mask::$identity"
    publish GITHUB_ENV MACOS_SIGN_IDENTITY "$identity"

    (umask 077 && printf '%s' "$APPLE_API_KEY_P8_BASE64" | base64 --decode > "$dir/notary.p8") ||
        fail "keychain: APPLE_API_KEY_P8_BASE64 is not base64"
    grep -q 'BEGIN PRIVATE KEY' "$dir/notary.p8" || fail "keychain: APPLE_API_KEY_P8_BASE64 does not decode to a .p8 private key"
    publish GITHUB_ENV APPLE_API_KEY_PATH "$dir/notary.p8"
    echo "keychain: one Developer ID Application identity imported, notary key written"
}

codesign_one() {
    local path=$1 args=(--force --options runtime --sign "$MACOS_SIGN_IDENTITY")
    if adhoc; then
        args+=(--timestamp=none)
    else
        args+=(--timestamp --keychain "$MACOS_SIGN_KEYCHAIN")
    fi
    codesign "${args[@]}" "$path"
}

sign() {
    [ $# -gt 0 ] || usage
    [ -n "${MACOS_SIGN_IDENTITY:-}" ] || fail "sign: no identity; run the keychain step first, or set MACOS_SIGN_IDENTITY=- to sign ad hoc"
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
    for name in APPLE_API_KEY_PATH APPLE_API_KEY_ID APPLE_API_ISSUER_ID; do
        [ -n "${!name:-}" ] || fail "notarize: $name is not set"
    done
    local answer id status
    for file in "$@"; do
        case "$file" in *.zip | *.dmg | *.pkg) ;; *) fail "notarize: $file is not a .zip, .dmg or .pkg, the only shapes the notary service takes" ;; esac
        answer=$(xcrun notarytool submit "$file" --key "$APPLE_API_KEY_PATH" --key-id "$APPLE_API_KEY_ID" --issuer "$APPLE_API_ISSUER_ID" --wait --timeout 30m --output-format json) ||
            fail "notarize: notarytool could not submit $file: $answer"
        id=$(plutil -extract id raw -o - - <<< "$answer")
        status=$(plutil -extract status raw -o - - <<< "$answer")
        if [ "$status" != Accepted ]; then
            xcrun notarytool log "$id" --key "$APPLE_API_KEY_PATH" --key-id "$APPLE_API_KEY_ID" --issuer "$APPLE_API_ISSUER_ID" >&2 || true
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

cleanup() {
    if [ -n "${MACOS_SIGN_KEYCHAIN:-}" ] && [ -e "$MACOS_SIGN_KEYCHAIN" ]; then
        security delete-keychain "$MACOS_SIGN_KEYCHAIN"
    fi
    if [ -n "${MACOS_SIGN_DIR:-}" ]; then
        rm -rf "$MACOS_SIGN_DIR"
    fi
    echo "cleanup: the keychain and the key files of this run are deleted"
}

[ $# -gt 0 ] || usage
step=$1
shift
case "$step" in
    require | keychain | sign | notarize | staple | cleanup) "$step" "$@" ;;
    *) usage ;;
esac

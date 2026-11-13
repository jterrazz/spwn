# outside: sourced by check.sh and check-quick.sh, in the Linux VM only.
# value: `findings <log>...` prints one line per finding of the toolchain's passes, the path first: tsc's
# value: `path(l,c)`, oxlint's `╭─[path:l:c]`, oxfmt's `path (n ms)`, knip's lines under an `Unused …` head, and
# value: every other pass's `path:l:c:`.
# shellcheck shell=bash
findings() {
    sed 's/\x1b\[[0-9;]*m//g' "$@" | awk '
        /^Unused .*\([0-9]+\)$/ { knip = $0; next }
        knip != "" && /^$/ { knip = ""; next }
        knip != "" { print $NF "  knip: " knip; next }
        /^[A-Za-z0-9_.@\/-]+ \([0-9]+ms\)$/ { print $1 "  oxfmt: not formatted"; next }
        /^ *[×!] / { sub(/^ *[×!] +/, ""); rule = substr($0, 1, 110); next }
        /╭─\[/ { at = $0; sub(/.*╭─\[/, "", at); sub(/\].*/, "", at); print at "  " rule; next }
        /^[A-Za-z0-9_.@\/-]+(:[0-9]+:[0-9]+:|\([0-9]+,[0-9]+\): error)/ { print substr($0, 1, 200) }
    '
}
# value: `summary <log>...` lists the window's findings, counts the root's apart, and leaves the root's count in
# value: root_findings for the caller's last line; it returns 0 when apps/window has none.
summary() {
    local all window rest
    all=$(findings "$@")
    window=$(grep -E '^apps/window/' <<<"$all" || true)
    rest=$(grep -vE '^apps/window/' <<<"$all" | grep -v '^$' || true)
    root_findings=$(grep -c . <<<"$rest" || true)
    printf 'window findings: %s\n' "$(grep -c . <<<"$window" || true)"
    [ -z "$window" ] || sed 's/^/  /' <<<"$window"
    printf 'root findings (outside apps/window): %s\n' "$root_findings"
    [ -z "$rest" ] || sed -E 's/[:(].*//' <<<"$rest" | sort | uniq -c | sort -rn | head -12 | sed 's/^/  /'
    [ -z "$window" ]
}

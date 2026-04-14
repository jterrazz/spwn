#!/usr/bin/env bash
# Install bin/spwn into ~/.local/bin (or $INSTALL_DIR if set) and make
# Sure that dir is on PATH.
#
# Expected precondition: bin/spwn already built (run `make build` first).
# This script is invoked by `make install`.

set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
PATH_EXPORT='export PATH="$HOME/.local/bin:$PATH"'

BIN="bin/spwn"
if [ ! -x "$BIN" ]; then
    echo "error: $BIN not found — run 'make build' first" >&2
    exit 1
fi

mkdir -p "$INSTALL_DIR"
cp "$BIN" "$INSTALL_DIR/spwn"
chmod +x "$INSTALL_DIR/spwn"

# Ad-hoc codesign on macOS so gatekeeper doesn't nag. Ignore failures
# (non-darwin, missing codesign, unsigned identity, etc.).
codesign -s - "$INSTALL_DIR/spwn" 2>/dev/null || true

# Ensure INSTALL_DIR is on PATH. If not, append an export line to the
# First rc file we find; fall back to ~/.profile.
case ":$PATH:" in
    *":$INSTALL_DIR:"*)
        # Already on PATH, nothing to do.
        ;;
    *)
        added=false
        for rc in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile"; do
            if [ -f "$rc" ]; then
                if ! grep -q '.local/bin' "$rc" 2>/dev/null; then
                    {
                        echo ""
                        echo "# Added by spwn (make install)"
                        echo "$PATH_EXPORT"
                    } >> "$rc"
                    echo "  Added ~/.local/bin to PATH in $(basename "$rc")"
                fi
                added=true
                break
            fi
        done
        if [ "$added" = false ]; then
            {
                echo ""
                echo "# Added by spwn (make install)"
                echo "$PATH_EXPORT"
            } >> "$HOME/.profile"
            echo "  Added ~/.local/bin to PATH in .profile"
        fi
        ;;
esac

echo ""
echo "  ✓ spwn installed to $INSTALL_DIR/spwn"
echo ""
echo "  Get started:"
echo "    spwn init"
echo "    spwn agent new neo"
echo "    spwn up --agent neo -w ."
echo ""

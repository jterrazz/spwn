#!/usr/bin/env bash

set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
PATH_EXPORT='export PATH="$HOME/.local/bin:$PATH"'

BIN=".artifacts/go/spwn"
if [ ! -x "$BIN" ]; then
    echo "error: $BIN not found; run 'make build' first" >&2
    exit 1
fi

mkdir -p "$INSTALL_DIR"
cp "$BIN" "$INSTALL_DIR/spwn"
chmod +x "$INSTALL_DIR/spwn"

# outside: an ad-hoc signature keeps Gatekeeper quiet on macOS; elsewhere codesign is absent and the failure is moot
codesign -s - "$INSTALL_DIR/spwn" 2>/dev/null || true

case ":$PATH:" in
    *":$INSTALL_DIR:"*)
        ;;
    *)
        # value: the first shell rc file found gets the export, else ~/.profile
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

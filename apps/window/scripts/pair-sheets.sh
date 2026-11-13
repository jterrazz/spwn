#!/usr/bin/env bash
# outside: ImageMagick (convert, montage) exists in the Linux VM; a Mac never runs this.
# value: usage: pair-sheets.sh <captures> <mockups> <out-dir> [<live>]: each page the window draws beside the
# value: mockup's same page, the window's at the left. <captures> is one world's folder written by capture.ts, stories
# value: by default, whose linux-1 and This Mac, PAIR_MAC (macbook-pro by default), its pairs use; <mockups> holds the
# value: mockup's shot of each pair as <pair>.png, any width, and <live> the shots of a live-tauri.sh session holding
# value: palette.png and destroy.png; the dialogs and clicked views are capture.ts's OPENED shots, and a0-first-home
# value: is the bare-mac world's Home, beside <captures>. Each side is its whole shot, every pixel of its height, scaled
# value: from the window's 1280 wide to 640, both sides aligned at the top; a pair is <out-dir>/<pair>.png, and three
# value: pairs stacked are <out-dir>/sheet-<n>.png. A pair whose shot is missing is named and left out.
set -euo pipefail
captures=$1 mockups=$2 out=$3 live=${4:-}
mac=${PAIR_MAC:-macbook-pro}
mkdir -p "$out"
PAIRS=(
    a0-first-home:../bare-mac/home a1-home:home a2-inbox:inbox a3-storage:storage
    b1-wsov:linux-1-overview b2-wsact:linux-1-activity b3-wspkg:linux-1-packages
    b4-wsports:linux-1-ports b5-wsacc:linux-1-accounts b6-wsstor:linux-1-storage
    b7-wslogs:linux-1-logs c1-macov:"$mac"-overview
    c2-macact:"$mac"-activity c3-macpkg:"$mac"-packages
    c4-macports:"$mac"-ports c5-macacc:"$mac"-accounts
    c6-macstor:"$mac"-storage c7-maclogs:"$mac"-logs
    d1-files:machine-files d3-catapps:catalog-apps d4-catset:catalog-settings
    d5-itemapp:catalog-item-node d6-itemset:catalog-item-agent-hooks
    e1-palette:live/palette e2-destroy:live/destroy
    f1-find:dialog-find f2-nmhome:dialog-new-machine-home f3-nmmac:dialog-new-machine-mac
    f4-nmfile:dialog-new-machine-file f5-delete:dialog-delete-linux-1 f6-restart:dialog-restart-infra
    g1-plan:linux-1-packages-plan g2-tofile:linux-1-packages-to-file
)
side() {
    convert "$1" -resize 640x "$2"
}
made=()
for pair in "${PAIRS[@]}"; do
    name=${pair%%:*} shot=${pair#*:}
    if [ "${shot#live/}" != "$shot" ]; then
        shot=${live:+$live/${shot#live/}.png}
    else
        shot=$captures/$shot.png
    fi
    if [ -z "$shot" ] || [ ! -f "$shot" ] || [ ! -f "$mockups/$name.png" ]; then
        echo "left out $name: no ${shot:-live shot} or no $mockups/$name.png"
        continue
    fi
    side "$shot" "$out/.$name-window.png"
    side "$mockups/$name.png" "$out/.$name-mockup.png"
    convert -background '#0c0c0c' "$out/.$name-window.png" "$out/.$name-mockup.png" +append "$out/$name.png"
    rm -f "$out/.$name-window.png" "$out/.$name-mockup.png"
    made+=("$out/$name.png")
done
sheet=1
for ((at = 0; at < ${#made[@]}; at += 3)); do
    convert "${made[@]:at:3}" -append "$out/sheet-$sheet.png"
    sheet=$((sheet + 1))
done
echo "${#made[@]} pairs, $((sheet - 1)) sheets in $out"

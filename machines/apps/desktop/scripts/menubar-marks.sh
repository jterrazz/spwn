#!/usr/bin/env bash
# outside: a Linux VM only.
# value: writes src/presentation/ui/menubar-marks.ts from the menu bar app's apps/menubar/scripts/symbols.swift, after a change
# value: to its marks; src/presentation/ui/__tests__/menubar-marks.test.ts fails until it runs.
set -euo pipefail
# shellcheck source=bench-env.sh
source "$(dirname "$0")/bench-env.sh"
cd "$window"
node --input-type=module -e '
import { readFileSync, writeFileSync } from "node:fs";
import { fileOf, marksOf } from "./scripts/menubar-marks.ts";
writeFileSync("src/presentation/ui/menubar-marks.ts", fileOf(marksOf(readFileSync("../menubar/scripts/symbols.swift", "utf8"))));
'
echo "menubar-marks: src/presentation/ui/menubar-marks.ts written"

#!/usr/bin/env bash
# pre-spawn hook — runs before the startup world initializes.
# Place setup logic, environment checks, or logging here.

set -euo pipefail

echo "[startup] world initializing — $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "[startup] agents: ceo, devops, analyst"
echo "[startup] pre-spawn hook complete"

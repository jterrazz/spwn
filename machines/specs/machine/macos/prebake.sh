#!/bin/bash
# invariant: the test-only image holds Homebrew and nothing of station; it is
# rebuilt only when the catalogue hash changes.
set -euo pipefail
if [ ! -x /opt/homebrew/bin/brew ]; then
  NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
fi
/opt/homebrew/bin/brew --version

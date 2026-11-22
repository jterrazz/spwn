#!/bin/bash
set -uo pipefail
. "$(dirname "$0")/env.sh"
wt=${1:-}; label=${2:-}
[ -d "$wt" ] && [ -n "$label" ] || { echo "usage: darwin-build.sh <worktree> <label>" >&2; exit 2; }
[ -d "$wt/machines/apps/engine" ] && wt="$wt/machines"
[ ! -e $RUN/vm-hold/all ] || { echo "FAIL darwin-build: $RUN/vm-hold/all present"; exit 1; }
sub=$(git -C "$wt" rev-parse --show-prefix); sha=$(git -C "$wt" rev-parse HEAD); sha9=${sha:0:9}
ver=${DARWIN_VERSION:-v3.1.0-local-$sha9}
gover=$(awk '$1=="go"{print $2; exit}' "$wt/go.work")
case "$gover" in 1.27.3) gosum=dd3908cf20959d798c1e06ebe5b72213f0505ce01373b40ce3110975a6cd99c2 ;; *) echo "FAIL darwin-build: no pinned checksum for go $gover (add it from go.dev/dl/?mode=json)"; exit 1 ;; esac
log=$RUN/inst-darwin-$label.log; : > $log
W=$RUN/inst-darwin-$label-work; rm -rf $W; mkdir -p $W
tart() { TART_NO_AUTO_PRUNE=1 $HOME/.local/bin/tart "$@"; }
base=$(tart list --quiet --source local | grep "^${GATE_TART_BASE:-station-e2e-xcode-}" | head -1); vm=station-e2e-darwin-$label
[ -n "$base" ] || { echo "FAIL darwin-build: no station-e2e-xcode base"; exit 1; }
opts=(-F /dev/null -o IdentitiesOnly=yes -o IdentityAgent=none -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -o ConnectTimeout=5 -o ServerAliveInterval=15)
tp=
cleanup() { tart stop $vm >/dev/null 2>&1; tart delete $vm >/dev/null 2>&1; [ -z "$tp" ] || kill $tp 2>/dev/null; echo "cleaned $vm; base $base kept" >> $log; }
trap cleanup EXIT
die() { echo "FAIL darwin-build $sha9: $*" | tee -a $log; exit 1; }
tart list --quiet --source local | grep -qx $vm && die "$vm exists"
tart clone $base $vm >> $log 2>&1 || die clone
tart run --no-graphics $vm > $W/tart.log 2>&1 & tp=$!
ip=$(tart ip $vm --wait 180) || die ip
key=$W/id; ssh-keygen -q -t ed25519 -N '' -f $key
printf '#!/bin/sh\necho admin\n' > $W/ap; chmod 700 $W/ap
n=0; until SSH_ASKPASS=$W/ap SSH_ASKPASS_REQUIRE=force DISPLAY=none ssh "${opts[@]/BatchMode=yes/BatchMode=no}" -o PubkeyAuthentication=no admin@$ip "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys" < $key.pub 2>/dev/null; do n=$((n+1)); [ $n -lt 60 ] || die ssh; sleep 2; done
vmx() { ssh -n "${opts[@]}" -i $key admin@$ip "$1" >> $log 2>&1; }
vmx 'rm -rf ~/station && git init -q -b scratch ~/station' || die init
GIT_SSH_COMMAND="ssh ${opts[*]} -i $key" git -C "$wt" push -q admin@$ip:station "$sha:refs/heads/run" >> $log 2>&1 || die push
vmx 'cd ~/station && git checkout -q run' || die checkout
vmx "set -e; cd ~; curl -fsSLo go.tgz https://go.dev/dl/go$gover.darwin-arm64.tar.gz; echo '$gosum  go.tgz' | shasum -a 256 -c -; mkdir -p gotool && tar -xzf go.tgz -C gotool; ~/gotool/go/bin/go version" || die "go $gover install or checksum"
env='export PATH=$HOME/gotool/go/bin:$PATH MACOSX_DEPLOYMENT_TARGET=14.0 CGO_ENABLED=1 GOOS=darwin GOARCH=arm64; sdk=$(xcrun --sdk macosx --show-sdk-path); export CC="clang -arch arm64 -isysroot $sdk -mmacosx-version-min=$MACOSX_DEPLOYMENT_TARGET"; cd ~/station/'"$sub"
vmx "set -e; $env; if grep -q '^enclave:' Makefile; then make enclave; fi" || die "make enclave"
vmx "set -e; $env; for d in \$(go list -m -f '{{.Dir}}'); do echo module \$d; (cd \$d && go build ./... && go vet ./...) || exit 1; done" || die "go build ./... or go vet (per workspace module)"
vmx "set -e; $env; go build -ldflags '-X github.com/jterrazz/studio-station/apps/engine/wire.version=$ver' -o ~/station-bin ./apps/engine/cmd/station; go version -m ~/station-bin | grep -q CGO_ENABLED=1; lipo -archs ~/station-bin; vtool -show-build ~/station-bin | grep minos; ~/station-bin --version; otool -L ~/station-bin; file ~/station-bin; shasum -a 256 ~/station-bin" || die "release build"
vmx "set -e; $env; command -v cargo >/dev/null || [ -x \$HOME/.cargo/bin/cargo ] || curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --profile minimal >/dev/null 2>&1; . \$HOME/.cargo/env; cd apps/desktop/src-tauri; cargo test --bin spwn-window" || die "cargo test of the window (apps/desktop/src-tauri)"
vmx "set -e; $env; ../scripts/sign-macos-test.sh" || die "scripts/sign-macos-test.sh"
if [ -n "${DARWIN_STEPS:-}" ]; then vmx "set -e; $env; $DARWIN_STEPS" || die "DARWIN_STEPS"; fi
if [ -n "${DARWIN_OUT:-}" ]; then scp "${opts[@]}" -i $key admin@$ip:station-bin "$DARWIN_OUT" >> $log 2>&1 || die scp; fi
echo "PASS darwin-build $sha ($ver)" | tee -a $log

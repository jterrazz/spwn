#!/usr/bin/env bash
# outside: a Linux VM only.
# value: no argument; every step is skipped when what it makes already stands, so a second run is a few seconds.
set -euo pipefail
# shellcheck source=bench-env.sh
source "$(dirname "$0")/bench-env.sh"
cd "$root"
step() { printf '%-12s %s\n' "$1" "$2"; }
start=$(date +%s)

# value: the stamp `make install` keeps: `npm ci` runs again once the root manifest or the lockfile is newer.
if [ ! node_modules/.install -nt package.json ] || [ ! node_modules/.install -nt package-lock.json ]; then
    if ! npm ci --prefer-offline --no-audit --no-fund >"$bench/install.log" 2>&1; then
        tail -20 "$bench/install.log" >&2
        echo "bench-setup: npm ci failed; log $bench/install.log" >&2
        exit 1
    fi
    touch node_modules/.install
    step install "node_modules by npm ci, cache $npm_config_cache"
fi

if ! ls "$PLAYWRIGHT_BROWSERS_PATH" 2>/dev/null | grep -q chromium_headless_shell; then
    (cd "$window" && npx playwright install chromium-headless-shell >"$bench/playwright.log" 2>&1)
    step playwright "chromium in $PLAYWRIGHT_BROWSERS_PATH"
fi

if [ -s "$HOME/.cargo/env" ]; then
    # shellcheck disable=SC1091
    source "$HOME/.cargo/env"
fi
if command -v rustup >/dev/null && ! rustup component list --installed | grep -q '^clippy'; then
    rustup component add rustfmt clippy >"$bench/rustup.log" 2>&1
    step rustup "rustfmt and clippy added to $(rustup show active-toolchain | cut -d' ' -f1)"
fi

# hazard: under vm.overcommit_memory=0 a reservation past RAM and swap is refused, and oxc's allocator pool asks for
# hazard: such reservations: oxlint and knip run under LD_PRELOAD of this shim, which marks them MAP_NORESERVE.
if [ ! -s "$NORESERVE" ]; then
    mkdir -p "$(dirname "$NORESERVE")"
    cc -shared -fPIC -O2 -o "$NORESERVE" -x c - -ldl <<'C'
#define _GNU_SOURCE
#include <dlfcn.h>
#include <errno.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <sys/mman.h>
typedef void *(*mmap_fn)(void *, size_t, int, int, int, off_t);
#define HUGE ((size_t)1 << 30)
#define SLOTS 256
static void *slot_ptr[SLOTS];
static void *slot_base[SLOTS];
static size_t slot_length[SLOTS];
static mmap_fn real_mmap(void) {
    static mmap_fn real;
    if (!real) real = (mmap_fn)dlsym(RTLD_NEXT, "mmap");
    return real;
}
void *mmap(void *addr, size_t length, int prot, int flags, int fd, off_t offset) {
    if ((flags & MAP_ANONYMOUS) && length >= HUGE) flags |= MAP_NORESERVE;
    return real_mmap()(addr, length, prot, flags, fd, offset);
}
void *mmap64(void *addr, size_t length, int prot, int flags, int fd, off_t offset) __attribute__((alias("mmap")));
static void *huge_aligned(size_t alignment, size_t size) {
    size_t length = size + alignment;
    char *base = real_mmap()(NULL, length, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANONYMOUS | MAP_NORESERVE, -1, 0);
    if (base == MAP_FAILED) return NULL;
    uintptr_t aligned = ((uintptr_t)base + alignment - 1) & ~(uintptr_t)(alignment - 1);
    for (int i = 0; i < SLOTS; i++) {
        void *expected = NULL;
        if (__atomic_compare_exchange_n(&slot_ptr[i], &expected, (void *)aligned, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) {
            slot_base[i] = base;
            slot_length[i] = length;
            return (void *)aligned;
        }
    }
    munmap(base, length);
    return NULL;
}
int posix_memalign(void **out, size_t alignment, size_t size) {
    static int (*real)(void **, size_t, size_t);
    if (size >= HUGE) {
        void *p = huge_aligned(alignment < 16 ? 16 : alignment, size);
        if (!p) return ENOMEM;
        *out = p;
        return 0;
    }
    if (!real) real = (int (*)(void **, size_t, size_t))dlsym(RTLD_NEXT, "posix_memalign");
    return real(out, alignment, size);
}
void *malloc(size_t size) {
    static void *(*real)(size_t);
    if (size >= HUGE) return huge_aligned(16, size);
    if (!real) real = (void *(*)(size_t))dlsym(RTLD_NEXT, "malloc");
    return real(size);
}
void *calloc(size_t count, size_t size) {
    static void *(*real)(size_t, size_t);
    if (size != 0 && count >= HUGE / size) return huge_aligned(16, count * size);
    if (!real) real = (void *(*)(size_t, size_t))dlsym(RTLD_NEXT, "calloc");
    return real(count, size);
}
void free(void *ptr) {
    static void (*real)(void *);
    if (ptr) {
        for (int i = 0; i < SLOTS; i++) {
            if (slot_ptr[i] == ptr) {
                munmap(slot_base[i], slot_length[i]);
                __atomic_store_n(&slot_ptr[i], NULL, __ATOMIC_SEQ_CST);
                return;
            }
        }
    }
    if (!real) real = (void (*)(void *))dlsym(RTLD_NEXT, "free");
    real(ptr);
}
C
    step shim "$NORESERVE"
fi

if ! command -v tesseract >/dev/null; then
    if sudo -n apt-get install -y -qq tesseract-ocr >"$bench/tesseract.log" 2>&1; then
        step tesseract "installed, for live-tauri.sh's see step"
    else
        step tesseract "missing and not installable without a password: live-tauri.sh's see step fails"
    fi
fi

step world "$(python3 "$scripts/world.py" 2>"$bench/world.log")"
step done "in $(($(date +%s) - start)) s"

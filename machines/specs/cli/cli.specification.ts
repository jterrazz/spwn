import { specification } from '@jterrazz/test';
import { execSync } from 'node:child_process';
import { existsSync, readdirSync, renameSync, statSync } from 'node:fs';
import { resolve } from 'node:path';
import { afterAll } from 'vitest';

const REPO_ROOT = resolve(import.meta.dirname, '../..');
const STATION_BIN = resolve(REPO_ROOT, '.artifacts/go/station-test');
const SOURCES = [resolve(REPO_ROOT, 'apps/engine'), resolve(REPO_ROOT, 'catalog')];
const VM_SSHD = resolve(import.meta.dirname, '__fixtures__/vm-sshd');
const WORLDS = resolve(REPO_ROOT, 'specs/usecases/worlds');

// hazard: a stale binary silently tests old behaviour, so the suite rebuilds when any
// file under `apps/engine/` or `catalog/` is newer than the binary: the embedded dotfiles and recipes are
// build inputs as much as the Go sources are.
function binaryIsStale(): boolean {
    if (!existsSync(STATION_BIN)) {
        return true;
    }
    const binModifiedMs = statSync(STATION_BIN).mtimeMs;
    for (const source of SOURCES) {
        for (const entry of readdirSync(source, { recursive: true, withFileTypes: true })) {
            if (
                entry.isFile() &&
                statSync(resolve(entry.parentPath, entry.name)).mtimeMs > binModifiedMs
            ) {
                return true;
            }
        }
    }
    return false;
}

// invariant: `make test` builds the binary before the runner starts and says so
// (STATION_SPEC_BUILT), so every document of a gate runs the one binary it built, even
// when `apps/engine/` or `catalog/` changes under the run; a direct run that finds it stale builds it beside
// and renames it into place, so a worker never runs a binary another worker is still
// writing.
if (process.env.STATION_SPEC_BUILT !== '1' && binaryIsStale()) {
    const version = execSync('git describe --tags --always --dirty', { cwd: REPO_ROOT })
        .toString()
        .trim();
    const building = `${STATION_BIN}.${process.pid}`;
    execSync(
        `go build -tags world -ldflags "-X github.com/jterrazz/studio-station/apps/engine/wire.version=${version}" -o ${building} ./apps/engine/cmd/station`,
        { cwd: REPO_ROOT, stdio: 'inherit' },
    );
    renameSync(building, STATION_BIN);
}

// invariant: no run reaches the real machine. A run's environment is the defaults
// below, what its document states, and of the runner's own variables only those
// INHERITED names: every other one is unset, so a cache folder the runner exports
// (GOMODCACHE, GOCACHE, npm_config_cache) is never the folder a run's stub empties,
// and the CI runner's XDG_CONFIG_HOME never becomes one folder of preferences every
// run shares. HOME is the run's own directory; PATH
// holds only the fixture's stubs and STATION_SYSTEM_DIRS adds no system directory, so
// an unstubbed tool is "not found" alike on a Mac and on the Linux CI runner;
// STATION_SYSTEM_ROOT reads /Applications, /opt/homebrew, /Library and /usr/local under
// the run's own `root/`, so the owner's apps are neither listed nor sized nor run;
// STATION_POWER_STUB points Power at the run's own `power/` (absent: no Power here), so no
// run takes a real assertion or touches the clamshell; STATION_KERNEL_STUB points the
// Mac's counters and process tables at the run's own `kernel/` (absent: the kernel
// answers nothing), so no run reads the real machine's activity; the
// machine kind is pinned to host unless a document says otherwise; GitHub is a closed
// port unless a document serves its stub; a VM's sshd is the served stub vm-sshd.
// value: one env set per world of specs/usecases/worlds/, named after it: a
// use-case document that writes `env: two-macs` runs the spec build over a copy of
// that world, `world.json` in its own directory, read and written by every run.
// invariant: AR-02: with STATION_SPEC_HYPERVISOR set (`make test` sets it to vz for
// its second pass over the machines and engine/keep suites), every world's Macs run
// the fake of that hypervisor instead of tart's; `two-macs-vz` is two-macs over vz.
const hypervisor = process.env.STATION_SPEC_HYPERVISOR ?? '';
// value: the temp root @jterrazz/test points TMPDIR, TMP and TEMP at, its run's own.
const INHERITED = new Set(['TEMP', 'TMP', 'TMPDIR']);
const runnerUnset = Object.fromEntries(
    Object.keys(process.env)
        .filter((name) => !INHERITED.has(name))
        .map((name) => [name, null]),
);
// value: a world runs in a terminal with the station block (STATION_TERMINAL set);
// `no-block` is two-macs from a terminal without it.
const world = (seed: string, over: string, block = true) => ({
    STATION_WORLD: '$WORKDIR/world.json',
    STATION_WORLD_HYPERVISOR: over,
    STATION_WORLD_SEED: resolve(WORLDS, seed),
    ...(block ? { STATION_TERMINAL: 'spec-terminal' } : {}),
});
const worlds = {
    ...Object.fromEntries(
        readdirSync(WORLDS)
            .filter((name) => name.endsWith('.json'))
            .map((name) => [name.replace(/\.json$/u, ''), world(name, hypervisor)]),
    ),
    'no-block': world('two-macs.json', hypervisor, false),
    'two-macs-vz': world('two-macs.json', 'vz'),
};

export const { cleanup, cli } = await specification.cli(STATION_BIN, {
    defaults: {
        ...runnerUnset,
        COLUMNS: '80',
        HOME: '$WORKDIR',
        NO_COLOR: '1',
        PATH: '$WORKDIR/bin',
        STATION_GITHUB_API_URL: 'http://127.0.0.1:9',
        STATION_MACHINE_KIND: 'mac',
        STATION_KERNEL_STUB: '$WORKDIR/kernel',
        STATION_POWER_STUB: '$WORKDIR/power',
        STATION_SYSTEM_DIRS: '',
        STATION_SYSTEM_ROOT: '$WORKDIR/root',
        TZ: 'UTC',
    },
    // value: a document that needs a terminal runs `terminal "$STATION"`, the binary on a
    // pseudo-terminal its fixture's driver opens, since a run is never given a TTY, and
    // names it a 256-colour terminal that draws true colour; one that installs runs `$INSTALL_SH` from a local file through its one test seam.
    env: {
        ...worlds,
        engine: { STATION: STATION_BIN },
        install: { INSTALL_SH: resolve(REPO_ROOT, 'scripts/install.sh'), STATION: STATION_BIN },
        terminal: { COLORTERM: 'truecolor', STATION: STATION_BIN, TERM: 'xterm-256color' },
    },
    serve: {
        github: {
            command: `node ${resolve(import.meta.dirname, '__setup__/github-api.ts')}`,
            env: 'STATION_GITHUB_API_URL',
            ready: /listening on port (?<port>\d+)/u,
            url: (port) => `http://127.0.0.1:${port}`,
        },
        sshd: {
            command: `STATION_TEST_BIN=${STATION_BIN} go run ${VM_SSHD}`,
            env: 'STATION_GUEST_SSH_PORT',
            ready: /listening on port (?<port>\d+)/u,
            url: String,
        },
    },
});

afterAll(cleanup);

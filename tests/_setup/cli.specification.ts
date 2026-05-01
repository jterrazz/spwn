import {
    type CliResult,
    command,
    type SeedHandlerContext,
    spec as specRunner,
} from '@jterrazz/test';
import {
    copyFileSync,
    mkdirSync,
    readdirSync,
    readFileSync,
    statSync,
    writeFileSync,
} from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { parse, stringify } from 'yaml';

/**
 * CLI specification runner for spwn.
 *
 * Each spec call gets a fresh temporary directory. `.project('name')`
 * copies `tests/_fixtures/<name>/` into that directory before exec.
 * The spwn binary is run with that directory as its working directory.
 *
 * Seed handlers route per-test overlay files into the right place:
 *   .seed('spwn.yaml/foo.yaml')      → merged into spwn.yaml
 *   .seed('agent/neo/journal/x.md')  → copied under spwn/agents/neo/journal/
 *   .seed('state/foo.json')          → merged into .spwn/state.json
 *   .seed('activity/foo.jsonl')      → appended to .spwn/activity.jsonl
 *
 * Source fixtures live next to the test file under seeds/<path>; the
 * framework reads them and routes by leading path segment.
 */
const SPWN_BIN = resolve(import.meta.dirname, '../../bin/spwn');

// Blanket-disable live credential validation in the test process so
// Child `spwn` invocations don't hit real provider APIs. Each test
// That actually wants to exercise validation can unset this via
// `.env({ SPWN_SKIP_AUTH_VALIDATION: null })`. Rationale: the auth
// Dashboard + spawn pre-flight both run validations; a parallel e2e
// Suite of ~30 tests hitting Anthropic /oauth/usage burns rate
// Limits and flakes on 429s. See packages/auth/validate_cache.go.
process.env.SPWN_SKIP_AUTH_VALIDATION = '1';

const PROJECT_PATH_PLACEHOLDER = '<PROJECT>';

/**
 * Strip ANSI colour escapes. The codestyle writer emits them unconditionally
 * so stdout from `spwn check` contains SGR sequences in CI too.
 */
function stripAnsi(input: string): string {
    // eslint-disable-next-line no-control-regex
    return input.replace(/\u001b\[[0-9;]*m/g, '');
}

/**
 * Normalise test-run specific noise before comparing to a stored fixture.
 *
 * Every `spec(...).project(...).exec(...)` call materialises the fixture
 * in a fresh `os.tmpdir()/spec-*` directory — macOS resolves that to
 * `/private/var/folders/...`, Linux to `/tmp/...`. Collapse any of those
 * forms into a single `<PROJECT>` token so stored snapshots stay portable
 * across machines.
 */
function normalise(actual: string): string {
    let out = stripAnsi(actual);
    out = out.replace(
        /(?:\/private)?\/(?:var\/folders\/[^\s/]+\/[^\s/]+\/T|tmp)\/[A-Za-z0-9._-]+/g,
        PROJECT_PATH_PLACEHOLDER,
    );
    return out;
}

function copyTree(srcPath: string, dstPath: string): void {
    const srcStat = statSync(srcPath);
    if (!srcStat.isDirectory()) {
        mkdirSync(dirname(dstPath), { recursive: true });
        copyFileSync(srcPath, dstPath);
        return;
    }
    mkdirSync(dstPath, { recursive: true });
    for (const entry of readdirSync(srcPath)) {
        copyTree(join(srcPath, entry), join(dstPath, entry));
    }
}

/**
 * `command()` mode always produces a `CliResult`, but the framework's
 * generic `spec()` signature widens the result to a union. Narrow it
 * once at the runner level so tests get the full CliResult shape
 * (including `.container(...)` when the docker option is on) without
 * per-call casts. Followup: tighten upstream so this cast goes away.
 */
type CliBuilder = {
    project(name: string): CliBuilder;
    seed(path: string): CliBuilder;
    env(env: Record<string, null | string>): CliBuilder;
    exec(args: string | string[]): CliBuilder;
    spawn(args: string, options: { waitFor: string; timeout: number }): CliBuilder;
    run(): Promise<CliResult>;
};

const seedHandlers = {
    'spwn.yaml/': (ctx: SeedHandlerContext, fragmentPath: string) => {
        const fragment = parse(readFileSync(fragmentPath, 'utf8')) as Record<string, unknown>;
        const targetPath = join(ctx.cwd, 'spwn.yaml');
        const target = parse(readFileSync(targetPath, 'utf8')) as Record<string, unknown>;
        const merged: Record<string, unknown> = { ...target, ...fragment };
        if (
            target.worlds &&
            fragment.worlds &&
            typeof target.worlds === 'object' &&
            typeof fragment.worlds === 'object'
        ) {
            merged.worlds = {
                ...(target.worlds as Record<string, unknown>),
                ...(fragment.worlds as Record<string, unknown>),
            };
        }
        writeFileSync(targetPath, stringify(merged));
    },
    'agent/': (ctx: SeedHandlerContext, fragmentPath: string) => {
        const seedsRoot = fragmentPath.split('/seeds/agent/')[1];
        if (!seedsRoot) {
            throw new Error(`unexpected seed path shape: ${fragmentPath}`);
        }
        const dst = join(ctx.cwd, 'spwn', 'agents', seedsRoot);
        copyTree(fragmentPath, dst);
    },
    'state/': (ctx: SeedHandlerContext, fragmentPath: string) => {
        const fragment = JSON.parse(readFileSync(fragmentPath, 'utf8')) as Record<string, unknown>;
        const targetPath = join(ctx.cwd, '.spwn', 'state.json');
        mkdirSync(dirname(targetPath), { recursive: true });
        let target: Record<string, unknown> = {};
        try {
            target = JSON.parse(readFileSync(targetPath, 'utf8')) as Record<string, unknown>;
        } catch {
            // First write — leave target empty
        }
        writeFileSync(targetPath, JSON.stringify({ ...target, ...fragment }, null, 2));
    },
    'activity/': (ctx: SeedHandlerContext, fragmentPath: string) => {
        const lines = readFileSync(fragmentPath, 'utf8');
        const targetPath = join(ctx.cwd, '.spwn', 'activity.jsonl');
        mkdirSync(dirname(targetPath), { recursive: true });
        writeFileSync(targetPath, lines, { flag: 'a' });
    },
    'lock/': (ctx: SeedHandlerContext, fragmentPath: string) => {
        // Route `.seed('lock/.up.neo.lock')` into `.spwn/.up.neo.lock` so tests can exercise the per-world up-lock without needing real concurrency. Strip everything up to the last `lock/` segment to mirror how the `agent/` handler works.
        const segments = fragmentPath.split('/seeds/lock/');
        const rel = segments[1];
        if (!rel) {
            throw new Error(`unexpected seed path shape: ${fragmentPath}`);
        }
        const targetPath = join(ctx.cwd, '.spwn', rel);
        mkdirSync(dirname(targetPath), { recursive: true });
        const data = readFileSync(fragmentPath);
        writeFileSync(targetPath, data);
    },
};

/**
 * Single spwn specification runner. CLI-only tests use `.exec(...)` and
 * reach for stdout/stderr/file accessors. Tests that need container
 * assertions add `await using` and call `.container(name)` — the first
 * access lazily queries Docker, CLI-only tests never touch it.
 *
 * Container lookup uses `sh.spwn.world.config` (the manifest-declared
 * world key) because spwn only sets `sh.spwn.world.name` when the user
 * assigns an explicit display name — empty for fixture-declared worlds.
 * The per-run test id flows through `SPWN_TEST_LABEL` and lands on every
 * container as `sh.spwn.test.run` (see packages/world/internal/labels).
 */
const rawRunner = await specRunner(command(SPWN_BIN), {
    docker: {
        envVar: 'SPWN_TEST_LABEL',
        nameLabel: 'sh.spwn.world.config',
        testRunLabel: 'sh.spwn.test.run',
    },
    root: '../_fixtures',
    seedHandlers,
    transform: normalise,
});

export const spec = rawRunner as unknown as (label: string) => CliBuilder;

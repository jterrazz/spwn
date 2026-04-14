import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';

/**
 * Spwn-side fixture snapshotting helpers.
 *
 * @jterrazz/test now ships StreamAccessor / JsonAccessor with a built-in
 * .toMatchFixture(), but spwn needs ANSI stripping + temp-path
 * normalisation BEFORE the comparison runs (the codestyle writer emits
 * SGR sequences and every spec materialises in a fresh temp dir whose
 * absolute path leaks into output). These helpers wrap the raw text
 * with the spwn-specific transforms, then delegate to the upstream
 * file-snapshot semantics — same `JTERRAZZ_TEST_UPDATE=1` contract.
 *
 * Convention:
 *   tests/e2e/cli/<feature>/<feature>.e2e.test.ts
 *   tests/e2e/cli/<feature>/expected/stdout/<name>.txt
 *   tests/e2e/cli/<feature>/expected/json/<name>.json
 */

const UPDATE = process.env.JTERRAZZ_TEST_UPDATE === '1';

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
 * in a fresh `os.tmpdir()/spec-*` directory, so the literal temp path
 * leaks into any output that prints absolute paths (e.g. the checker
 * header). Replace it with a stable placeholder so the fixture stays
 * human-readable and reviewable.
 */
function normalise(actual: string): string {
    let out = stripAnsi(actual);

    /*
     * Spec runner materialises each fixture under `$TMPDIR/spec-*` —
     * macOS resolves that to `/private/var/folders/...`, Linux to
     * `/tmp/...`. Collapse any of those forms into a single `<PROJECT>`
     * token so stored snapshots stay portable across machines.
     */
    out = out.replace(
        /(?:\/private)?\/(?:var\/folders\/[^\s/]+\/[^\s/]+\/T|tmp)\/[A-Za-z0-9._-]+/g,
        PROJECT_PATH_PLACEHOLDER,
    );

    return out;
}

function resolveFixture(
    testFilePath: string,
    kind: 'filesystem' | 'json' | 'stdout',
    name: string,
    ext: string,
): string {
    return join(dirname(testFilePath), 'expected', kind, `${name}.${ext}`);
}

function writeFixture(path: string, contents: string): void {
    mkdirSync(dirname(path), { recursive: true });
    writeFileSync(path, contents);
}

export interface StdoutMatcher {
    toMatchFixture(name: string): Promise<void>;
}

/** Read text from either a raw string or a `@jterrazz/test` accessor. */
function asText(actual: string | { text: string } | { toString(): string }): string {
    if (typeof actual === 'string') {
        return actual;
    }
    if ('text' in actual && typeof actual.text === 'string') {
        return actual.text;
    }
    return actual.toString();
}

export function stdoutMatcher(
    testFilePath: string,
    actual: string | { text: string },
): StdoutMatcher {
    return {
        async toMatchFixture(name: string) {
            const path = resolveFixture(testFilePath, 'stdout', name, 'txt');
            const normalised = normalise(asText(actual));
            if (UPDATE || !existsSync(path)) {
                writeFixture(path, normalised);
                return;
            }
            const expected = readFileSync(path, 'utf8');
            if (normalised !== expected) {
                throw new Error(
                    `stdout did not match fixture ${name}.txt\n\n` +
                        `Expected (from ${path}):\n${expected}\n\n` +
                        `Actual:\n${normalised}\n\n` +
                        `Re-run with JTERRAZZ_TEST_UPDATE=1 to refresh.`,
                );
            }
        },
    };
}

export interface JsonMatcher {
    toMatchFixture(name: string): Promise<void>;
}

export function jsonMatcher(testFilePath: string, actual: string | { text: string }): JsonMatcher {
    return {
        async toMatchFixture(name: string) {
            const path = resolveFixture(testFilePath, 'json', name, 'json');
            const parsed = JSON.parse(asText(actual));
            const formatted = `${JSON.stringify(parsed, null, 4)}\n`;
            if (UPDATE || !existsSync(path)) {
                writeFixture(path, formatted);
                return;
            }
            const expected = readFileSync(path, 'utf8');
            if (formatted !== expected) {
                throw new Error(
                    `json output did not match fixture ${name}.json\n\n` +
                        `Expected (from ${path}):\n${expected}\n\n` +
                        `Actual:\n${formatted}\n\n` +
                        `Re-run with JTERRAZZ_TEST_UPDATE=1 to refresh.`,
                );
            }
        },
    };
}

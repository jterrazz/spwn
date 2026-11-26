import { execFileSync } from 'node:child_process';
import { mkdtempSync, readdirSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { extname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { brotliCompressSync, constants } from 'node:zlib';
import { describe, expect, test } from 'vitest';

import { BUDGET, KINDS, MEASURES } from '../bundle-budget';
import type { Kind, Measure } from '../bundle-budget';

const WINDOW = fileURLToPath(new URL('../../../', import.meta.url));

const EXTENSIONS: Record<string, Kind> = { '.css': 'css', '.js': 'js', '.woff2': 'fonts' };

type Weights = Record<Kind, Record<Measure, number>>;

function weigh(folder: string): Weights {
    const weights: Weights = {
        css: { brotli: 0, files: 0, raw: 0 },
        fonts: { brotli: 0, files: 0, raw: 0 },
        js: { brotli: 0, files: 0, raw: 0 },
    };
    for (const entry of readdirSync(folder, { recursive: true, withFileTypes: true })) {
        const kind = EXTENSIONS[extname(entry.name)];
        if (kind !== undefined && entry.isFile()) {
            const bytes = readFileSync(join(entry.parentPath, entry.name));
            const packed = brotliCompressSync(bytes, {
                params: { [constants.BROTLI_PARAM_QUALITY]: constants.BROTLI_MAX_QUALITY },
            });
            weights[kind].files += 1;
            weights[kind].raw += bytes.length;
            weights[kind].brotli += packed.length;
        }
    }
    return weights;
}

function growth(weights: Weights): string[] {
    return KINDS.flatMap((kind) =>
        MEASURES.flatMap((measure) => {
            const line = BUDGET[kind][measure];
            const found = weights[kind][measure];
            if (line === undefined || found <= line) {
                return [];
            }
            const unit = measure === 'files' ? 'files' : `B ${measure}`;
            return [
                `the ${kind} grew to ${found.toLocaleString('en')} ${unit}, over its budget of ${line.toLocaleString('en')} (src/composition/bundle-budget.ts)`,
            ];
        }),
    );
}

describe('bundle budget', () => {
    test('the production build weighs no more than its budget in js, css and fonts', () => {
        // Given - the production build, written to a folder of its own
        const out = mkdtempSync(join(tmpdir(), 'bundle-budget-'));
        try {
            execFileSync('npx', ['vite', 'build', '--outDir', out, '--emptyOutDir'], {
                cwd: WINDOW,
                env: { ...process.env, NODE_ENV: 'production' },
                stdio: 'pipe',
            });

            // When - each kind is weighed, raw and brotli, against its line
            const over = growth(weigh(out));

            // Then - no kind grew past its line
            expect(over).toStrictEqual([]);
        } finally {
            rmSync(out, { force: true, recursive: true });
        }
    }, 120_000);
});

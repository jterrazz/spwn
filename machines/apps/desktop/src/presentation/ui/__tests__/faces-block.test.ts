import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { preprocessCSS, resolveConfig } from 'vite';
import { describe, expect, test } from 'vitest';

const DOCUMENT = fileURLToPath(new URL('../document.css', import.meta.url));
const CONFIG = fileURLToPath(new URL('../../../../vite.config.ts', import.meta.url));

describe('faces block', () => {
    test('every face the build draws blocks its text until it is loaded, and none swaps', async () => {
        // Given - the document's stylesheet and the window's own Vite config
        const config = await resolveConfig({ configFile: CONFIG }, 'build');

        // When - the stylesheet goes through the build's CSS pass, its @fontsource imports inlined
        const { code } = await preprocessCSS(readFileSync(DOCUMENT, 'utf8'), DOCUMENT, config);
        const displays = [...code.matchAll(/font-display:\s*(?<display>\w+)/gu)].map(
            (found) => found.groups?.display,
        );

        // Then - Geist and Geist Mono's faces, every one `block`
        expect(code).toMatch(/font-family:\s*["']Geist Mono["']/u);
        expect(displays.length).toBeGreaterThan(0);
        expect(new Set(displays)).toStrictEqual(new Set(['block']));
    });
});

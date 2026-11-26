import { readdirSync, readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { preprocessCSS, resolveConfig } from 'vite';
import { describe, expect, test } from 'vitest';

const DOCUMENT = fileURLToPath(new URL('../document.css', import.meta.url));
const CONFIG = fileURLToPath(new URL('../../../../vite.config.ts', import.meta.url));
const SOURCE = fileURLToPath(new URL('../../../', import.meta.url));
const FONTSOURCE = [
    '@fontsource/geist/400.css',
    '@fontsource/geist/500.css',
    '@fontsource/geist/600.css',
    '@fontsource/geist-mono/400.css',
    '@fontsource/geist-mono/500.css',
];
const SHIPPED = /-(?:latin|latin-ext|symbols2)-\d+-normal\.woff2/u;
const SYSTEM_GLYPHS = new Set([
    0x21_92, 0x21_97, 0x21_e7, 0x23_18, 0x23_03, 0x23_04, 0x25_d0, 0x27_13,
]);

interface Face {
    file: string;
    ranges: readonly (readonly [number, number])[];
}

function rangesOf(text: string): Face['ranges'] {
    return text.split(',').map((part) => {
        const [from = '', to = from] = part.trim().replace('U+', '').split('-');
        return [Number.parseInt(from, 16), Number.parseInt(to, 16)] as const;
    });
}

function facesOf(css: string): Face[] {
    return [...css.matchAll(/@font-face\s*\{(?<body>[^}]*)\}/gu)].map((found) => {
        const body = found.groups?.body ?? '';
        const file = /url\((?<path>[^)]*\.woff2)\)/u.exec(body)?.groups?.path ?? '';
        const range = /unicode-range:\s*(?<list>[^;]+);/u.exec(body)?.groups?.list ?? '';
        return { file, ranges: rangesOf(range) };
    });
}

function covers(faces: readonly Face[], point: number): boolean {
    return faces.some((face) => face.ranges.some(([from, to]) => point >= from && point <= to));
}

function sourcesOf(folder: string): string[] {
    return readdirSync(folder, { withFileTypes: true }).flatMap((entry) => {
        const path = join(folder, entry.name);
        if (entry.isDirectory()) {
            return ['fixtures', '__tests__'].includes(entry.name) ? [] : sourcesOf(path);
        }
        return /\.(?:css|tsx?)$/u.test(entry.name) && !entry.name.endsWith('.test.ts')
            ? [path]
            : [];
    });
}

function drawnPoints(): Map<number, string> {
    const drawn = new Map<number, string>();
    for (const path of sourcesOf(SOURCE)) {
        for (const char of readFileSync(path, 'utf8')) {
            const point = char.codePointAt(0) ?? 0;
            if (point > 0x7f && !drawn.has(point)) {
                drawn.set(point, path.replace(SOURCE, 'src/'));
            }
        }
    }
    return drawn;
}

describe('faces shipped', () => {
    test('the build keeps the Latin faces and the mono symbols in woff2 alone', async () => {
        // Given - the document's stylesheet and the window's own Vite config
        const config = await resolveConfig({ configFile: CONFIG }, 'build');

        // When - the stylesheet goes through the build's CSS pass, its @fontsource imports inlined
        const { code } = await preprocessCSS(readFileSync(DOCUMENT, 'utf8'), DOCUMENT, config);
        const files = facesOf(code).map((face) => face.file);

        // Then - ten Latin faces and two symbol faces, each a woff2, and no other file is named
        expect(files).toHaveLength(12);
        expect(files.filter((file) => !SHIPPED.test(file))).toStrictEqual([]);
        expect(code).not.toContain('.woff)');
    });

    test('no character the source draws is left to a system face by a subset the build drops', () => {
        // Given - every face @fontsource declares for the five weights, split by what the build ships
        const require = createRequire(import.meta.url);
        const faces = FONTSOURCE.flatMap((name) =>
            facesOf(readFileSync(require.resolve(name), 'utf8')),
        );
        const shipped = faces.filter((face) => SHIPPED.test(face.file));
        const dropped = faces.filter((face) => !SHIPPED.test(face.file));

        // When - each non-ASCII character of the stylesheets and the code is looked up
        const lost = [...drawnPoints()]
            .filter(([point]) => covers(dropped, point) && !covers(shipped, point))
            .map(([point, path]) => `U+${point.toString(16).toUpperCase()} in ${path}`);

        // Then - a dropped face drew none that a shipped face does not, and the rest are the system's already
        expect(lost).toStrictEqual([]);
        expect(
            [...drawnPoints().keys()].filter(
                (point) => !covers(shipped, point) && !SYSTEM_GLYPHS.has(point),
            ),
        ).toStrictEqual([]);
    });
});

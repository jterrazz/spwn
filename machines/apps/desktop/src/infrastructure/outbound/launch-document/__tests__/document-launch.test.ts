import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { describe, expect, test } from 'vitest';

import { documentLaunch, FACES } from '../document-launch';

const DOCUMENT = new URL('../../../../presentation/ui/document.css', import.meta.url);

interface Face {
    family: string;
    from: number;
    to: number;
}

function faceOf(import_: string): Face {
    const css = readFileSync(createRequire(import.meta.url).resolve(import_), 'utf8');
    const family = /font-family:\s*['"](?<name>[^'"]+)['"]/u.exec(css)?.groups?.name ?? '';
    const [from = '', to = from] = (
        /font-weight:\s*(?<range>[\d ]+);/u.exec(css)?.groups?.range ?? ''
    )
        .trim()
        .split(' ');
    return { family, from: Number(from), to: Number(to) };
}

function loadedOf(face: string): { family: string; weight: number } {
    const [weight = '', ...family] = face.replace(' 1em', '').split(' ');
    return { family: family.join(' ').replaceAll('"', ''), weight: Number(weight) };
}

function byName(one: string, other: string): number {
    return one.localeCompare(other);
}

describe('document launch', () => {
    test('the faces loaded are exactly the ones the document imports', () => {
        // Given - the document's stylesheet
        const css = readFileSync(DOCUMENT, 'utf8');
        const imported = [
            ...css.matchAll(/@import '(?<path>@fontsource(?:-variable)?\/[^']+)'/gu),
        ].map((found) => faceOf(found.groups?.path ?? ''));

        // When - each face loaded is read as a family and a weight
        const loaded = FACES.map((face) => loadedOf(face));

        // Then - the same families, none missing and none extra, each loaded at a weight its import declares
        expect(loaded.map((face) => face.family).toSorted(byName)).toStrictEqual(
            imported.map((face) => face.family).toSorted(byName),
        );
        expect(
            loaded.filter((face) =>
                imported.some(
                    (one) =>
                        one.family === face.family &&
                        (face.weight < one.from || face.weight > one.to),
                ),
            ),
        ).toStrictEqual([]);
    });

    test("the faces' load settles and the show is the one handed in", async () => {
        // Given - the document's launch, over a show that counts
        let shows = 0;
        const launch = documentLaunch(async () => {
            shows += 1;
            await Promise.resolve();
        });

        // When - the faces are loaded and the window shown
        await launch.faces();
        await launch.show();

        // Then - one show, and the page's clock read
        expect(shows).toBe(1);
        expect(launch.since()).toBeGreaterThanOrEqual(0);
    });

    test('a run asked after a delay runs', async () => {
        // Given - the document's launch
        const launch = documentLaunch(async () => {
            await Promise.resolve();
        });

        // When - a run is asked with no delay
        const ran = await new Promise<boolean>((resolve) => {
            launch.after(0, () => {
                resolve(true);
            });
        });

        // Then - it ran
        expect(ran).toBe(true);
    });
});

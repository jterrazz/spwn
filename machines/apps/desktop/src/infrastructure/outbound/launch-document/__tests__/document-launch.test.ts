import { readFileSync } from 'node:fs';

import { describe, expect, test } from 'vitest';

import { documentLaunch, FACES } from '../document-launch';

const DOCUMENT = new URL('../../../../presentation/ui/document.css', import.meta.url);

function importOf(face: string): string {
    const [weight = '', ...family] = face.replace(' 1em', '').replaceAll('"', '').split(' ');
    return `@fontsource/${family.join('-').toLowerCase()}/${weight}.css`;
}

function byName(one: string, other: string): number {
    return one.localeCompare(other);
}

describe('document launch', () => {
    test('the faces loaded are exactly the ones the document imports', () => {
        // Given - the document's stylesheet
        const css = readFileSync(DOCUMENT, 'utf8');
        const imported = [...css.matchAll(/@import '(?<path>@fontsource\/[^']+)'/gu)].map(
            (found) => found.groups?.path ?? '',
        );

        // When - each face loaded is named by its import
        const loaded = FACES.map((face) => importOf(face));

        // Then - the same faces, none missing and none extra
        expect(loaded.toSorted(byName)).toStrictEqual(imported.toSorted(byName));
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

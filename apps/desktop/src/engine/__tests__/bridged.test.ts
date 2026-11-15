import { describe, expect, test } from 'vitest';

import { BRIDGED } from '../bridged';

const SOURCES = import.meta.glob<string>(
    ['../../**/*.ts', '../../**/*.tsx', '!**/__tests__/**', '!../../fixtures/**'],
    { eager: true, import: 'default', query: '?raw' },
);

const SENT =
    /(?:use(?:Gesture|Read|Watch)\(\s*|\bcall=|\.(?:call|watch)\(\s*)['"](?<name>[a-z]+\.[a-z-]+)['"]/gu;

function sentIn(sources: Record<string, string>): string[] {
    const names = Object.values(sources).flatMap((source) =>
        [...source.matchAll(SENT)].map((match) => match.groups?.name ?? ''),
    );
    return [...new Set(names)].toSorted();
}

describe('bridged', () => {
    test('the bridge forwards exactly the calls the window sends', () => {
        // Given - every source of the window, its specs and fixtures apart
        const sources = SOURCES;

        // When - the names it sends by a read, a watch, a gesture or the client are gathered
        const sent = sentIn(sources);

        // Then - they are the bridged list, no name more and none less
        expect(Object.keys(sources).length).toBeGreaterThan(50);
        expect(sent).toStrictEqual([...BRIDGED].toSorted());
    });
});

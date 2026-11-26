import { describe, expect, test } from 'vitest';

const SOURCES = import.meta.glob<string>(
    ['../../**/*.tsx', '!**/__tests__/**', '!**/*.stories.tsx', '!../../ui/page.tsx'],
    { eager: true, import: 'default', query: '?raw' },
);

// value: the density of every page, by the file that draws it: comfortable where the body is a few facts and
// value: groups, compact where it is one list that can pass about twelve rows (spwn-ui-r1, the owner's answer d5).
const DENSITY_OF = {
    'screens/catalog/item.tsx': 'comfortable',
    'screens/catalog/shelf.tsx': 'compact',
    'screens/core/engine.tsx': 'comfortable',
    'screens/gallery/gallery.tsx': 'comfortable',
    'screens/home/home.tsx': 'comfortable',
    'screens/inbox/inbox.tsx': 'comfortable',
    'screens/machine-files/machine-files.tsx': 'compact',
    'screens/machine/activity/activity.tsx': 'comfortable',
    'screens/machine/overview/overview.tsx': 'comfortable',
    'screens/machine/packages/packages.tsx': 'compact',
    'screens/machine/packages/plan.tsx': 'compact',
    'screens/machine/ports/ports.tsx': 'compact',
    'screens/machine/sign-ins/sign-ins.tsx': 'compact',
    'screens/machine/storage/machine-storage.tsx': 'compact',
    'screens/passwords/entry.tsx': 'comfortable',
    'screens/passwords/passwords.tsx': 'compact',
    'screens/placeholder.tsx': 'comfortable',
    'screens/storage/storage.tsx': 'compact',
} as const;

function presentationPathOf(path: string): string {
    const here = ['ui', '__tests__'];
    for (const step of path.split('/')) {
        if (step === '..') {
            here.pop();
        } else {
            here.push(step);
        }
    }
    return here.join('/');
}

function pagesOf(sources: Record<string, string>): Map<string, string[]> {
    const pages = new Map<string, string[]>();
    for (const [path, source] of Object.entries(sources)) {
        const opened = [...source.matchAll(/<Page\b[^>]*>/gu)].map((match) => match[0]);
        if (opened.length > 0) {
            pages.set(
                presentationPathOf(path),
                opened.map(
                    (tag) => /density="(?<density>[a-z]+)"/u.exec(tag)?.groups?.density ?? '',
                ),
            );
        }
    }
    return pages;
}

describe('Page', () => {
    test('every page declares a density, and each file draws one density in all its states', () => {
        // Given - every `<Page>` the window draws, with the density it declares
        const pages = pagesOf(SOURCES);

        // When - the declarations are read per file
        const undecided = [...pages]
            .filter(([, densities]) => new Set(densities).size !== 1 || densities.includes(''))
            .map(([path]) => path);

        // Then - none is missing, and a file never mixes two
        expect(pages.size).toBeGreaterThan(0);
        expect(undecided).toStrictEqual([]);
    });

    test('the declared densities are the plan: nine comfortable pages, nine compact ones', () => {
        // Given - the density each file declares
        const pages = pagesOf(SOURCES);

        // When - they are laid against the plan's table
        const declared = Object.fromEntries(
            [...pages].map(([path, densities]) => [path, densities[0]]),
        );

        // Then - they are equal, no page more and none less
        expect(Object.keys(SOURCES).length).toBeGreaterThan(50);
        expect(declared).toStrictEqual(DENSITY_OF);
    });
});

import { button, component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Button, Screen } from '../button';

const SOURCES = import.meta.glob<string>(['../../**/*.{ts,tsx}', '!../../**/__tests__/**'], {
    eager: true,
    import: 'default',
    query: '?raw',
});

const DRAWS_DESTROY = [
    /(?:^|\/)button\.tsx$/u,
    /(?:^|\/)confirm-dialog\.tsx$/u,
    /\.stories\.tsx?$/u,
    /(?:^|\/)gallery\//u,
];

describe('Button', () => {
    test('one primary per screen draws', async () => {
        // Given - a screen with one primary and one plain button
        const result = await component.render(
            <Screen>
                <Button variant="primary">Create</Button>
                <Button>Cancel</Button>
            </Screen>,
            async (visitor) => {
                await visitor.see(button('Create'));
            },
        );

        // Then - both are drawn, and nothing is refused
        expect(result.tree.text).toBe('- button "Create"\n- button "Cancel"');
        expect(result.errors.text).toBe('');
    });

    test('a second primary on the same screen is refused', async () => {
        // Given - a screen asked to draw two primaries
        const render = component.render(
            <Screen>
                <Button variant="primary">Create</Button>
                <Button variant="primary">Start</Button>
            </Screen>,
        );

        // Then - the render fails on the rule
        await expect(render).rejects.toThrow(
            'One primary button per screen: a second one was drawn.',
        );
    });

    test('a production build draws a second primary plain and says so', async () => {
        // Given - a screen of a production build asked to draw two primaries
        const result = await component.render(
            <Screen strict={false}>
                <Button variant="primary">Create</Button>
                <Button variant="primary">Start</Button>
            </Screen>,
            async (visitor) => {
                await visitor.see(button('Start'));
            },
        );

        // Then - both are drawn, the second one plain, and the console names the rule
        expect(result.html.text.match(/class="[^"]*primary/gu)).toHaveLength(1);
        expect(result.errors.text).toContain('One primary button per screen');
    });

    test('a nested screen, a dialog, holds its own primary', async () => {
        // Given - a screen's primary, and a nested screen with a primary of its own
        const result = await component.render(
            <Screen>
                <Button variant="primary">New machine</Button>
                <Screen>
                    <Button variant="primary">Create</Button>
                </Screen>
            </Screen>,
        );

        // Then - both primaries are drawn
        expect(result.tree.text).toBe('- button "New machine"\n- button "Create"');
        expect(result.errors.text).toBe('');
    });

    test("a yielding primary is drawn plain beside the page's own primary, and nothing is refused", async () => {
        // Given - a header's yielding primary, and a page's primary on the same screen
        const result = await component.render(
            <Screen>
                <Button variant="primary" yields>
                    Open in Terminal
                </Button>
                <Button variant="primary">Plan</Button>
            </Screen>,
            async (visitor) => {
                await visitor.see(button('Plan'));
            },
        );

        // Then - only the page's primary is drawn primary, and no rule is broken
        expect(result.html.text.match(/class="[^"]*primary/gu)).toHaveLength(1);
        expect(result.errors.text).toBe('');
    });

    test('the filled danger is drawn only by the confirmation of a destruction', () => {
        // Given - every source file of the window but the button, its confirmation and the gallery's samples
        const swept = Object.entries(SOURCES).filter(
            ([file]) => !DRAWS_DESTROY.some((allowed) => allowed.test(file)),
        );
        const drawing = swept
            .filter(([, source]) => /['"]destroy['"]/u.test(source))
            .map(([file]) => file);

        // Then - the sweep read the window, and none of its files names the destroy look
        expect(swept.length).toBeGreaterThan(100);
        expect(drawing).toStrictEqual([]);
    });

    test('a yielding primary alone on its screen is drawn primary', async () => {
        // Given - a header's yielding primary, with no other primary
        const result = await component.render(
            <Screen>
                <Button variant="primary" yields>
                    New machine
                </Button>
            </Screen>,
            async (visitor) => {
                await visitor.see(button('New machine'));
            },
        );

        // Then - it holds the screen's one primary
        expect(result.html.text.match(/class="[^"]*primary/gu)).toHaveLength(1);
        expect(result.errors.text).toBe('');
    });
});

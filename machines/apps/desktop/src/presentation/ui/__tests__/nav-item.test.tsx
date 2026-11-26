import { button, component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { NavItem } from '../nav-item';

function ignore(): void {}

describe('NavItem', () => {
    test('a calm state is its mark alone, which carries the state word as name and title', async () => {
        // Given - the relay's item, its machine online
        const result = await component.render(
            <NavItem
                current={false}
                label="relay"
                onSelect={ignore}
                state={{ tone: 'ok', word: 'Online' }}
            />,
            async (visitor) => {
                await visitor.see(button('relay Online'));
            },
        );

        // Then - the mark is an image named by its word, inside the item's name
        expect(result.tree.text).toBe(
            '- button "relay Online":\n  - text: relay\n  - img "Online"',
        );
        expect(result.html.text).toContain('title="Online"');
    });

    test('the current place is marked as the page', async () => {
        // Given - Home's item, Home being shown
        const result = await component.render(
            <NavItem current icon="home" label="Home" onSelect={ignore} />,
        );

        // Then - the item says it is the current page
        expect(result.html.text).toContain('aria-current="page"');
    });

    test('a state other than calm is written beside its mark, which the reader is spared', async () => {
        // Given - two machines, one down, one up
        const down = await component.render(
            <NavItem
                current={false}
                label="relay"
                onSelect={ignore}
                state={{ tone: 'bad', word: 'down' }}
            />,
        );
        const up = await component.render(
            <NavItem
                current={false}
                label="mac-mini"
                onSelect={ignore}
                state={{ tone: 'ok', word: 'online' }}
            />,
        );

        // Then - the one down writes its word, the one up shows its mark alone, and both keep their name
        expect(down.content.text).toContain('down');
        expect(down.tree.text).toBe('- button "relay down"');
        expect(up.content.text).not.toContain('online');
        expect(up.tree.text).toBe(
            '- button "mac-mini online":\n  - text: mac-mini\n  - img "online"',
        );
    });
});

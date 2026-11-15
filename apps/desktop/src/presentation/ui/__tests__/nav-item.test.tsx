import { button, component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { NavItem } from '../nav-item';

function ignore(): void {}

describe('NavItem', () => {
    test('its state dot carries the state word as label and title', async () => {
        // Given - the relay's item, its machine down
        const result = await component.render(
            <NavItem
                current={false}
                label="relay"
                onSelect={ignore}
                state={{ tone: 'bad', word: 'Down' }}
            />,
            async (visitor) => {
                await visitor.see(button('relay Down'));
            },
        );

        // Then - the dot is an image named by its word, inside the item's name
        expect(result.tree.text).toBe('- button "relay Down":\n  - text: relay\n  - img "Down"');
        expect(result.html.text).toContain('title="Down"');
    });

    test('the current place is marked as the page', async () => {
        // Given - Home's item, Home being shown
        const result = await component.render(
            <NavItem current icon="home" label="Home" onSelect={ignore} />,
        );

        // Then - the item says it is the current page
        expect(result.html.text).toContain('aria-current="page"');
    });
});

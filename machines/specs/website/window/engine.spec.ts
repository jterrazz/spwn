import { button, content, heading, navigation, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');
const ROW = button('v0.22.0 available Update');
const BEHIND = heading('spwn v0.22.0 is out.');

describe('engine', () => {
    test('U1 a Mac behind the release shows the row at the sidebar foot, and it opens the Engine page', async () => {
        // Given - the proof build over an engine whose reading is behind
        const result = await website.visit('/?world=release-behind&page=home', async (visitor) => {
            // When - the row is pressed
            await visitor.click(within(sidebar, ROW));

            // Then - the Engine page, its head as the engine sent it
            await visitor.see(BEHIND);
            await visitor.see(content('This Mac runs v0.21.0.'));
            await visitor.see(button('What changed'));
        });
        expect(result.errors.text).toBe('');
    });

    test('U2 a Mac not behind shows no row, and its Engine page says what the engine says', async () => {
        // Given - the proof build over an engine whose reading is not behind
        const result = await website.visit('/?world=stories&page=engine', async (visitor) => {
            // When - the page has read the release
            await visitor.see(heading('The latest release is not known yet.'));

            // Then - when it last checked, and no Update row at the foot
            await visitor.see(content('Latest release'));
            await visitor.see(button('Check now'));
        });
        expect(result.errors.text).toBe('');
        expect(result.html.text).not.toContain(' available</span>');
    });

    test("U3 a refused Check now says the engine's message", async () => {
        // Given - an engine whose ask of GitHub is refused
        const result = await website.visit(
            '/?world=release-unreachable&page=engine&release=refused',
            async (visitor) => {
                await visitor.see(BEHIND);

                // When - Check now is pressed
                await visitor.click(button('Check now'));

                // Then - the refusal in the engine's words
                await visitor.see(
                    content(
                        'The latest release was not read: GitHub: dial tcp: connection refused.',
                        { exact: false },
                    ),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('U4 Update runs the installer then the engine start, and the row goes once the engine is not behind', async () => {
        // Given - a behind engine and a core whose install and start succeed
        const result = await website.visit(
            '/?world=release-behind&page=engine&core=update',
            async (visitor) => {
                await visitor.see(within(sidebar, ROW));

                // When - Update is pressed
                await visitor.click(button('Update'));

                // Then - both steps said their lines, and the head is the engine's new one
                await visitor.see(heading('Updated.'));
                await visitor.see(content('✓ engine · installed, running · v0.22.0'));
                await visitor.see(heading('This Mac runs the latest release.'));
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.html.text).not.toContain('v0.22.0 available</span>');
    });
});

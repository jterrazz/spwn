import { button, content, heading, navigation, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const sidebar = navigation('Sidebar');
const windowPanel = region('Window');
const WINDOW_ROW = button('Spwn v0.22.0 available Update');
const ENGINE_ROW = button('v0.22.0 available Update');
const OUT = content('Spwn v0.22.0 is out; this window runs v0.21.0.');
const INSTALL = button('Install and relaunch');

describe('window release', () => {
    test('R1 a window with a release out shows its row at the sidebar foot, and it opens the Window panel', async () => {
        // Given - the proof build whose window finds v0.22.0, over an engine not behind
        const result = await website.visit(
            '/?world=stories&page=home&window=out',
            async (visitor) => {
                // When - the row is pressed
                await visitor.click(within(sidebar, WINDOW_ROW));

                // Then - the Engine page, whose Window panel offers the release
                await visitor.see(within(windowPanel, OUT));
                await visitor.see(within(windowPanel, INSTALL));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('R2 a window current says its version, and no row at the foot', async () => {
        // Given - the proof build whose window finds nothing newer
        const result = await website.visit(
            '/?world=stories&page=home&window=current',
            async (visitor) => {
                // When - the sidebar foot's Engine line is pressed
                await visitor.click(within(sidebar, button('Engine running · v4.0.0')));
                await visitor.see(heading('The latest release is not known yet.'));

                // Then - the window's own line, and no offer
                await visitor.see(
                    within(windowPanel, content('This window runs v0.21.0, the latest release.')),
                );
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.html.text).not.toContain('available</span>');
        expect(result.html.text).not.toContain('Install and relaunch');
    });

    test("R3 a check that cannot reach the releases says so with the shell's message, and Check now asks again", async () => {
        // Given - the proof build whose window cannot reach its releases
        const result = await website.visit(
            '/?world=stories&page=engine&window=refused',
            async (visitor) => {
                await visitor.see(
                    within(windowPanel, content('The window could not read its latest release.')),
                );

                // When - the Window panel's Check now is pressed
                await visitor.click(within(windowPanel, button('Check now')));

                // Then - the same refusal, the shell's message beside it
                await visitor.see(
                    within(windowPanel, content('error sending request', { exact: false })),
                );
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.html.text).not.toContain('available</span>');
    });

    test('R4 Install and relaunch downloads the release, then relaunches', async () => {
        // Given - the proof build whose window finds v0.22.0
        const result = await website.visit(
            '/?world=stories&page=engine&window=out',
            async (visitor) => {
                // When - Install and relaunch is pressed
                await visitor.click(within(windowPanel, INSTALL));

                // Then - the window says it relaunches
                await visitor.see(within(windowPanel, content('Relaunching Spwn.')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('R4 a download under way counts what it took', async () => {
        // Given - the proof build whose install holds after its first chunk
        const result = await website.visit(
            '/?world=stories&page=engine&window=install-holding',
            async (visitor) => {
                // When - Install and relaunch is pressed
                await visitor.click(within(windowPanel, INSTALL));

                // Then - the download and what it counted
                await visitor.see(within(windowPanel, content('Downloading Spwn v0.22.0.')));
                await visitor.see(within(windowPanel, content('7.4 MB of 18 MB')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('R5 a failed install keeps the window, and Try again installs with no new check', async () => {
        // Given - the proof build whose first install is refused
        const result = await website.visit(
            '/?world=stories&page=engine&window=install-failing',
            async (visitor) => {
                // When - Install and relaunch is pressed
                await visitor.click(within(windowPanel, INSTALL));

                // Then - the failure, its message and its hint, then Try again relaunches
                await visitor.see(
                    within(windowPanel, content('The update did not install. Spwn is unchanged.')),
                );
                await visitor.see(
                    within(windowPanel, content('Permission denied', { exact: false })),
                );
                await visitor.see(
                    within(
                        windowPanel,
                        content('If Spwn runs from Downloads, move it to Applications first.'),
                    ),
                );
                await visitor.click(within(windowPanel, button('Try again')));
                await visitor.see(within(windowPanel, content('Relaunching Spwn.')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("R6 with the engine behind and the window's release out, the foot keeps the engine's row and the page offers both", async () => {
        // Given - the proof build over an engine behind, whose window finds v0.22.0
        const result = await website.visit(
            '/?world=release-behind&page=home&window=out',
            async (visitor) => {
                // When - the foot's one row is pressed
                await visitor.click(within(sidebar, ENGINE_ROW));

                // Then - the engine's Update and the window's Install and relaunch
                await visitor.see(heading('Spwn v0.22.0 is out.'));
                await visitor.see(within(region('Update'), button('Update')));
                await visitor.see(within(windowPanel, INSTALL));
            },
        );
        expect(result.errors.text).toBe('');
        expect(result.html.text).not.toContain('Spwn v0.22.0 available');
    });

    test.todo(
        'R7 the real relaunch as the new version: proved by the macOS rehearsal and the first real publish, outside make test',
    );
});

import { button, content, heading, navigation, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { PINNED, website } from '../website.specification.js';

const column = navigation('Column');
const OFFER = heading('Spwn is not installed on this Mac.');
const INSTALLED = heading('Spwn is installed. Start the engine next.');

describe('no core', () => {
    test('E1 a Mac with no core says so, and its column waits for an engine', async () => {
        // Given - the proof build with no engine and no station on the disk
        const result = await website
            .storage(PINNED)
            .visit('/?world=two-macs&page=home&core=missing', async (visitor) => {
                // When - the window has looked
                await visitor.see(OFFER);

                // Then - the command to copy, and a foot that says no engine is here
                await visitor.see(content('curl -fsSL https://spwn.sh/install.sh | bash'));
                await visitor.see(within(column, content('No engine on this Mac')));
            });
        expect(result.errors.text).toBe('');
    });

    test('E2 Install runs the installer, then Start opens the window the engine answers', async () => {
        // Given - no core on the disk
        const result = await website
            .storage(PINNED)
            .visit('/?world=two-macs&page=home&core=missing', async (visitor) => {
                await visitor.see(OFFER);

                // When - Install, then Start the engine
                await visitor.click(button('Install'));
                await visitor.see(INSTALLED);
                await visitor.click(button('Start the engine'));

                // Then - the normal Home, with its verdict
                await visitor.see(heading('2 things need you.'));
            });
        expect(result.errors.text).toBe('');
    });

    test('E3 a failed install says its last error line and offers Retry', async () => {
        // Given - an installer that ends on a checksum
        const result = await website
            .storage(PINNED)
            .visit('/?world=two-macs&page=home&core=failing', async (visitor) => {
                // When - Install is pressed
                await visitor.click(button('Install'));

                // Then - the error line as sent, and Retry
                await visitor.see(
                    heading(
                        'Error: station-darwin-arm64 does not match its checksum, nothing installed',
                    ),
                );
                await visitor.see(button('Retry'));
            });
        expect(result.errors.text).toBe('');
    });

    test('E4 an install from the terminal is found without a gesture', async () => {
        // Given - a core that lands on the disk while the window waits
        const result = await website
            .storage(PINNED)
            .visit('/?world=two-macs&page=home&core=terminal', async (visitor) => {
                // When - the terminal path is chosen
                await visitor.click(button("I'll install it from the terminal"));
                await visitor.see(heading('Waiting for Spwn on this Mac.'));

                // Then - the start is offered once station appears
                await visitor.see(INSTALLED);
            });
        expect(result.errors.text).toBe('');
    });

    test('E5 a cancelled install goes back to the offer', async () => {
        // Given - an installer that holds
        const result = await website
            .storage(PINNED)
            .visit('/?world=two-macs&page=home&core=holding', async (visitor) => {
                await visitor.click(button('Install'));
                await visitor.see(heading('Installing Spwn.'));

                // When - Cancel is pressed
                await visitor.click(button('Cancel'));

                // Then - the offer again
                await visitor.see(OFFER);
            });
        expect(result.errors.text).toBe('');
    });

    test('E6 a core installed but not started offers the start', async () => {
        // Given - a station on the disk, no engine answering
        const result = await website
            .storage(PINNED)
            .visit('/?world=two-macs&page=home&core=installed', async (visitor) => {
                // When - the window has looked
                await visitor.see(INSTALLED);

                // Then - the start and its command, and a foot that says the engine is not started
                await visitor.see(content('station engine start'));
                await visitor.see(within(column, content('Engine not started')));
            });
        expect(result.errors.text).toBe('');
    });
});

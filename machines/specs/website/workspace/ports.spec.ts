import { button, content, dialog, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

const CHAPTER =
    "each difference's action on machinefile.drift, and a call applying or capturing a chapter";

describe('ports', () => {
    test('W7 QR code of a shared port gives the address on the devices network, and its code', async () => {
        // Given - linux-1's Ports over two-macs, where :5173 is shared Private
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Ports',
            async (visitor) => {
                const shared = within(region('Shared'), region('Port 5173'));
                await visitor.see(within(shared, content('Private')));

                // When - QR code is pressed
                await visitor.click(within(shared, button('QR code')));

                // Then - the code of its address is drawn, and can be hidden again
                await visitor.see(within(shared, button('Hide QR code')));
                await visitor.see(
                    within(shared, content('https://macbook-pro.tail1234.ts.net:40001')),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("W7 the Ports tab is headed by the listing's verdict and its line", async () => {
        // Given - linux-1's Ports over two-macs, two ports listening, both shared
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Ports',
            async (visitor) => {
                // When - the listing has been read
                // Then - the engine's verdict heads it, with its line
                await visitor.see(content('2 ports listen.'));
                await visitor.see(content('2 are shared.'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('W7 Private moves a share that starts Local to the devices network, and its address with it', async () => {
        // Given - linux-1's Ports over share-local, :5173 recorded reserved Local, its address the Mac's own
        const result = await website.visit(
            '/?world=share-local&machine=linux-1&tab=Ports&pace=600',
            async (visitor) => {
                const shared = within(region('Shared'), region('Port 5173'));
                await visitor.see(within(shared, content('http://127.0.0.1:40001')));

                // When - Private is pressed on it
                await visitor.click(within(shared, button('Private')));

                // Then - its address is the one on the devices network the recording gives
                await visitor.see(
                    within(shared, content('https://macbook-pro.tail1234.ts.net:40001')),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('W18 Local moves a share to this Mac only, and its address with it', async () => {
        // Given - linux-1's Ports over two-macs, :5173 recorded shared Private on the devices network
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Ports&pace=600',
            async (visitor) => {
                const shared = within(region('Shared'), region('Port 5173'));
                await visitor.see(
                    within(shared, content('https://macbook-pro.tail1234.ts.net:40001')),
                );

                // When - Local is pressed on it
                await visitor.click(within(shared, button('Local')));

                // Then - its address is the one on this Mac the recording gives
                await visitor.see(within(shared, content('http://127.0.0.1:40001')));
                await visitor.gone(
                    within(shared, content('https://macbook-pro.tail1234.ts.net:40001')),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('W18 Public moves a share to the public address, with no confirmation asked', async () => {
        // Given - linux-1's Ports over two-macs, :5173 recorded shared Private, and its move to Public recorded answered at once, its public address the Mac's on port 443
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Ports&pace=600',
            async (visitor) => {
                const shared = within(region('Shared'), region('Port 5173'));
                await visitor.see(
                    within(shared, content('https://macbook-pro.tail1234.ts.net:40001')),
                );

                // When - Public is pressed on it
                await visitor.click(within(shared, button('Public')));

                // Then - its address is the public one the recording gives, the devices network's gone
                await visitor.see(
                    within(shared, content('https://macbook-pro.tail1234.ts.net:443')),
                );
                await visitor.gone(
                    within(shared, content('https://macbook-pro.tail1234.ts.net:40001')),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('W19 Stop sharing moves a port to Listening, not shared, where it stays', async () => {
        // Given - linux-1's Ports over two-macs, :5173 shared
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Ports',
            async (visitor) => {
                const shared = within(region('Shared'), region('Port 5173'));
                await visitor.see(shared);

                // When - Stop sharing is pressed on it, which asks nothing
                await visitor.click(within(shared, button('Stop sharing')));
                await visitor.gone(dialog('Stop sharing linux-1:5173?'));

                // Then - it has left Shared, and is listed as listening, offered to be shared again
                await visitor.gone(shared);
                await visitor.see(within(region('Listening, not shared'), button('Share :5173')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("W20 a granted program's Remove ends its grant, and the row offers a sign-in again", async () => {
        // Given - linux-1's Sign-ins over two-macs, npm held by a grant of pw-2
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Sign-ins',
            async (visitor) => {
                const rows = region('Sign-ins');

                // When - Remove of npm is pressed and confirmed
                await visitor.click(within(rows, button('Remove npm')));
                await visitor.click(
                    within(
                        dialog('Remove npm? The grant ends here; the token stays yours.'),
                        button('Remove'),
                    ),
                );

                // Then - the row offers a sign-in again
                await visitor.see(within(rows, button('Sign in npm')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('W21 Its file counts the differences per chapter', async () => {
        // Given - linux-1's Overview over stories, a VM that differs from linux-dev in its apps, its ports and its accounts
        const result = await website.visit(
            '/?world=stories&machine=linux-1&tab=Overview',
            async (visitor) => {
                const file = region('Its file');

                // When - Its file is read
                // Then - the file's verdict, then each chapter's own count
                await visitor.see(within(file, content('8 differences from linux-dev.')));
                const row = (tab: string) => within(file, button(tab, { exact: false }));
                await visitor.see(within(row('Packages'), content('3 differences')));
                await visitor.see(within(row('Ports'), content('3 differences')));
                await visitor.see(within(row('Sign-ins'), content('2 differences')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("W21 a row of Its file opens its chapter's tab", async () => {
        // Given - linux-1's Overview over stories
        const result = await website.visit(
            '/?world=stories&machine=linux-1&tab=Overview',
            async (visitor) => {
                const file = region('Its file');
                await visitor.see(file);

                // When - its Sign-ins row is pressed
                await visitor.click(within(file, button('Sign-ins', { exact: false })));

                // Then - the Sign-ins tab stands where the Overview was
                await visitor.gone(file);
                await visitor.see(region('Sign-ins'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(`W22 waits for ${CHAPTER}`);

    test.todo(`W23 waits for ${CHAPTER}`);

    test.todo(`W24 waits for ${CHAPTER}`);

    test.todo(`W29 waits for ${CHAPTER}`);
});

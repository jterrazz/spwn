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

    test.todo("W18 Public waits for the engine's confirmation of a public share");

    test('W19 Stop sharing moves a port to Listening, not shared, where it stays', async () => {
        // Given - linux-1's Ports over two-macs, :5173 shared
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Ports',
            async (visitor) => {
                const shared = within(region('Shared'), region('Port 5173'));
                await visitor.see(shared);

                // When - Stop sharing is pressed on it and confirmed
                await visitor.click(within(shared, button('Stop sharing')));
                await visitor.click(
                    within(dialog('Stop sharing linux-1:5173?'), button('Stop sharing')),
                );

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

    test.todo(
        'W21 waits for a recording of a VM that differs from its file in packages, ports and accounts, and for the count of its differences per chapter on machinefile.drift',
    );

    test.todo(`W22 waits for ${CHAPTER}`);

    test.todo(`W23 waits for ${CHAPTER}`);

    test.todo(`W24 waits for ${CHAPTER}`);

    test.todo(`W29 waits for ${CHAPTER}`);
});

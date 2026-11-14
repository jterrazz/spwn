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
                    within(shared, content('https://jterrazz-macbook-pro.tail1234.ts.net:40001')),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo('W7 waits for a call setting a share to Private, from a port that is Local');

    test.todo("W18 waits for a call setting a share's To");

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

    test('W20 Take back removes the copy of an account from the machine', async () => {
        // Given - linux-1's Accounts over two-macs, github and claude shared with it
        const result = await website.visit(
            '/?world=two-macs&machine=linux-1&tab=Accounts',
            async (visitor) => {
                const shared = region('Shared with it');
                await visitor.see(within(shared, button('Take back github')));

                // When - Take back is pressed on github and confirmed
                await visitor.click(within(shared, button('Take back github')));
                await visitor.click(
                    within(dialog('Take back github from linux-1?'), button('Take back')),
                );

                // Then - the copy is gone, and claude's stays
                await visitor.gone(within(shared, button('Take back github')));
                await visitor.see(within(shared, button('Take back claude')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo(
        'W20 waits for a recording of accounts.list after a Take back, where the account is offered again under Can be shared',
    );

    test.todo(
        'W21 waits for a recording of a VM that differs from its file in packages, ports and accounts, and for the count of its differences per chapter on machinefile.drift',
    );

    test.todo(`W22 waits for ${CHAPTER}`);

    test.todo(`W23 waits for ${CHAPTER}`);

    test.todo(`W24 waits for ${CHAPTER}`);

    test.todo(`W29 waits for ${CHAPTER}`);
});

import { button, dialog } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('mac', () => {
    test.todo('M1 lot 6');

    test.todo('M2 waits for the choices awake.keep takes, worded');

    test.todo('M3 lot 6');

    test('M4 New machine on a Mac opens the sheet with that Mac chosen', async () => {
        // Given - the window on the mini's page over two-macs, which is not This Mac
        const result = await website.visit(
            '/?world=two-macs&machine=jterrazz-mac-mini',
            async (visitor) => {
                // When - its New machine is pressed
                await visitor.click(button('New machine on jterrazz-mac-mini'));

                // Then - the sheet opens on the mini
                await visitor.see(dialog('New machine on jterrazz-mac-mini'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test.todo('M5 lot 6');

    test.todo('M6 lot 6');

    test.todo('M7 lot 6');

    test.todo('M8 lot 6');

    test.todo('M9 lot 6');

    test.todo('M10 lot 6');

    test.todo('M11 lot 6');

    test.todo('M12 lot 6');

    test.todo("M13 waits for a call reading the engine's lines and its setup runs");
});

import { button, component, content, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { tauriOpener } from '../../../engine/opener';
import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('two-macs');

const SHARED = 'https://jterrazz-macbook-pro.tail1234.ts.net:40001';

type Invoked = { command: string; args: Record<string, unknown> | undefined };

function portsOf(refusal?: string) {
    const invoked: Invoked[] = [];
    const opener = tauriOpener({
        invoke: async (command, args) => {
            invoked.push({ args, command });
            return await (refusal === undefined
                ? Promise.resolve(null)
                : Promise.reject(
                      Object.assign(new Error(refusal), { code: 'opener.not-a-web-address' }),
                  ));
        },
    });
    const page = pageOver(
        recordingFor('two-macs'),
        { machine: 'linux-1', page: 'machine', tab: 'Ports' },
        opener,
    );
    return { invoked, page };
}

const port = region('Port 5173');

describe('address', () => {
    test("open hands the browser the share's address as the engine sends it", async () => {
        // Given - linux-1's Ports over two-macs, 5173 shared over HTTP
        const { invoked, page } = portsOf();
        const result = await component.render(page, async (visitor) => {
            await visitor.see(within(port, content(SHARED)));

            // When - its Open is clicked
            await visitor.click(within(port, button('Open')));

            // Then - the share stays as it was
            await visitor.see(within(port, button('Stop sharing')));
        });
        expect(invoked).toStrictEqual([{ args: { url: SHARED }, command: 'open_url' }]);
        expect(result.content.text).not.toContain('Done');
    });

    test('an address the shell refuses is drawn as a refusal under the share', async () => {
        // Given - the same share, and a shell that refuses the address
        const { invoked, page } = portsOf('the window opens no address that names no host');
        await component.render(page, async (visitor) => {
            await visitor.see(within(port, content(SHARED)));

            // When - its Open is clicked
            await visitor.click(within(port, button('Open')));

            // Then - the refusal in the shell's words
            await visitor.see(
                within(port, content('The window opens no address', { exact: false })),
            );
        });
        expect(invoked).toHaveLength(1);
    });
});

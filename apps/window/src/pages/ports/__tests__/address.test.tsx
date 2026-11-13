import { button, component, content, disabled, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { tauriOpener } from '../../../engine/opener';
import { recordingOf } from '../../../engine/replay';
import type { Recording } from '../../../engine/replay';
import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';
import type { Route } from '../../../shell/route';
import { LOCAL_ELSEWHERE } from '../shared-row';

const recordingFor = await recordingsOf('stories', 'two-macs');

const SHARED = 'https://jterrazz-macbook-pro.tail1234.ts.net:40001';
const ON_THE_MINI = 'https://mac-mini.tail77a797.ts.net:40001';
const LOCAL = 'http://127.0.0.1:40001';

const LINUX_1: Route = { machine: 'linux-1', page: 'machine', tab: 'Ports' };
const LINUX_3: Route = { machine: 'linux-3', page: 'machine', tab: 'Ports' };

type Invoked = { command: string; args: Record<string, unknown> | undefined };

function withoutPublic(recording: Recording, address: string): Recording {
    const value: unknown = JSON.parse(
        JSON.stringify(recording).replaceAll(`"public":"${address}"`, '"public":""'),
    );
    return recordingOf(value);
}

function portsOf(recording: Recording, route: Route, refusal?: string) {
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
    return { invoked, page: pageOver(recording, route, opener) };
}

function titleOf(label: string): null | string | undefined {
    return [...document.querySelectorAll('button')]
        .find((one) => one.textContent === label)
        ?.getAttribute('title');
}

const port5173 = region('Port 5173');
const port8080 = region('Port 8080');
const heldOpen = disabled(button('Open'));

describe('address', () => {
    test("open hands the browser the share's address as the engine sends it", async () => {
        // Given - linux-1's Ports over two-macs, 5173 shared over HTTP
        const { invoked, page } = portsOf(recordingFor('two-macs'), LINUX_1);
        const result = await component.render(page, async (visitor) => {
            await visitor.see(within(port5173, content(SHARED)));

            // When - its Open is clicked
            await visitor.click(within(port5173, button('Open')));

            // Then - the share stays as it was
            await visitor.see(within(port5173, button('Stop sharing')));
        });
        expect(invoked).toStrictEqual([{ args: { url: SHARED }, command: 'open_url' }]);
        expect(result.content.text).not.toContain('Done');
    });

    test('an address the shell refuses is drawn as a refusal under the share', async () => {
        // Given - the same share, and a shell that refuses the address
        const { invoked, page } = portsOf(
            recordingFor('two-macs'),
            LINUX_1,
            'the window opens no address that names no host',
        );
        await component.render(page, async (visitor) => {
            await visitor.see(within(port5173, content(SHARED)));

            // When - its Open is clicked
            await visitor.click(within(port5173, button('Open')));

            // Then - the refusal in the shell's words
            await visitor.see(
                within(port5173, content('The window opens no address', { exact: false })),
            );
        });
        expect(invoked).toHaveLength(1);
    });

    test("a share of this Mac with a local address alone opens it, this Mac's own", async () => {
        // Given - linux-1's 5173 on this Mac, the engine giving it no public address
        const { invoked, page } = portsOf(withoutPublic(recordingFor('two-macs'), SHARED), LINUX_1);
        await component.render(page, async (visitor) => {
            await visitor.see(within(port5173, content(LOCAL)));

            // When - its Open is clicked
            await visitor.click(within(port5173, button('Open')));

            // Then - the share stays as it was
            await visitor.see(within(port5173, button('Stop sharing')));
        });
        expect(invoked).toStrictEqual([{ args: { url: LOCAL }, command: 'open_url' }]);
    });

    test('a share of another Mac opens the public address the engine gives, never its local one', async () => {
        // Given - linux-3's 8080, shared on the mac-mini with a public address
        const { invoked, page } = portsOf(recordingFor('stories'), LINUX_3);
        await component.render(page, async (visitor) => {
            await visitor.see(within(port8080, content(ON_THE_MINI)));

            // When - its Open is clicked
            await visitor.click(within(port8080, button('Open')));

            // Then - the share stays as it was
            await visitor.see(within(port8080, button('Stop sharing')));
        });
        expect(invoked).toStrictEqual([{ args: { url: ON_THE_MINI }, command: 'open_url' }]);
    });

    test('a share of another Mac with a local address alone holds Open, and says why', async () => {
        // Given - linux-3's 8080 on the mac-mini, the engine giving it no public address
        const { invoked, page } = portsOf(
            withoutPublic(recordingFor('stories'), ON_THE_MINI),
            LINUX_3,
        );
        let title: null | string | undefined;
        await component.render(page, async (visitor) => {
            // When - its address is drawn
            await visitor.see(within(port8080, content(LOCAL)));
            await visitor.see(within(port8080, heldOpen));
            title = titleOf('Open');
        });

        // Then - held, its title the reason, and the shell never asked
        expect(title).toBe(LOCAL_ELSEWHERE);
        expect(invoked).toStrictEqual([]);
    });
});

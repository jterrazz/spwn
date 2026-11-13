import { button, component, disabled, enabled, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { tauriOpener } from '../../../engine/opener';
import { pageOver } from '../../../fixtures/page-over';
import { recordingsOf } from '../../../fixtures/recordings';

const recordingFor = await recordingsOf('two-macs');

type Invoked = { command: string; args: Record<string, unknown> | undefined };

function overviewOf(machine: string) {
    const invoked: Invoked[] = [];
    const opener = tauriOpener({
        invoke: async (command, args) => {
            invoked.push({ args, command });
            return await Promise.resolve(null);
        },
    });
    const page = pageOver(
        recordingFor('two-macs'),
        { machine, page: 'machine', tab: 'Overview' },
        opener,
    );
    return { invoked, page };
}

function titleOf(label: string): null | string | undefined {
    return [...document.querySelectorAll('button')]
        .find((one) => one.textContent === label)
        ?.getAttribute('title');
}

describe('terminal', () => {
    test("open in Terminal hands the shell machines.ssh's command, and marks nothing done", async () => {
        // Given - linux-1's Overview over two-macs, running, its SSH answering
        const { invoked, page } = overviewOf('linux-1');
        const result = await component.render(page, async (visitor) => {
            await visitor.see(heading('linux-1 Running'));

            // When - Open in Terminal is clicked once machines.ssh has answered
            await visitor.see(enabled(button('Open in Terminal')));
            await visitor.click(button('Open in Terminal'));

            // Then - the button stays offered
            await visitor.see(enabled(button('Open in Terminal')));
        });
        expect(invoked).toStrictEqual([
            { args: { command: 'ssh admin@linux-1' }, command: 'open_in_terminal' },
        ]);
        expect(result.content.text).not.toContain('Done');
        expect(result.errors.text).toBe('');
    });

    test('a stopped VM holds Open in Terminal, saying its SSH does not answer', async () => {
        // Given - api's Overview over two-macs, stopped
        const { invoked, page } = overviewOf('api');
        let why: null | string | undefined;
        await component.render(page, async (visitor) => {
            // When - its header is drawn
            await visitor.see(heading('api', { exact: false }));
            await visitor.see(disabled(button('Open in Terminal')));
            why = titleOf('Open in Terminal');
        });

        // Then - held with the engine's reason, the shell never asked
        expect(why).toBe('SSH does not answer on api');
        expect(invoked).toStrictEqual([]);
    });
});

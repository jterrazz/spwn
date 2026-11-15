import { button, component, content, disabled } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../composition/compose';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { tauriOpener } from '../../../infrastructure/outbound/opener-tauri/tauri-opener';
import { EngineProvider } from '../../dependencies/dependencies';
import { Openable } from '../openable';
import { OpenButton, WITHHELD_OPENS } from '../opens';

const recordingFor = await recordingsOf('two-macs');

const TOKEN = ['ghp', '0123456789abcdefghijABCDEFGHIJ'].join('_');

type Invoked = { command: string; args: Record<string, unknown> | undefined };

function shell(refuses = false) {
    const invoked: Invoked[] = [];
    const opener = tauriOpener({
        invoke: async (command, args) => {
            invoked.push({ args, command });
            return await (refuses
                ? Promise.reject(
                      Object.assign(new Error('the window runs no command holding a line break'), {
                          code: 'opener.not-one-line',
                      }),
                  )
                : Promise.resolve(null));
        },
    });
    return { invoked, opener };
}

const HELD = 'The engine gives no command to reach linux-1';

function reach(command: string, opener: ReturnType<typeof shell>['opener']) {
    const dependencies = composeReplay({ opener, recording: recordingFor('two-macs') });
    return (
        <EngineProvider dependencies={dependencies}>
            <OpenButton
                held={HELD}
                label="Open in Terminal"
                openable={Openable.ofSsh({ Address: 'linux-1', Command: command, User: 'admin' })}
            />
        </EngineProvider>
    );
}

function titleOf(label: string): null | string | undefined {
    return [...document.querySelectorAll('button')]
        .find((one) => one.textContent === label)
        ?.getAttribute('title');
}

describe('opens', () => {
    test('a value shaped like a secret is never opened, and says why', async () => {
        // Given - a command that holds a token
        const { invoked, opener } = shell();
        let title: null | string | undefined;
        await component.render(
            reach(`ssh -o IdentityAgent=${TOKEN} admin@linux-1`, opener),
            async (visitor) => {
                // When - its button is drawn
                await visitor.see(disabled(button('Open in Terminal')));
                title = titleOf('Open in Terminal');
            },
        );

        // Then - held, its title naming the guard and never the value, and the shell never asked
        expect(title).toBe(WITHHELD_OPENS);
        expect(invoked).toStrictEqual([]);
    });

    test('no value from the engine holds the button with its reason', async () => {
        // Given - a machine with no command to reach it
        const { invoked, opener } = shell();
        let title: null | string | undefined;
        await component.render(reach('', opener), async (visitor) => {
            // When - its button is drawn
            await visitor.see(disabled(button('Open in Terminal')));
            title = titleOf('Open in Terminal');
        });

        // Then - held, saying the engine gives none
        expect(title).toBe(HELD);
        expect(invoked).toStrictEqual([]);
    });

    test("a refusal of the shell is drawn like the engine's, and nothing is marked done", async () => {
        // Given - linux-1's command, and a shell that refuses it
        const { invoked, opener } = shell(true);
        const result = await component.render(
            reach('ssh admin@linux-1', opener),
            async (visitor) => {
                // When - Open in Terminal is clicked
                await visitor.click(button('Open in Terminal'));

                // Then - the refusal, in its words, with a way to dismiss it
                await visitor.see(content('Refused'));
                await visitor.see(
                    content('The window runs no command holding a line break', { exact: false }),
                );
                await visitor.click(button('Dismiss'));
                await visitor.gone(button('Dismiss'));
            },
        );
        expect(invoked).toStrictEqual([
            { args: { command: 'ssh admin@linux-1' }, command: 'open_in_terminal' },
        ]);
        expect(result.content.text).not.toContain('Done');
    });
});

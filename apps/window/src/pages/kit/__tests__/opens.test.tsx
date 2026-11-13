import { button, component, content, disabled } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../../../engine/client';
import { Openable, tauriOpener } from '../../../engine/opener';
import { EngineProvider } from '../../../engine/react';
import { replayTransport } from '../../../engine/replay';
import { recordingsOf } from '../../../fixtures/recordings';
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

function signIn(hint: string, opener: ReturnType<typeof shell>['opener']) {
    const client = new Client(replayTransport(recordingFor('two-macs')));
    return (
        <EngineProvider client={client} opener={opener}>
            <OpenButton
                held="The engine gives no command that signs in"
                label="Sign in"
                openable={Openable.ofFix({ hint, provider: 'npm', state: 'broken' })}
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
        // Given - a fix command that holds a token
        const { invoked, opener } = shell();
        let title: null | string | undefined;
        await component.render(signIn(`npm login --token ${TOKEN}`, opener), async (visitor) => {
            // When - its button is drawn
            await visitor.see(disabled(button('Sign in')));
            title = titleOf('Sign in');
        });

        // Then - held, its title naming the guard and never the value, and the shell never asked
        expect(title).toBe(WITHHELD_OPENS);
        expect(invoked).toStrictEqual([]);
    });

    test('no value from the engine holds the button with its reason', async () => {
        // Given - an account with no fix command
        const { invoked, opener } = shell();
        let title: null | string | undefined;
        await component.render(signIn('', opener), async (visitor) => {
            // When - its button is drawn
            await visitor.see(disabled(button('Sign in')));
            title = titleOf('Sign in');
        });

        // Then - held, saying the engine gives none
        expect(title).toBe('The engine gives no command that signs in');
        expect(invoked).toStrictEqual([]);
    });

    test("a refusal of the shell is drawn like the engine's, and nothing is marked done", async () => {
        // Given - npm's fix, and a shell that refuses it
        const { invoked, opener } = shell(true);
        const result = await component.render(signIn('npm login', opener), async (visitor) => {
            // When - Sign in is clicked
            await visitor.click(button('Sign in'));

            // Then - the refusal, in its words, with a way to dismiss it
            await visitor.see(content('Refused'));
            await visitor.see(
                content('The window runs no command holding a line break', { exact: false }),
            );
            await visitor.click(button('Dismiss'));
            await visitor.gone(button('Dismiss'));
        });
        expect(invoked).toStrictEqual([
            { args: { command: 'npm login' }, command: 'open_in_terminal' },
        ]);
        expect(result.content.text).not.toContain('Done');
    });
});

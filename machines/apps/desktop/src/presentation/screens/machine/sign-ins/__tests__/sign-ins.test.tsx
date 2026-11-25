import { button, component, content, dialog, file, heading, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { pageOver } from '../../../../../composition/__tests__/page-over';
import { isRecord } from '../../../../../domain/request-key';
import { recordingsOf } from '../../../../../infrastructure/outbound/engine-replay/recordings';
import type { Recording } from '../../../../../infrastructure/outbound/engine-replay/replay';

const recordingFor = await recordingsOf('stories', 'two-macs', 'leftovers');

function signInsOver(world: string, machine: string) {
    return pageOver(recordingFor(world), { machine, page: 'machine', tab: 'Sign-ins' });
}

function refusingSignIns(world: string, machine: string): Recording {
    const recording = recordingFor(world);
    return {
        ...recording,
        exchanges: recording.exchanges.map((exchange) => {
            const asked = isRecord(exchange.request.in) ? exchange.request.in : {};
            return exchange.request.watch === 'accounts.list' && asked.Machine === machine
                ? {
                      ...exchange,
                      answers: [
                          {
                              error: {
                                  code: 'guest_timeout',
                                  message: `${machine} did not answer`,
                                  type: 'failed',
                              },
                              v: 2,
                          },
                      ],
                  }
                : exchange;
        }),
    };
}

describe('sign-ins', () => {
    test("a workspace's programs carry the engine's state and the one gesture it allows", async () => {
        // Given - linux-1's Sign-ins over two-macs
        const result = await component.render(
            signInsOver('two-macs', 'linux-1'),
            async (visitor) => {
                // When - accounts.list has answered
                await visitor.see(heading('Signed in to 2 of 5 programs.'));

                // Then - one row a program, each with the gesture its Action names
                const signIns = region('Sign-ins');
                await visitor.see(within(signIns, button('Sign in gh')));
                await visitor.see(within(signIns, button('Sign out claude')));
                await visitor.see(within(signIns, button('Sign in again wrangler')));
                await visitor.see(
                    within(signIns, content('from the password npm publish', { exact: false })),
                );
            },
        );
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/two-macs-linux-1.aria.yaml'));
    });

    test("this Mac's programs are asked of its own engine", async () => {
        // Given - this Mac's Sign-ins over two-macs
        const result = await component.render(
            signInsOver('two-macs', 'macbook-pro'),
            async (visitor) => {
                // When - accounts.list has answered
                // Then - its verdict and its five programs
                await visitor.see(heading('Signed in to 3 of 5 programs.'));
                await visitor.see(within(region('Sign-ins'), button('Sign in again npm')));
            },
        );
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/two-macs-macbook-pro.aria.yaml'));
    });

    test("a far Mac's VM is asked of this Mac's engine, by its name", async () => {
        // Given - linux-3's Sign-ins over stories, a workspace on mac-mini
        const result = await component.render(
            signInsOver('stories', 'linux-3'),
            async (visitor) => {
                // When - accounts.list has answered
                // Then - the rows this Mac's engine gives for it
                await visitor.see(region('Sign-ins'));
            },
        );
        expect(result.errors.text).toBe('');
        await expect(result.tree).toMatch(file('./__expected__/stories-linux-3.aria.yaml'));
    });

    test("a stopped VM's tab draws the engine's answer for it, never a skeleton", async () => {
        // Given - api's Sign-ins over leftovers, a workspace that is stopped
        const result = await component.render(signInsOver('leftovers', 'api'), async (visitor) => {
            // When - accounts.list:api has answered
            // Then - its stopped verdict and a row each program that offers nothing
            await visitor.see(
                heading('api is stopped: its sign-ins are read when it runs', { exact: false }),
            );
            await visitor.see(region('Sign-ins'));
        });
        expect(result.errors.text).toBe('');
        expect(result.content.text).toContain('Start the machine first');
    });

    test("a VM whose sign-ins cannot be read draws the call's refusal, never a skeleton", async () => {
        // Given - api's Sign-ins over leftovers, its sign-ins read refused by the engine
        const recording = refusingSignIns('leftovers', 'api');
        const result = await component.render(
            pageOver(recording, { machine: 'api', page: 'machine', tab: 'Sign-ins' }),
            async (visitor) => {
                // When - the engine sends the view unread, with the code of its refusal
                // Then - the page draws what the call answers
                await visitor.see(content('api did not answer', { exact: false }));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a server answers with its own verdict, and its rows offer no gesture', async () => {
        // Given - infra's Sign-ins over two-macs, a server
        const result = await component.render(signInsOver('two-macs', 'infra'), async (visitor) => {
            // When - accounts.list has answered
            // Then - the engine's sentence, and each row saying why it offers nothing
            await visitor.see(content('infra is a server', { exact: false }));
            await visitor.see(region('Sign-ins'));
        });
        expect(result.errors.text).toBe('');
        expect(result.content.text.split('A server keeps its own sign-ins')).toHaveLength(6);
        expect(result.content.text).not.toContain('Sign out');
    });

    test("signing out asks the engine's own question, and the row stays where it stands", async () => {
        // Given - linux-1's Sign-ins over two-macs
        const result = await component.render(
            signInsOver('two-macs', 'linux-1'),
            async (visitor) => {
                // When - Sign out of claude is clicked
                await visitor.click(within(region('Sign-ins'), button('Sign out claude')));

                // Then - the dialog asks the row's Ask, and confirming signs out
                const asked = dialog('Sign out of Claude? The token stays valid at Claude.');
                await visitor.see(asked);
                await visitor.click(within(asked, button('Sign out')));
                await visitor.gone(asked);
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("a granted program's Remove ends its grant, and the row asks the engine again", async () => {
        // Given - linux-1's Sign-ins over two-macs, npm granted from pw-2
        const result = await component.render(
            signInsOver('two-macs', 'linux-1'),
            async (visitor) => {
                // When - Remove of npm is clicked and confirmed
                await visitor.click(within(region('Sign-ins'), button('Remove npm')));
                const asked = dialog('Remove npm? The grant ends here; the token stays yours.');
                await visitor.click(within(asked, button('Remove')));

                // Then - the row offers a sign-in again
                await visitor.see(within(region('Sign-ins'), button('Sign in npm')));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a granted program opens its password on the Passwords page', async () => {
        // Given - linux-1's Sign-ins over two-macs, npm granted from pw-2
        const result = await component.render(
            signInsOver('two-macs', 'linux-1'),
            async (visitor) => {
                // When - the password of npm's row is opened
                await visitor.click(
                    within(region('Sign-ins'), button('Open the password npm publish')),
                );

                // Then - its page, with its grant to linux-1
                await visitor.see(heading('npm publish'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});

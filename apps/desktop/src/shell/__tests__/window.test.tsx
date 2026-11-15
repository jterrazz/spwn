import {
    button,
    component,
    content,
    dialog,
    disabled,
    file,
    heading,
    navigation,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../composition/compose';
import { EngineProvider } from '../../engine/react';
import { recordingsOf } from '../../infrastructure/outbound/engine-replay/recordings';
import type { Recording } from '../../infrastructure/outbound/engine-replay/replay';
import { PAGES } from '../../pages/registry';
import type { Route } from '../route';
import { Window } from '../window';

const recordingFor = await recordingsOf(
    'bare-mac',
    'mini-offline',
    'one-mac',
    'stories',
    'two-macs',
);

const sidebar = navigation('Sidebar');
const back = button('Back');
const forward = button('Forward');
const HOME: Route = { page: 'home' };

function windowOver(world: string, initial: Route) {
    const dependencies = composeReplay({ recording: recordingFor(world) });
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={initial} pages={PAGES} />
        </EngineProvider>
    );
}

describe('Window', () => {
    test("a world whose needs press heads Home with the engine's verdict, in the engine words", async () => {
        // Given - the window on Home over two-macs, where a background task fails
        const result = await component.render(windowOver('two-macs', HOME), async (visitor) => {
            // When - This Mac's overview has been read
            await visitor.see(content('2 things need you.'));
        });

        // Then - Home holds the verdict, the lines and the machines the engine answered
        await expect(result.tree).toMatch(file('./__expected__/two-macs-home.aria.yaml'));
    });

    test("a stopped machine's Overview is quiet: its file's chapters and its delete", async () => {
        // Given - the window on Home over two-macs
        const result = await component.render(windowOver('two-macs', HOME), async (visitor) => {
            // When - api, which is stopped, is chosen in the sidebar
            await visitor.click(within(sidebar, button('api Stopped')));

            // Then - its header is the engine's, and its Overview asks nothing a stopped machine refuses
            await visitor.see(heading('api Stopped'));
            await visitor.see(content('Its file'));
        });
        await expect(result.tree).toMatch(file('./__expected__/two-macs-api.aria.yaml'));
    });

    test("a calm Mac's Overview is its tiles and the file it follows", async () => {
        // Given - the window on This Mac over one-mac
        const result = await component.render(
            windowOver('one-mac', {
                machine: 'macbook-pro',
                page: 'machine',
                tab: 'Overview',
            }),
            async (visitor) => {
                // When - its overview has been read
                await visitor.see(content('This Mac is ready.'));
                await visitor.see(region('Packages'));
            },
        );

        // Then - the page shows the engine's measures and its refusal, and no verdict it did not give
        await expect(result.tree).toMatch(file('./__expected__/one-mac-this-mac.aria.yaml'));
    });

    test('a Mac that does not answer is shown as the engine reads it', async () => {
        // Given - the window on Home over mini-offline
        const result = await component.render(windowOver('mini-offline', HOME), async (visitor) => {
            // When - the Mac that does not answer is opened
            await visitor.click(within(sidebar, button('mac-mini not seen')));
            await visitor.see(heading('mac-mini Not seen'));
        });

        // Then - its header says it is not seen, and its Overview that the world holds no answer for it
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-mini.aria.yaml'));
    });

    test("a far Mac not seen heads its Overview with the engine's verdict, whatever its overview answers", async () => {
        // Given - mini-offline, whose mini's overview.read answers remote.unreachable
        const recording = recordingFor('mini-offline');
        const unreachable: Recording = {
            ...recording,
            exchanges: [
                ...recording.exchanges,
                {
                    answers: [
                        {
                            error: {
                                code: 'remote.unreachable',
                                message: 'mac-mini does not answer',
                                subject: 'mac-mini',
                                type: 'unreachable',
                            },
                            v: 2,
                        },
                    ],
                    request: {
                        in: {},
                        target: { name: 'mac-mini' },
                        watch: 'overview.read',
                    },
                },
            ],
        };
        const dependencies = composeReplay({ recording: unreachable });

        // When - the mini is opened
        const result = await component.render(
            <EngineProvider dependencies={dependencies}>
                <Window initial={HOME} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                await visitor.click(within(sidebar, button('mac-mini not seen')));

                // Then - its Overview stands on the verdict machines.list gives it, not on the refusal
                await visitor.see(heading('mac-mini has not been seen for 6 h.'));
                await visitor.see(
                    content('Nothing proves it is down: it may be asleep, or off your network.', {
                        exact: false,
                    }),
                );
                await visitor.gone(content('Unreachable', { exact: false }));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('no engine listening is a state of the window, not a crash', async () => {
        // Given - a transport whose socket answers nothing
        const dependencies = composeReplay({
            link: {
                kind: 'absent',
                reason: 'connection refused',
                retryIn: 4000,
                socket: '/run/engine.sock',
            },
            recording: { exchanges: [], station: '', world: 'none' },
        });
        const result = await component.render(
            <EngineProvider dependencies={dependencies}>
                <Window initial={{ page: 'home' }} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                // When - the window has heard from its transport
                await visitor.see(heading('No engine answers at /run/engine.sock.'));
            },
        );

        // Then - Home and the foot line say no engine answers at its socket, never an engine stopped
        await expect(result.tree).toMatch(file('./__expected__/no-engine-home.aria.yaml'));
    });

    test('a link still connecting says so in the foot line, never a read under way', async () => {
        // Given - a transport still connecting to its socket
        const dependencies = composeReplay({
            link: { kind: 'connecting', socket: '/run/engine.sock' },
            recording: { exchanges: [], station: '', world: 'none' },
        });
        const result = await component.render(
            <EngineProvider dependencies={dependencies}>
                <Window initial={HOME} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                // When - the window has heard from its transport
                await visitor.see(within(sidebar, content('Engine connecting')));
            },
        );

        // Then - the foot's dot is labelled Connecting, never Reading
        expect(result.tree.text).toMatch(/img "Connecting"\n\s*- text: Engine connecting/u);
    });

    test("an engine.read the engine refuses is the foot line's error, in the engine's words", async () => {
        // Given - two-macs, where engine.read answers the engine's refusal
        const recording = recordingFor('two-macs');
        const refused: Recording = {
            ...recording,
            exchanges: recording.exchanges.map((exchange) =>
                exchange.request.call === 'engine.read' && exchange.request.target === undefined
                    ? {
                          ...exchange,
                          answers: [
                              {
                                  error: {
                                      code: 'not_running',
                                      message: 'the engine is stopping',
                                      type: 'engine-stopped',
                                  },
                                  v: 2,
                              },
                          ],
                      }
                    : exchange,
            ),
        };
        const dependencies = composeReplay({ recording: refused });

        // When - the window has read it
        const result = await component.render(
            <EngineProvider dependencies={dependencies}>
                <Window initial={HOME} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(within(sidebar, content('the engine is stopping')));
            },
        );

        // Then - the foot's dot is labelled by the error's word, not as a read under way
        await expect(result.tree).toMatch(file('./__expected__/engine-refused-home.aria.yaml'));
    });

    test("the mouse's back and forward buttons walk the history", async () => {
        // Given - the window on Home over two-macs, This Mac opened
        const result = await component.render(windowOver('two-macs', HOME), async (visitor) => {
            await visitor.click(within(sidebar, button('macbook-pro online')));
            await visitor.see(heading('macbook-pro Online'));

            // When - the mouse's back button is released, then its forward button
            globalThis.dispatchEvent(new MouseEvent('mouseup', { button: 3 }));
            await visitor.see(content('2 things need you.'));
            globalThis.dispatchEvent(new MouseEvent('mouseup', { button: 4 }));

            // Then - the window is on This Mac again
            await visitor.see(heading('macbook-pro Online'));
            await visitor.see(disabled(forward));
        });
        expect(result.errors.text).toBe('');
    });

    test('a route to a machine the engine no longer lists lands on Home with one line saying so', async () => {
        // Given - the window reopened on a machine two-macs does not hold
        const result = await component.render(
            windowOver('two-macs', { machine: 'gate9', page: 'machine', tab: 'Activity' }),
            async (visitor) => {
                // When - machines.list has answered whole
                await visitor.see(content('2 things need you.'));

                // Then - Home is shown, under the line that says why
                await visitor.see(
                    content(
                        'The window was on gate9, which the engine no longer lists, so it opened Home.',
                    ),
                );
                await visitor.see(disabled(back));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a Mac with no machine yet draws no Last night, as no night ran on a machine', async () => {
        // Given - the window on Home over bare-mac, which lists no VM
        const result = await component.render(windowOver('bare-mac', HOME), async (visitor) => {
            // When - Home has answered
            await visitor.see(region('Start here'));
        });

        // Then - its panels end on Machines
        expect(result.html.text).not.toContain('Last night');
        expect(result.errors.text).toBe('');
    });

    test("a workspace's machine file offers New machine, whose dialog draws the engine's refusal of a file that is no template", async () => {
        // Given - stories, where the engine explains linux-dev as a workspace's file that says no image to make
        const result = await component.render(
            windowOver('stories', { file: 'linux-dev', page: 'machine-files' }),
            async (visitor) => {
                // When - its head's New machine is clicked
                await visitor.click(
                    within(region('linux-dev.machine.yaml'), button('New machine')),
                );

                // Then - the dialog opens on machines.new's refusal, in the engine's words
                const said = content('linux-dev makes no image', { exact: false });
                await visitor.see(within(dialog('New machine'), said));
            },
        );
        expect(result.errors.text).toBe('');
    });
});

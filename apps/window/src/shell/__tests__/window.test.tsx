import {
    button,
    component,
    content,
    dialog,
    disabled,
    enabled,
    file,
    heading,
    navigation,
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../../engine/client';
import { EngineProvider } from '../../engine/react';
import { replayTransport } from '../../engine/replay';
import type { Recording } from '../../engine/replay';
import { recordingsOf } from '../../fixtures/recordings';
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
    const client = new Client(replayTransport(recordingFor(world)));
    return (
        <EngineProvider client={client}>
            <Window initial={initial} pages={PAGES} />
        </EngineProvider>
    );
}

describe('Window', () => {
    test('a world with something broken lists what needs you, in the engine words', async () => {
        // Given - the window on Home over two-macs, where npm's sign-in expired
        const result = await component.render(windowOver('two-macs', HOME), async (visitor) => {
            // When - This Mac's overview has been read
            await visitor.see(content('npm sign-in expired'));
        });

        // Then - Home holds the needs, the lines and the machines the engine answered
        await expect(result.tree).toMatch(file('./__expected__/two-macs-home.aria.yaml'));
    });

    test("a stopped machine's Overview is quiet: its file's chapters and its delete", async () => {
        // Given - the window on Home over two-macs
        const result = await component.render(windowOver('two-macs', HOME), async (visitor) => {
            // When - api, which is stopped, is chosen in the sidebar
            await visitor.click(within(sidebar, button('api stopped')));

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
                machine: 'jterrazz-macbook-pro',
                page: 'machine',
                tab: 'Overview',
            }),
            async (visitor) => {
                // When - its overview has been read
                await visitor.see(content('No verdict from the engine.'));
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
            await visitor.click(within(sidebar, button('jterrazz-mac-mini not seen')));
            await visitor.see(heading('jterrazz-mac-mini Not seen'));
        });

        // Then - its header says it is not seen, and its Overview that the world holds no answer for it
        await expect(result.tree).toMatch(file('./__expected__/mini-offline-mini.aria.yaml'));
    });

    test('no engine listening is a state of the window, not a crash', async () => {
        // Given - a transport whose socket answers nothing
        const client = new Client(
            replayTransport(
                { exchanges: [], station: '', world: 'none' },
                {
                    kind: 'absent',
                    reason: 'connection refused',
                    retryIn: 4000,
                    socket: '/run/engine.sock',
                },
            ),
        );
        const result = await component.render(
            <EngineProvider client={client}>
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
        const client = new Client(
            replayTransport(
                { exchanges: [], station: '', world: 'none' },
                { kind: 'connecting', socket: '/run/engine.sock' },
            ),
        );
        const result = await component.render(
            <EngineProvider client={client}>
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
        const client = new Client(replayTransport(refused));

        // When - the window has read it
        const result = await component.render(
            <EngineProvider client={client}>
                <Window initial={HOME} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(within(sidebar, content('the engine is stopping')));
            },
        );

        // Then - the foot's dot is labelled by the error's word, not as a read under way
        await expect(result.tree).toMatch(file('./__expected__/engine-refused-home.aria.yaml'));
    });

    test('back returns through tabs and pages, then is disabled', async () => {
        // Given - the window on Home over two-macs, nothing behind it
        const result = await component.render(windowOver('two-macs', HOME), async (visitor) => {
            await visitor.see(disabled(back));

            // When - This Mac is opened, its Ports tab chosen, then back is taken twice
            await visitor.click(within(sidebar, button('jterrazz-macbook-pro online')));
            await visitor.click(content('Ports'));
            await visitor.see(region('What listens on this Mac'));
            await visitor.see(enabled(back));
            await visitor.click(back);
            await visitor.see(content("What each machine holds of this Mac's memory."));
            await visitor.click(back);

            // Then - the window is back on Home, back disabled
            await visitor.see(disabled(back));
            await visitor.see(content('npm sign-in expired'));
        });
        expect(result.errors.text).toBe('');
    });

    test('forward retraces what back undid, and is disabled at the last place', async () => {
        // Given - the window on Home over two-macs, nothing behind it nor ahead
        const result = await component.render(windowOver('two-macs', HOME), async (visitor) => {
            await visitor.see(disabled(back));
            await visitor.see(disabled(forward));

            // When - This Mac is opened, then back taken
            await visitor.click(within(sidebar, button('jterrazz-macbook-pro online')));
            await visitor.see(heading('jterrazz-macbook-pro Online'));
            await visitor.see(disabled(forward));
            await visitor.click(back);
            await visitor.see(disabled(back));
            await visitor.see(enabled(forward));
            await visitor.click(forward);

            // Then - the window is on This Mac again, back enabled and nothing ahead
            await visitor.see(heading('jterrazz-macbook-pro Online'));
            await visitor.see(enabled(back));
            await visitor.see(disabled(forward));
        });
        expect(result.errors.text).toBe('');
    });

    test('⌘[ and ⌘] walk the history as the buttons do', async () => {
        // Given - the window on Home over two-macs
        const result = await component.render(windowOver('two-macs', HOME), async (visitor) => {
            // When - This Mac is opened, then ⌘[ and ⌘] pressed
            await visitor.click(within(sidebar, button('jterrazz-macbook-pro online')));
            await visitor.see(heading('jterrazz-macbook-pro Online'));
            await visitor.press('Control>}[BracketLeft]{/Control');
            await visitor.see(content('npm sign-in expired'));
            await visitor.see(enabled(forward));
            await visitor.press('Control>}[BracketRight]{/Control');

            // Then - the window is on This Mac again, back enabled
            await visitor.see(heading('jterrazz-macbook-pro Online'));
            await visitor.see(enabled(back));
        });
        expect(result.errors.text).toBe('');
    });

    test('a route to a machine the engine no longer lists lands on Home with one line saying so', async () => {
        // Given - the window reopened on a machine two-macs does not hold
        const result = await component.render(
            windowOver('two-macs', { machine: 'gate9', page: 'machine', tab: 'Logs' }),
            async (visitor) => {
                // When - machines.list has answered whole
                await visitor.see(content('npm sign-in expired'));

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

    test('a Mac with no machine yet opens on Start here, whose New machine opens the dialog', async () => {
        // Given - the window on Home over bare-mac, which lists no VM
        const result = await component.render(windowOver('bare-mac', HOME), async (visitor) => {
            // When - Start here's New machine is clicked
            await visitor.click(button('New machine, a workspace'));

            // Then - the New machine dialog opens
            await visitor.see(dialog('New machine on this Mac'));
        });
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

    test("a workspace's machine file offers New machine, which opens the dialog", async () => {
        // Given - stories, where the engine explains linux-dev by name as a file whose machine.kind is workspace
        const result = await component.render(
            windowOver('stories', { file: 'linux-dev', page: 'machine-files' }),
            async (visitor) => {
                // When - its head's New machine is clicked
                await visitor.click(
                    within(region('linux-dev.machine.yaml'), button('New machine')),
                );

                // Then - the New machine dialog opens
                await visitor.see(dialog('New machine on this Mac'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});

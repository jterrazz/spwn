import {
    button,
    component,
    content,
    dialog,
    disabled,
    enabled,
    field,
    heading,
    navigation,
    waitUntil,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { composeReplay } from '../../../../composition/compose';
import { documentPresence } from '../../../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../../../infrastructure/outbound/engine-link/client';
import type { Transport } from '../../../../infrastructure/outbound/engine-link/transport';
import { holdingCall } from '../../../../infrastructure/outbound/engine-replay/__tests__/held-call';
import { recordingsOf } from '../../../../infrastructure/outbound/engine-replay/recordings';
import {
    recordingOf,
    stagedReplay,
} from '../../../../infrastructure/outbound/engine-replay/replay';
import type { Recording } from '../../../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../../../infrastructure/outbound/opener-tauri/tauri-opener';
import { EngineProvider } from '../../../dependencies/dependencies';
import type { Route } from '../../../navigation/route';
import { Window } from '../../../navigation/window';
import { PAGES } from '../../registry';

const recordingFor = await recordingsOf('creating', 'engine-stopped', 'two-macs');

const sidebar = navigation('Sidebar');

function machineOver(world: string, machine: string) {
    const initial: Route = { machine, page: 'machine', tab: 'Overview' };
    const dependencies = composeReplay({ recording: recordingFor(world) });
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={initial} pages={PAGES} />
        </EngineProvider>
    );
}

function linuxOneOver(transport: Transport) {
    const dependencies = {
        engine: new Client(transport),
        opener: NO_OPENER,
        presence: documentPresence(),
    };
    const initial: Route = { machine: 'linux-1', page: 'machine', tab: 'Overview' };
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={initial} pages={PAGES} />
        </EngineProvider>
    );
}

function unhealthy(recording: Recording): Recording {
    const said = JSON.stringify(recording).replaceAll(
        '"Tone":"ok","Sentence":"Healthy."',
        '"Tone":"bad","Sentence":"Down."',
    );
    return recordingOf(JSON.parse(said));
}

type Boxes = { lifecycle: string; tabs: number };

function lifecycleBox(): string {
    const found = [...document.querySelectorAll('button')].find((one) =>
        ['Start', 'Stop'].includes(one.textContent ?? ''),
    );
    const box = found?.getBoundingClientRect();
    return box === undefined ? 'none' : `y${box.y} h${box.height}`;
}

function headerBoxes(): Boxes {
    const tabs = document.querySelector('[role="tablist"]')?.getBoundingClientRect().y ?? 0;
    return { lifecycle: lifecycleBox(), tabs };
}

describe('gestures', () => {
    test('stop asks first, then stops the machine, and the header and sidebar read it again', async () => {
        // Given - linux-1's page over two-macs, linux-1 running
        const result = await component.render(
            machineOver('two-macs', 'linux-1'),
            async (visitor) => {
                await visitor.see(heading('linux-1 running'));

                // When - Stop is clicked, then confirmed in its dialog
                await visitor.click(button('Stop'));
                await visitor.see(dialog('Stop linux-1?'));
                await visitor.see(
                    within(
                        dialog('Stop linux-1?'),
                        content('linux-1 stops: what it serves stops until you start it again.'),
                    ),
                );
                await visitor.click(within(dialog('Stop linux-1?'), button('Stop')));

                // Then - the engine's machines.list, read again, says it is stopped
                await visitor.gone(dialog('Stop linux-1?'));
                await visitor.see(heading('linux-1 stopped'));
                await visitor.see(within(sidebar, button('linux-1 stopped')));
                await visitor.see(content('done'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('the Done of a header gesture moves neither the buttons nor the tabs', async () => {
        // Given - linux-1's page over two-macs, its header measured while it runs
        let before: Boxes | undefined;
        let after: Boxes | undefined;
        await component.render(machineOver('two-macs', 'linux-1'), async (visitor) => {
            await visitor.see(heading('linux-1 running'));
            before = headerBoxes();

            // When - Stop is confirmed and the engine says it is done
            await visitor.click(button('Stop'));
            await visitor.click(within(dialog('Stop linux-1?'), button('Stop')));
            await visitor.see(heading('linux-1 stopped'));
            await visitor.see(content('done'));
            after = headerBoxes();
        });

        // Then - the gesture the wire lists now stands where Stop stood, and so does the tab bar
        expect(before?.tabs).toBeGreaterThan(0);
        expect(before?.lifecycle).not.toBe('none');
        expect(after).toStrictEqual(before);
    });

    test('only the last gesture of the header says how it ended', async () => {
        // Given - linux-1 stopped from its page over two-macs, its Start to be held
        const start = holdingCall(stagedReplay(recordingFor('two-macs')), 'machines.start');
        const result = await component.render(linuxOneOver(start.transport), async (visitor) => {
            await visitor.click(button('Stop'));
            await visitor.click(within(dialog('Stop linux-1?'), button('Stop')));
            await visitor.see(content('done'));

            // When - Start follows, and the connection drops before its answer
            await visitor.click(button('Start'));
            await waitUntil(() => start.sent(), { why: 'machines.start sent' });
            start.drop();

            // Then - Start's unknown end replaces Stop's done
            await visitor.see(content('the engine may have done it', { exact: false }));
            await visitor.gone(content('done'));
        });
        expect(result.errors.text).toBe('');
    });

    test('a stop whose connection drops before it answers is unknown, and the page shows what the engine says', async () => {
        // Given - linux-1's page over two-macs, where machines.stop is held unanswered once sent
        const stop = holdingCall(stagedReplay(recordingFor('two-macs')), 'machines.stop');
        const result = await component.render(linuxOneOver(stop.transport), async (visitor) => {
            await visitor.see(heading('linux-1 running'));

            // When - Stop is confirmed, and the connection drops before its answer
            await visitor.click(button('Stop'));
            await visitor.click(within(dialog('Stop linux-1?'), button('Stop')));
            await visitor.gone(dialog('Stop linux-1?'));
            await waitUntil(() => stop.sent(), { why: 'machines.stop sent' });
            stop.drop();

            // Then - the stop is unknown, never failed, and the header is the engine's machines.list read again
            await visitor.see(content('the engine may have done it', { exact: false }));
            await visitor.see(content('the engine closed the connection', { exact: false }));
            await visitor.see(heading('linux-1 running'));
            await visitor.gone(content('done'));
        });
        expect(result.errors.text).toBe('');
    });

    test('a cancel at the confirm leaves the machine as it was', async () => {
        // Given - linux-1's page over two-macs
        const result = await component.render(
            machineOver('two-macs', 'linux-1'),
            async (visitor) => {
                await visitor.see(heading('linux-1 running'));

                // When - Stop is clicked, then cancelled
                await visitor.click(button('Stop'));
                await visitor.click(within(dialog('Stop linux-1?'), button('Cancel')));

                // Then - no dialog, and the machine still runs
                await visitor.gone(dialog('Stop linux-1?'));
                await visitor.see(heading('linux-1 running'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('start asks nothing, and the machine reads running', async () => {
        // Given - api's page over two-macs, api stopped
        const result = await component.render(machineOver('two-macs', 'api'), async (visitor) => {
            await visitor.see(heading('api stopped'));

            // When - Start is clicked
            await visitor.click(button('Start'));

            // Then - it runs, with no dialog asked
            await visitor.see(heading('api running'));
        });
        expect(result.errors.text).toBe('');
    });

    test('delete asks for the name typed, then the machine leaves the sidebar', async () => {
        // Given - api's page over two-macs
        const result = await component.render(machineOver('two-macs', 'api'), async (visitor) => {
            await visitor.see(heading('api stopped'));

            // When - Delete api is clicked, a wrong name typed, then the right one
            await visitor.click(button('Delete api…'));
            const asked = dialog('Delete api?');
            const typed = within(asked, field('Type api to confirm'));
            const held = disabled(button('Delete'));
            const free = enabled(button('Delete'));
            await visitor.see(within(asked, held));
            await visitor.fill(typed, 'ap');
            await visitor.see(within(asked, held));
            await visitor.fill(typed, 'api');
            await visitor.see(within(asked, free));
            await visitor.click(within(asked, button('Delete')));

            // Then - Home, and api is no machine of the sidebar any more
            const gone = within(sidebar, button('api stopped'));
            await visitor.see(content('2 things need you.'));
            await visitor.gone(gone);
        });
        expect(result.errors.text).toBe('');
    });

    test('a delete of a machine being created shows the refusal the engine words and codes', async () => {
        // Given - linux-5's page over creating, its image still being built
        const result = await component.render(
            machineOver('creating', 'linux-5'),
            async (visitor) => {
                await visitor.see(content('Creating: image 1 of 5'));

                // When - its delete is confirmed by its name
                await visitor.click(button('Delete linux-5…'));
                const asked = dialog('Delete linux-5?');
                await visitor.fill(within(asked, field('Type linux-5 to confirm')), 'linux-5');
                await visitor.click(within(asked, button('Delete')));

                // Then - the engine's message and its hint, no word of the window's own
                await visitor.see(content('is being created on macbook-pro', { exact: false }));
                await visitor.see(
                    content('wait for its create to end, or interrupt it where it runs'),
                );
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a gesture the engine fails shows its error and its hint, and the machine as it was', async () => {
        // Given - linux-1's page over engine-stopped, where This Mac's engine is stopped
        const result = await component.render(
            machineOver('engine-stopped', 'linux-1'),
            async (visitor) => {
                await visitor.see(heading('linux-1 stopped'));

                // When - Start is clicked
                await visitor.click(button('Start'));

                // Then - the engine's words, and Dismiss clears them
                const said = content('the engine is stopped on macbook-pro', {
                    exact: false,
                });
                await visitor.see(said);
                await visitor.see(content('station engine start'));
                await visitor.click(button('Dismiss'));
                await visitor.gone(said);
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("a server's restart is asked first, sent to its Mac, and the engine's answer is shown", async () => {
        // Given - infra's page over two-macs, a server on mac-mini whose header says nothing of its uptime
        const result = await component.render(machineOver('two-macs', 'infra'), async (visitor) => {
            await visitor.see(heading('infra', { exact: false }));
            const restarted = content('Up 1 min.', { exact: false });
            await visitor.gone(restarted);

            // When - Restart is clicked, then confirmed in its dialog by the server's name
            await visitor.click(button('Restart'));
            const confirm = dialog('Restart infra?');
            await visitor.fill(within(confirm, field('Type infra to confirm')), 'infra');
            await visitor.click(within(confirm, button('Restart')));

            // Then - mac-mini restarts it, and the header, read again, says Done and the uptime its Mac gives
            await visitor.gone(dialog('Restart infra?'));
            await visitor.see(content('done'));
            await visitor.see(restarted);
        });
        expect(result.errors.text).toBe('');
    });

    test('a server whose health verdict is bad leads its header with Restart, and a server whose is not has no primary', async () => {
        // Given - infra's page over two-macs, once as recorded and once with its health verdict worded bad
        const sick = unhealthy(recordingFor('two-macs'));
        const result = await component.render(
            <EngineProvider dependencies={composeReplay({ recording: sick })}>
                <Window
                    initial={{ machine: 'infra', page: 'machine', tab: 'Overview' }}
                    pages={PAGES}
                />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(enabled(button('Restart')));
            },
        );
        const well = await component.render(machineOver('two-macs', 'infra'), async (visitor) => {
            await visitor.see(enabled(button('Restart')));
        });

        // Then - Restart alone is the primary of the first, and the second has none
        expect(result.html.text).toMatch(/<button[^>]*primary[^>]*>Restart</u);
        expect(result.html.text.match(/class="[^"]*primary/gu)).toHaveLength(1);
        expect(well.html.text).not.toMatch(/class="[^"]*primary/u);
        expect(result.errors.text).toBe('');
    });

    test("a workspace offers no Restart, and Open in Terminal shows the engine's command before it is clicked", async () => {
        // Given - linux-1's page over two-macs
        let shows: null | string | undefined;
        const result = await component.render(
            machineOver('two-macs', 'linux-1'),
            async (visitor) => {
                // When - its header is drawn and machines.ssh has answered
                await visitor.see(heading('linux-1 running'));
                await visitor.see(enabled(button('Open in Terminal')));
                shows = [...document.querySelectorAll('button')]
                    .find((one) => one.textContent === 'Open in Terminal')
                    ?.getAttribute('title');

                // Then - Start and Stop, no Restart, and the terminal offered
                await visitor.gone(button('Restart'));
            },
        );
        expect(shows).toBe('ssh admin@linux-1');
        expect(result.html.text).not.toMatch(/class="[^"]*primary/u);
        expect(result.errors.text).toBe('');
    });

    test('a header draws the gestures the wire lists for its state and no other', async () => {
        // Given - api stopped and linux-1 running over two-macs
        const stopped = await component.render(machineOver('two-macs', 'api'), async (visitor) => {
            // When - a stopped workspace's header is drawn
            await visitor.see(heading('api stopped'));

            // Then - its Start only
            await visitor.see(button('Start'));
            await visitor.gone(button('Stop'));
        });
        const running = await component.render(
            machineOver('two-macs', 'linux-1'),
            async (visitor) => {
                // When - a running workspace's header is drawn
                await visitor.see(heading('linux-1 running'));

                // Then - its Stop only
                await visitor.see(button('Stop'));
                await visitor.gone(button('Start'));
            },
        );
        expect(stopped.errors.text).toBe('');
        expect(running.errors.text).toBe('');
    });

    test("a Mac's header says what it is, its engine's state, and New machine", async () => {
        // Given - This Mac's page over two-macs
        const result = await component.render(
            machineOver('two-macs', 'macbook-pro'),
            async (visitor) => {
                // When - its header is drawn
                await visitor.see(heading('macbook-pro', { exact: false }));

                // Then - its line is the engine's, and New machine is its one action
                await visitor.see(content('This Mac · Engine running', { exact: false }));
                await visitor.see(enabled(button('New machine on macbook-pro')));
            },
        );
        expect(result.errors.text).toBe('');
    });
});

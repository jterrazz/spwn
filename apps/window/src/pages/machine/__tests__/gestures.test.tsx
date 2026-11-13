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
    region,
    within,
} from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../../../engine/client';
import { EngineProvider } from '../../../engine/react';
import { replayTransport, stagedReplay } from '../../../engine/replay';
import { recordingsOf } from '../../../fixtures/recordings';
import type { Route } from '../../../shell/route';
import { Window } from '../../../shell/window';
import { PAGES } from '../../registry';

const recordingFor = await recordingsOf('engine-stopped', 'two-macs');

const sidebar = navigation('Sidebar');

function machineOver(world: string, machine: string) {
    const initial: Route = { machine, page: 'machine', tab: 'Overview' };
    const client = new Client(replayTransport(recordingFor(world)));
    return (
        <EngineProvider client={client}>
            <Window initial={initial} pages={PAGES} />
        </EngineProvider>
    );
}

type Boxes = { start: string; stop: string; tabs: number };

function buttonBox(label: string): string {
    const found = [...document.querySelectorAll('button')].find((one) => one.textContent === label);
    const box = found?.getBoundingClientRect();
    return box === undefined ? 'none' : `${box.x},${box.y} ${box.width}x${box.height}`;
}

function headerBoxes(): Boxes {
    const tabs = document.querySelector('[role="tablist"]')?.getBoundingClientRect().y ?? 0;
    return { start: buttonBox('Start'), stop: buttonBox('Stop'), tabs };
}

describe('gestures', () => {
    test('stop asks first, then stops the machine, and the header and sidebar read it again', async () => {
        // Given - linux-1's page over two-macs, linux-1 running
        const result = await component.render(
            machineOver('two-macs', 'linux-1'),
            async (visitor) => {
                await visitor.see(heading('linux-1 Running'));

                // When - Stop is clicked, then confirmed in its dialog
                await visitor.click(button('Stop'));
                await visitor.see(dialog('Stop linux-1?'));
                await visitor.click(within(dialog('Stop linux-1?'), button('Stop')));

                // Then - the engine's machines.list, read again, says it is stopped
                await visitor.gone(dialog('Stop linux-1?'));
                await visitor.see(heading('linux-1 Stopped'));
                await visitor.see(within(sidebar, button('linux-1 stopped')));
                await visitor.see(content('Done'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('the Done of a header gesture moves neither the buttons nor the tabs', async () => {
        // Given - linux-1's page over two-macs, its header measured while it runs
        let before: Boxes | undefined;
        let after: Boxes | undefined;
        await component.render(machineOver('two-macs', 'linux-1'), async (visitor) => {
            await visitor.see(heading('linux-1 Running'));
            before = headerBoxes();

            // When - Stop is confirmed and the engine says it is done
            await visitor.click(button('Stop'));
            await visitor.click(within(dialog('Stop linux-1?'), button('Stop')));
            await visitor.see(heading('linux-1 Stopped'));
            await visitor.see(content('Done'));
            after = headerBoxes();
        });

        // Then - Start, Stop and the tab bar stand where they stood
        expect(before?.tabs).toBeGreaterThan(0);
        expect([before?.start, before?.stop]).not.toContain('none');
        expect(after).toStrictEqual(before);
    });

    test('only the last gesture of the header says how it ended', async () => {
        // Given - linux-1 stopped from its page over two-macs
        const result = await component.render(
            machineOver('two-macs', 'linux-1'),
            async (visitor) => {
                await visitor.click(button('Stop'));
                await visitor.click(within(dialog('Stop linux-1?'), button('Stop')));
                await visitor.see(content('Done'));

                // When - Start follows, which two-macs never recorded for linux-1
                await visitor.click(button('Start'));

                // Then - Start's failure replaces Stop's done
                await visitor.see(content('two-macs holds no answer', { exact: false }));
                await visitor.gone(content('Done'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a stop whose connection drops before it answers is unknown, and the page shows what the engine says', async () => {
        // Given - linux-1's page over two-macs, where machines.stop answers a second after it is sent
        const stage = stagedReplay(recordingFor('two-macs'), undefined, {
            gestureFrame: 0,
            latency: (request) =>
                'call' in request && request.call === 'machines.stop' ? 1000 : 0,
        });
        const result = await component.render(
            <EngineProvider client={new Client(stage.transport)}>
                <Window
                    initial={{ machine: 'linux-1', page: 'machine', tab: 'Overview' }}
                    pages={PAGES}
                />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(heading('linux-1 Running'));

                // When - Stop is confirmed, and the connection drops before its answer
                await visitor.click(button('Stop'));
                await visitor.click(within(dialog('Stop linux-1?'), button('Stop')));
                await visitor.gone(dialog('Stop linux-1?'));
                stage.drop();

                // Then - the stop is unknown, never failed, and the header is the engine's machines.list read again
                await visitor.see(content('the engine may have done it', { exact: false }));
                await visitor.see(content('the engine closed the connection', { exact: false }));
                await visitor.see(heading('linux-1 Running'));
                await visitor.gone(content('Done'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('a cancel at the confirm leaves the machine as it was', async () => {
        // Given - linux-1's page over two-macs
        const result = await component.render(
            machineOver('two-macs', 'linux-1'),
            async (visitor) => {
                await visitor.see(heading('linux-1 Running'));

                // When - Stop is clicked, then cancelled
                await visitor.click(button('Stop'));
                await visitor.click(within(dialog('Stop linux-1?'), button('Cancel')));

                // Then - no dialog, and the machine still runs
                await visitor.gone(dialog('Stop linux-1?'));
                await visitor.see(heading('linux-1 Running'));
            },
        );
        expect(result.errors.text).toBe('');
    });

    test('start asks nothing, and the machine reads running', async () => {
        // Given - api's page over two-macs, api stopped
        const result = await component.render(machineOver('two-macs', 'api'), async (visitor) => {
            await visitor.see(heading('api Stopped'));

            // When - Start is clicked
            await visitor.click(button('Start'));

            // Then - it runs, with no dialog asked
            await visitor.see(heading('api Running'));
        });
        expect(result.errors.text).toBe('');
    });

    test('delete asks for the name typed, then the machine leaves the sidebar', async () => {
        // Given - api's page over two-macs
        const result = await component.render(machineOver('two-macs', 'api'), async (visitor) => {
            await visitor.see(heading('api Stopped'));

            // When - Delete api is clicked, a wrong name typed, then the right one
            await visitor.click(button('Delete api…'));
            const asked = dialog('Delete api?');
            const typed = within(asked, field('Type api to confirm'));
            const held = disabled(button('Delete api'));
            const free = enabled(button('Delete api'));
            await visitor.see(within(asked, held));
            await visitor.fill(typed, 'ap');
            await visitor.see(within(asked, held));
            await visitor.fill(typed, 'api');
            await visitor.see(within(asked, free));
            await visitor.click(within(asked, button('Delete api')));

            // Then - Home, and api is no machine of the sidebar any more
            const gone = within(sidebar, button('api stopped'));
            await visitor.see(region('Needs you'));
            await visitor.gone(gone);
        });
        expect(result.errors.text).toBe('');
    });

    test('a gesture the engine fails shows its error and its hint, and the machine as it was', async () => {
        // Given - linux-1's page over engine-stopped, where This Mac's engine is stopped
        const result = await component.render(
            machineOver('engine-stopped', 'linux-1'),
            async (visitor) => {
                await visitor.see(heading('linux-1', { exact: false }));

                // When - Stop is confirmed
                await visitor.click(button('Stop'));
                await visitor.click(within(dialog('Stop linux-1?'), button('Stop')));

                // Then - the engine's words, and Dismiss clears them
                const said = content('the engine is stopped on This Mac', { exact: false });
                await visitor.see(said);
                await visitor.see(content('station engine start'));
                await visitor.click(button('Dismiss'));
                await visitor.gone(said);
            },
        );
        expect(result.errors.text).toBe('');
    });

    test("a server's restart is asked first, sent to its Mac, and the engine's answer is shown", async () => {
        // Given - infra's page over two-macs, a server on jterrazz-mac-mini
        const result = await component.render(machineOver('two-macs', 'infra'), async (visitor) => {
            await visitor.see(heading('infra', { exact: false }));

            // When - Restart is clicked, then confirmed in its dialog
            await visitor.click(button('Restart'));
            await visitor.click(within(dialog('Restart infra?'), button('Restart')));

            // Then - the world's engine refuses it, and the header says so
            await visitor.gone(dialog('Restart infra?'));
            await visitor.see(content('no machine named "infra"', { exact: false }));
        });
        expect(result.errors.text).toBe('');
    });

    test('a workspace offers no Restart, and Open in Terminal is drawn but held, saying why', async () => {
        // Given - linux-1's page over two-macs
        let why: null | string | undefined;
        const result = await component.render(
            machineOver('two-macs', 'linux-1'),
            async (visitor) => {
                // When - its header is drawn
                await visitor.see(heading('linux-1 Running'));
                why = [...document.querySelectorAll('button')]
                    .find((one) => one.textContent === 'Open in Terminal')
                    ?.getAttribute('title');

                // Then - Start and Stop, no Restart, and the terminal held
                await visitor.see(disabled(button('Open in Terminal')));
                await visitor.gone(button('Restart'));
            },
        );
        expect(why).toBe('The window cannot open a terminal yet');
        expect(result.errors.text).toBe('');
    });

    test("a Mac's header says what it is, its engine's state, and New machine", async () => {
        // Given - This Mac's page over two-macs
        const result = await component.render(
            machineOver('two-macs', 'jterrazz-macbook-pro'),
            async (visitor) => {
                // When - its header is drawn
                await visitor.see(heading('jterrazz-macbook-pro', { exact: false }));

                // Then - its line is the engine's, and New machine is its one action
                await visitor.see(content('This Mac · Engine running'));
                await visitor.see(enabled(button('New machine')));
            },
        );
        expect(result.errors.text).toBe('');
    });
});

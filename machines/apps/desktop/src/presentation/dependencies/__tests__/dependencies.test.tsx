import { component, content, waitUntil } from '@jterrazz/test';
import { Profiler } from 'react';
import { describe, expect, test } from 'vitest';

import { keyOf } from '../../../domain/request-key';
import { documentPresence } from '../../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../../infrastructure/outbound/engine-link/client';
import { holdingCall } from '../../../infrastructure/outbound/engine-replay/__tests__/held-call';
import { recordingsOf } from '../../../infrastructure/outbound/engine-replay/recordings';
import { stagedReplay } from '../../../infrastructure/outbound/engine-replay/replay';
import type { Recording } from '../../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../../infrastructure/outbound/opener-tauri/tauri-opener';
import { Window } from '../../navigation/window';
import { PAGES } from '../../screens/registry';
import { EngineProvider, useRead } from '../dependencies';
import { useGesture } from '../use-gesture';
import { counted } from './counted';
import { CHANGES, HELLO, NOTICING, opened, watched } from './over-a-replay';
import { pushed, said } from './pushed';
import { socketEnded } from './socket-ended';

const recordingFor = await recordingsOf('two-macs');

function EngineState({ label }: { label: string }) {
    const engine = useRead('engine.read', {});
    return <p>{engine.kind === 'read' ? `${label}: ${engine.out.State}` : `${label}: reading`}</p>;
}

const STARTING: Recording = {
    exchanges: [
        {
            answers: [{ out: { State: 'stopped' }, v: 2 }],
            request: { call: 'engine.read', in: {} },
        },
    ],
    station: 'dev',
    world: 'test',
};

function Start() {
    const gesture = useGesture('engine.start', {
        input: {},
        rereads: ['engine.read'],
        subject: 'engine',
    });
    const { state } = gesture;
    const task = state.kind === 'running' ? (state.progress.at(-1)?.task ?? '') : '';
    return (
        <>
            <button onClick={gesture.start} type="button">
                {`Start: ${state.kind}`}
            </button>
            <p>{`progress: ${task === '' ? 'none' : task}`}</p>
        </>
    );
}

function upgraded(recording: Recording, version: string): Recording {
    return {
        ...recording,
        exchanges: recording.exchanges.map((exchange) =>
            keyOf(exchange.request) === 'call engine.read {} {}'
                ? {
                      ...exchange,
                      answers: [{ out: { State: 'running', Version: version }, v: 2 }],
                  }
                : exchange,
        ),
    };
}

function readingsIn(html: string): number {
    return html.split('Reading').length;
}

type Counted = ReturnType<typeof counted>;

async function answeredAgain(stage: ReturnType<typeof opened>, count: Counted, ask: () => void) {
    const asked = stage.asked().length;
    const framed = count.frames();
    ask();
    await waitUntil(() => stage.asked().length > asked, { why: 'reads asked again' });
    const again = stage.asked().length - asked;
    await waitUntil(() => count.frames() >= framed + again, {
        why: 'every read asked again answered',
    });
}

async function disturbed(stage: ReturnType<typeof opened>, count: Counted) {
    await answeredAgain(stage, count, () => {
        stage.restart({ ...NOTICING, pid: 42 });
    });
    const restarted = pageDrawn();
    count.refuse('machines.ssh');
    count.notice();
    await waitUntil(() => count.refused() > 0, { why: 'machines.ssh refused' });
    await waitUntil(() => staleMarked() !== null, { why: 'the stale mark drawn' });
    return { marked: staleMarked(), refused: pageDrawn(), restarted };
}

function pageDrawn(): string {
    const below = document.querySelectorAll('main > :not(header)');
    return [...below].map((element) => element.outerHTML).join('');
}

function staleMarked(): null | string {
    return document.querySelector('header [role="status"]')?.getAttribute('aria-label') ?? null;
}

describe('dependencies', () => {
    test('two components reading the same call with their own inputs share one request', async () => {
        // Given - two components that each pass a fresh input object on every render
        const stage = opened(recordingFor('two-macs'));
        const dependencies = {
            engine: new Client(stage.transport),
            opener: NO_OPENER,
            presence: documentPresence(),
        };

        // When - both are rendered and have read the engine
        await component.render(
            <EngineProvider dependencies={dependencies}>
                <EngineState label="sidebar" />
                <EngineState label="page" />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(content('sidebar: running'));
                await visitor.see(content('page: running'));
            },
        );

        // Then - engine.read was asked once, and never again on a re-render
        const asked = stage
            .asked()
            .filter((request) => keyOf(request) === 'call engine.read {} {}');
        expect(asked).toHaveLength(1);
    });

    test("the sidebar's engine line follows an engine that restarts as another version", async () => {
        // Given - the window over two-macs, its engine read as dev
        const recording = recordingFor('two-macs');
        const stage = opened(recording);
        const dependencies = {
            engine: new Client(stage.transport),
            opener: NO_OPENER,
            presence: documentPresence(),
        };

        // When - the engine restarts with another pid and answers as v9
        const result = await component.render(
            <EngineProvider dependencies={dependencies}>
                <Window initial={{ page: 'home' }} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(content('Engine running · dev'));
                stage.restart({ ...HELLO, pid: 42 }, upgraded(recording, 'v9'));
                await visitor.see(content('Engine running · v9'));
            },
        );

        // Then - the window followed without a relaunch, and nothing failed
        expect(result.errors.text).toBe('');
    });

    test('a gesture whose connection drops before it answers is unknown, and its reads are asked again', async () => {
        // Given - a start the spec answers step by step, and engine.read shown beside it
        const stage = stagedReplay(STARTING, {
            hello: HELLO,
            kind: 'open',
            refused: undefined,
            socket: 'replay',
        });
        const start = holdingCall(stage, 'engine.start');
        const dependencies = {
            engine: new Client(start.transport),
            opener: NO_OPENER,
            presence: documentPresence(),
        };

        // When - it is started, sends its progress, and its connection drops while it runs
        const result = await component.render(
            <EngineProvider dependencies={dependencies}>
                <Start />
                <EngineState label="engine" />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(content('engine: stopped'));
                await visitor.click(content('Start: idle'));
                await visitor.see(content('Start: running'));
                await waitUntil(() => start.sent(), { why: 'engine.start sent' });
                start.progress('Starting the engine');
                await visitor.see(content('progress: Starting the engine'));
                start.drop();
                await visitor.see(content('Start: unknown'));
            },
        );

        // Then - it is not failed, and engine.read was asked again after it
        const reads = stage
            .asked()
            .filter((request) => keyOf(request) === 'call engine.read {} {}');
        expect(reads).toHaveLength(2);
        expect(result.errors.text).toBe('');
    });

    test('reads asked again change nothing visible, and draw no refreshing mark', async () => {
        // Given - the window on Home over two-macs, its answers counted as they come
        const stage = opened(recordingFor('two-macs'), NOTICING);
        const count = counted(stage.transport);
        const { drawn, watcher } = watched();
        let before = '';
        let after = '';
        await component.render(
            <EngineProvider
                dependencies={{
                    engine: new Client(count.transport),
                    opener: NO_OPENER,
                    presence: documentPresence(),
                }}
            >
                <Window initial={{ page: 'home' }} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(content('2 things need you.'));
                before = document.body.innerHTML;
                watcher.observe(document.body, CHANGES);

                // When - the engine tells a notice, and every read the window shows is answered again
                await answeredAgain(stage, count, count.notice);
                watcher.disconnect();
                after = document.body.innerHTML;
            },
        );

        // Then - the page reads as before, and no frame in between said it was reading
        expect(after).toBe(before);
        expect(drawn.filter((html) => readingsIn(html) > readingsIn(before))).toStrictEqual([]);
    });

    test('home drawn whole is never drawn reading again by a stream the engine ends by closing its connection', async () => {
        // Given - the window on Home over two-macs, each stream ended as the engine's socket ends it, its Overview first unread
        const stage = opened(recordingFor('two-macs'));
        const socket = socketEnded(stage.transport);
        const { drawn, watcher } = watched();
        await component.render(
            <EngineProvider
                dependencies={{
                    engine: new Client(socket.transport),
                    opener: NO_OPENER,
                    presence: documentPresence(),
                }}
            >
                <Window initial={{ page: 'home' }} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(content('2 things need you.'));
                watcher.observe(document.body, CHANGES);

                // When - the window asks the ended stream again twice, and each new stream answers unread before whole
                const answered = socket.answered();
                await waitUntil(() => socket.answered() >= answered + 2, {
                    timeout: 10_000,
                    why: 'engine.state asked again twice, and answered whole each time',
                });
                watcher.disconnect();
            },
        );

        // Then - no frame drawn since said it was reading the Mac, or dropped its Needs you
        expect(drawn.filter((html) => html.includes('Reading macbook-pro.'))).toStrictEqual([]);
        expect(drawn.filter((html) => !html.includes('2 things need you.'))).toStrictEqual([]);
    });

    test('home is drawn once per pushed revision that moves its views, and never for one equal to the one drawn', async () => {
        // Given - the window on Home over two-macs, each React commit counted
        const stage = opened(recordingFor('two-macs'));
        const engine = pushed(stage.transport);
        let commits = 0;
        let counts: number[] = [];
        await component.render(
            <EngineProvider
                dependencies={{
                    engine: new Client(engine.transport),
                    opener: NO_OPENER,
                    presence: documentPresence(),
                }}
            >
                <Profiler
                    id="window"
                    onRender={() => {
                        commits += 1;
                    }}
                >
                    <Window initial={{ page: 'home' }} pages={PAGES} />
                </Profiler>
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(content('2 things need you.'));
                const committed = commits;

                // When - the engine pushes a revision equal to the one drawn, then one that moves both Overviews, then one that moves them back
                engine.same();
                engine.push(said('1 thing needs you.'));
                await visitor.see(content('1 thing needs you.'));
                const moved = commits;
                engine.push(said('2 things need you.'));
                await visitor.see(content('2 things need you.'));
                counts = [moved - committed, commits - moved];
            },
        );

        // Then - one commit for each revision that moved, none for the equal one
        expect(counts).toStrictEqual([1, 1]);
    });

    test('a page once drawn is never blank again: an equal re-ask, a restart, a refusal and a revisit', async () => {
        // Given - linux-1's Ports over two-macs, each React commit counted, a grace of nothing so a revisit is kept
        const stage = opened(recordingFor('two-macs'), NOTICING);
        const count = counted(stage.transport);
        const { drawn, watcher } = watched();
        let commits = 0;
        let before = '';
        const seen = { equal: -1, marked: null as null | string, refused: '', restarted: '' };
        const roots: (Element | null)[] = [];
        await component.render(
            <EngineProvider
                dependencies={{
                    engine: new Client(count.transport),
                    opener: NO_OPENER,
                    presence: documentPresence(),
                    reads: { grace: 0 },
                }}
            >
                <Profiler
                    id="window"
                    onRender={() => {
                        commits += 1;
                    }}
                >
                    <Window
                        initial={{ machine: 'linux-1', page: 'machine', tab: 'Ports' }}
                        pages={PAGES}
                    />
                </Profiler>
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(content(':5173'));
                before = pageDrawn();
                roots.push(document.querySelector('main'));
                watcher.observe(document.body, CHANGES);

                // When - every read is answered again equal, the engine restarts, machines.ssh is refused, and Ports is left and drawn again
                const committed = commits;
                await answeredAgain(stage, count, count.notice);
                seen.equal = commits - committed;
                Object.assign(seen, await disturbed(stage, count));
                roots.push(document.querySelector('main'));
                await visitor.click(content('Sign-ins'));
                await visitor.see(content('Signed in to 2 of 4 programs.'));
                drawn.length = 0;
                await visitor.click(content('Ports'));
                await visitor.see(content(':5173'));
                watcher.disconnect();
            },
        );

        // Then - no commit on the equal answer, the page unmoved by the restart and the refusal, which the toolbar marks
        // Then - in the engine's words, on one root, and never reading
        expect(seen.equal).toBe(0);
        expect([seen.restarted, seen.refused]).toStrictEqual([before, before]);
        expect(seen.marked).toBe('macbook-pro did not answer');
        expect(roots[1]).toBe(roots[0]);
        expect(drawn.filter((html) => readingsIn(html) > readingsIn(before))).toStrictEqual([]);
    });
});

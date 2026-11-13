import type { Request } from '@jterrazz/station-api';
import { component, content, waitUntil } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { recordingsOf } from '../../fixtures/recordings';
import { PAGES } from '../../pages/registry';
import { Window } from '../../shell/window';
import { Client } from '../client';
import { keyOf } from '../frame';
import { EngineProvider, useRead } from '../react';
import { stagedReplay } from '../replay';
import type { Recording } from '../replay';
import type { Receiver, Transport } from '../transport';
import { useGesture } from '../use-gesture';

const recordingFor = await recordingsOf('two-macs');

const HELLO = { pid: 41, started_at: '2026-11-13T09:00:00Z', version: 'dev' };

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
    gestures: [
        {
            after: [],
            answers: [
                { progress: { task: 'Starting the engine' }, v: 2 },
                { out: {}, v: 2 },
            ],
            request: { call: 'engine.start', in: {} },
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
    return (
        <button onClick={gesture.start} type="button">
            {`Start: ${gesture.state.kind}`}
        </button>
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

function opened(recording: Recording) {
    return stagedReplay(recording, {
        hello: HELLO,
        kind: 'open',
        refused: undefined,
        socket: 'replay',
    });
}

const CHANGES = { characterData: true, childList: true, subtree: true };

function readingsIn(html: string): number {
    return html.split('Reading').length;
}

function countingFrames(transport: Transport) {
    let frames = 0;
    return {
        frames: () => frames,
        transport: {
            link: (listener: Parameters<Transport['link']>[0]) => transport.link(listener),
            open: (request: Request, receiver: Receiver) =>
                transport.open(request, {
                    end: (reason, sent) => {
                        receiver.end(reason, sent);
                    },
                    frame: (frame) => {
                        frames += 1;
                        receiver.frame(frame);
                    },
                }),
        } satisfies Transport,
    };
}

describe('react', () => {
    test('two components reading the same call with their own inputs share one request', async () => {
        // Given - two components that each pass a fresh input object on every render
        const stage = opened(recordingFor('two-macs'));
        const client = new Client(stage.transport);

        // When - both are rendered and have read the engine
        await component.render(
            <EngineProvider client={client}>
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
        const client = new Client(stage.transport);

        // When - the engine restarts with another pid and answers as v9
        const result = await component.render(
            <EngineProvider client={client}>
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
        // Given - a start whose answer comes a second after its progress, and engine.read shown beside it
        const stage = stagedReplay(
            STARTING,
            { hello: HELLO, kind: 'open', refused: undefined, socket: 'replay' },
            { gestureFrame: 1000 },
        );
        const client = new Client(stage.transport);

        // When - it is started, and its connection drops while it runs
        const result = await component.render(
            <EngineProvider client={client}>
                <Start />
                <EngineState label="engine" />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(content('engine: stopped'));
                await visitor.click(content('Start: idle'));
                await visitor.see(content('Start: running'));
                stage.drop();
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
        const stage = opened(recordingFor('two-macs'));
        const counted = countingFrames(stage.transport);
        const drawn: string[] = [];
        const watcher = new MutationObserver(() => {
            drawn.push(document.body.innerHTML);
        });
        let before = '';
        let after = '';
        await component.render(
            <EngineProvider client={new Client(counted.transport)}>
                <Window initial={{ page: 'home' }} pages={PAGES} />
            </EngineProvider>,
            async (visitor) => {
                await visitor.see(content('npm sign-in expired'));
                before = document.body.innerHTML;
                watcher.observe(document.body, CHANGES);

                // When - the window loses the focus and regains it, and every read it shows is answered again
                const asked = stage.asked().length;
                const framed = counted.frames();
                globalThis.dispatchEvent(new Event('blur'));
                globalThis.dispatchEvent(new Event('focus'));
                await waitUntil(() => stage.asked().length > asked, { why: 'reads asked again' });
                const again = stage.asked().length - asked;
                await waitUntil(() => counted.frames() >= framed + again, {
                    why: 'every read asked again answered',
                });
                watcher.disconnect();
                after = document.body.innerHTML;
            },
        );

        // Then - the page reads as before, and no frame in between said it was reading
        expect(after).toBe(before);
        expect(drawn.filter((html) => readingsIn(html) > readingsIn(before))).toStrictEqual([]);
    });
});

import { describe, expect, test } from 'vitest';

import type { CoreEnd, CoreLine, CorePort, CoreRun, Heard } from '../../ports/core.port';
import { UpdateFlow } from '../update-flow';
import type { UpdateState } from '../update-flow';

type Held = {
    act: 'install' | 'start';
    heard: Heard;
    end: (end: CoreEnd) => void;
    executable?: string;
};

const UNHELD: (end: CoreEnd) => void = () => undefined;

function heldCore() {
    const runs: Held[] = [];
    const held =
        (act: Held['act']) =>
        (heard: Heard, executable?: string): CoreRun => {
            let end: (end: CoreEnd) => void = UNHELD;
            const ended = new Promise<CoreEnd>((resolve) => {
                end = resolve;
            });
            runs.push(
                executable === undefined ? { act, end, heard } : { act, end, executable, heard },
            );
            return {
                cancel: () => {
                    end({ cancelled: true, code: 143, kind: 'ended' });
                },
                ended,
            };
        };
    const core: CorePort = {
        install: held('install'),
        present: async () => await Promise.resolve(true),
        start: held('start'),
    };
    return { core, runs };
}

async function settled(flow: UpdateFlow): Promise<UpdateState> {
    await new Promise<void>((resolve) => {
        setImmediate(resolve);
    });
    return flow.state();
}

const ASKED = { executable: '/Users/you/.local/bin/station', to: 'v0.22.0' };
const OUT: CoreLine = { stream: 'out', text: 'Version: v0.22.0' };
const STARTED: CoreLine = { stream: 'out', text: '✓ engine · installed, running · v0.22.0' };
const ERR: CoreLine = {
    stream: 'err',
    text: 'Error: station-darwin-arm64 does not match its checksum',
};
const DONE: CoreEnd = { cancelled: false, code: 0, kind: 'ended' };

describe('update flow', () => {
    test('one press runs the installer, then starts the station the engine named, then is updated', async () => {
        // Given - an update under way
        const { core, runs } = heldCore();
        const flow = new UpdateFlow(core);
        flow.update(ASKED);

        // When - the installer says a line and ends well, then the start says one and ends well
        runs[0]?.heard(OUT);
        runs[0]?.end(DONE);
        const restarting = await settled(flow);
        runs[1]?.heard(STARTED);
        runs[1]?.end(DONE);

        // Then - each step's lines are kept apart, the start ran the engine's station, and the end is updated
        expect(restarting).toStrictEqual({
            kind: 'restarting',
            lines: { install: [OUT], start: [] },
            to: 'v0.22.0',
        });
        expect(runs.map((run) => [run.act, run.executable])).toStrictEqual([
            ['install', undefined],
            ['start', '/Users/you/.local/bin/station'],
        ]);
        await expect(settled(flow)).resolves.toStrictEqual({
            kind: 'updated',
            lines: { install: [OUT], start: [STARTED] },
            to: 'v0.22.0',
        });
    });

    test("a failed install is failed with the installer's last error line, and the engine is not restarted", async () => {
        // Given - an update whose installer says an error
        const { core, runs } = heldCore();
        const flow = new UpdateFlow(core);
        flow.update(ASKED);
        runs[0]?.heard(ERR);

        // When - it exits 1
        runs[0]?.end({ cancelled: false, code: 1, kind: 'ended' });

        // Then - its line is the sentence, and no start ran
        await expect(settled(flow)).resolves.toStrictEqual({
            code: 1,
            kind: 'failed',
            lines: { install: [ERR], start: [] },
            said: ERR.text,
            step: 'install',
            to: 'v0.22.0',
        });
        expect(runs).toHaveLength(1);
    });

    test('a refused start is failed with the refusal as sent, and Retry runs both steps again', async () => {
        // Given - an update whose start the shell refuses
        const { core, runs } = heldCore();
        const flow = new UpdateFlow(core);
        flow.update(ASKED);
        runs[0]?.end(DONE);
        await settled(flow);
        runs[1]?.end({ kind: 'refused', message: 'station is nowhere the window looks' });
        const failed = await settled(flow);

        // When - Retry is pressed
        flow.retry();

        // Then - the refusal was the sentence, and the installer runs again
        expect(failed).toStrictEqual({
            kind: 'failed',
            lines: { install: [], start: [] },
            said: 'station is nowhere the window looks',
            step: 'start',
            to: 'v0.22.0',
        });
        expect(runs.map((run) => run.act)).toStrictEqual(['install', 'start', 'install']);
    });

    test('a cancelled install goes back to idle, and a second press meanwhile runs nothing', async () => {
        // Given - an update under way
        const { core, runs } = heldCore();
        const flow = new UpdateFlow(core);
        flow.update(ASKED);

        // When - Update is pressed again, then Cancel
        flow.update(ASKED);
        flow.cancel();

        // Then - one run only, and the flow is idle
        await expect(settled(flow)).resolves.toStrictEqual({ kind: 'idle' });
        expect(runs).toHaveLength(1);
    });

    test('an ended update is closed by Done, and one still running is not', async () => {
        // Given - an update under way
        const { core, runs } = heldCore();
        const flow = new UpdateFlow(core);
        flow.update(ASKED);

        // When - Done is pressed while it runs, then after it is updated
        flow.done();
        const during = flow.state().kind;
        runs[0]?.end(DONE);
        await settled(flow);
        runs[1]?.end(DONE);
        await settled(flow);
        flow.done();

        // Then - it kept running, then closed
        expect([during, flow.state().kind]).toStrictEqual(['updating', 'idle']);
    });
});

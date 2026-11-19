import { describe, expect, test } from 'vitest';

import type { CoreEnd, CoreLine, CorePort, CoreRun, Heard } from '../../ports/core.port';
import { CoreFlow, saidOf } from '../core-flow';
import type { CoreState } from '../core-flow';

type Held = { heard: Heard; end: (end: CoreEnd) => void };

const UNHELD: (end: CoreEnd) => void = () => undefined;

function heldCore(present: boolean) {
    const runs: Held[] = [];
    const held = (heard: Heard): CoreRun => {
        let end: (end: CoreEnd) => void = UNHELD;
        const ended = new Promise<CoreEnd>((resolve) => {
            end = resolve;
        });
        runs.push({ end, heard });
        return {
            cancel: () => {
                end({ cancelled: true, code: 143, kind: 'ended' });
            },
            ended,
        };
    };
    const core: CorePort = {
        install: held,
        present: async () => await Promise.resolve(present),
        start: held,
    };
    return { core, runs };
}

async function settled(flow: CoreFlow): Promise<CoreState> {
    await new Promise<void>((resolve) => {
        setImmediate(resolve);
    });
    return flow.state();
}

const OUT: CoreLine = { stream: 'out', text: 'Version: v0.21.0' };
const ERR: CoreLine = { stream: 'err', text: 'Error: no release for darwin-amd64' };

describe('core flow', () => {
    test('the disk decides the first panel: no station is the offer, a station is the start', async () => {
        // Given - one flow over a disk without the core, one over a disk with it
        const without = new CoreFlow(heldCore(false).core);
        const within = new CoreFlow(heldCore(true).core);

        // When - each enters with no engine answering
        without.enter();
        within.enter();

        // Then - missing, then installed and not put off
        expect([await settled(without), await settled(within)]).toStrictEqual([
            { kind: 'missing' },
            { kind: 'installed', later: false },
        ]);
    });

    test("an install keeps the installer's lines as sent and ends on the start panel", async () => {
        // Given - a flow whose install is under way
        const { core, runs } = heldCore(false);
        const flow = new CoreFlow(core);
        flow.install();

        // When - it says one line, then exits 0
        runs[0]?.heard(OUT);
        const during = flow.state();
        runs[0]?.end({ cancelled: false, code: 0, kind: 'ended' });

        // Then - the line is kept while it runs, and the end is the installed panel
        expect(during).toStrictEqual({ kind: 'installing', lines: [OUT] });
        await expect(settled(flow)).resolves.toStrictEqual({ kind: 'installed', later: false });
    });

    test('a failing exit is failed with the last error line, never a cause of its own', async () => {
        // Given - an install that says a line on each stream
        const { core, runs } = heldCore(false);
        const flow = new CoreFlow(core);
        flow.install();
        runs[0]?.heard(ERR);
        runs[0]?.heard(OUT);

        // When - it exits 1
        runs[0]?.end({ cancelled: false, code: 1, kind: 'ended' });

        // Then - the sentence is the error line, the lines and the code are kept
        await expect(settled(flow)).resolves.toStrictEqual({
            code: 1,
            kind: 'failed',
            lines: [ERR, OUT],
            of: 'install',
            said: 'Error: no release for darwin-amd64',
        });
    });

    test('a second gesture while a run is under way starts nothing', async () => {
        // Given - an install under way
        const { core, runs } = heldCore(false);
        const flow = new CoreFlow(core);
        flow.install();

        // When - install, start and the terminal path are asked again
        flow.install();
        flow.start();
        flow.waitForTerminal();

        // Then - one run, still installing
        expect(runs).toHaveLength(1);
        await expect(settled(flow)).resolves.toStrictEqual({ kind: 'installing', lines: [] });
    });

    test('a cancelled start goes back to the start panel, a cancelled install to the offer', async () => {
        // Given - a flow over each gesture
        const install = new CoreFlow(heldCore(false).core);
        const start = new CoreFlow(heldCore(true).core);
        install.install();
        start.start();

        // When - each is cancelled
        install.cancel();
        start.cancel();

        // Then - nothing half done is shown
        expect([await settled(install), await settled(start)]).toStrictEqual([
            { kind: 'missing' },
            { kind: 'installed', later: false },
        ]);
    });

    test('an engine that answers ends the flow, unless a run is still under way', async () => {
        // Given - one flow at rest on the offer, one starting
        const resting = new CoreFlow(heldCore(false).core);
        resting.enter();
        await settled(resting);
        const { core } = heldCore(true);
        const starting = new CoreFlow(core);
        starting.start();

        // When - the engine answers both
        resting.answered();
        starting.answered();

        // Then - the resting one is back to looking, the starting one waits for its run
        expect([resting.state(), starting.state()]).toStrictEqual([
            { kind: 'looking' },
            { kind: 'starting', lines: [] },
        ]);
    });

    test('the sentence of a failure is its last error line, else its last line', () => {
        // Given - lines with an error, lines without, and none
        const said = [saidOf([ERR, OUT]), saidOf([OUT, { stream: 'out', text: '  ' }]), saidOf([])];

        // Then - each as sent
        expect(said).toStrictEqual(['Error: no release for darwin-amd64', 'Version: v0.21.0', '']);
    });
});

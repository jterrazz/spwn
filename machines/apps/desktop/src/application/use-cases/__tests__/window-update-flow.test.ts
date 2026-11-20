import { describe, expect, test } from 'vitest';

import type { WindowChecked, WindowInstalled, WindowRefused } from '../../../domain/window-update';
import type { HeardProgress, WindowUpdatePort } from '../../ports/window-update.port';
import { WindowUpdateFlow } from '../window-update-flow';
import type { WindowUpdateState } from '../window-update-flow';

type Install = { heard: HeardProgress; end: (end: WindowInstalled | WindowRefused) => void };

function heldShell(answers: readonly (WindowChecked | WindowRefused)[]) {
    const installs: Install[] = [];
    let checks = 0;
    const port: WindowUpdatePort = {
        check: async () => {
            const answer = answers[Math.min(checks, answers.length - 1)];
            checks += 1;
            return await Promise.resolve(answer ?? { kind: 'unsupported' });
        },
        install: async (heard) =>
            await new Promise((resolve) => {
                installs.push({ end: resolve, heard });
            }),
    };
    return { checked: () => checks, installs, port };
}

async function settled(flow: WindowUpdateFlow): Promise<WindowUpdateState> {
    await new Promise<void>((resolve) => {
        setImmediate(resolve);
    });
    return flow.state();
}

const OUT: WindowChecked = {
    current: '0.21.0',
    date: '2026-11-19T09:00:00Z',
    kind: 'out',
    version: '0.22.0',
};
const RELEASE = { current: '0.21.0', date: '2026-11-19T09:00:00Z', version: '0.22.0' };
const BROKEN: WindowRefused = {
    code: 'window.update-install',
    kind: 'refused',
    message: 'Permission denied (os error 13)',
};

describe('window update flow', () => {
    test('start checks once, and a second start asks the shell nothing', async () => {
        // Given - a shell whose release is current
        const shell = heldShell([{ current: '0.21.0', kind: 'current' }]);
        const flow = new WindowUpdateFlow(shell.port);

        // When - the window starts twice
        flow.start();
        const first = flow.state();
        flow.start();

        // Then - a first check, then the current version, from one ask
        expect(first).toStrictEqual({ first: true, kind: 'checking' });
        await expect(settled(flow)).resolves.toStrictEqual({ current: '0.21.0', kind: 'current' });
        expect(shell.checked()).toBe(1);
    });

    test('an install out counts its download, then relaunches', async () => {
        // Given - a release out
        const shell = heldShell([OUT]);
        const flow = new WindowUpdateFlow(shell.port);
        flow.start();
        await settled(flow);

        // When - Install and relaunch is pressed, and the shell hears two chunks then the install
        flow.install();
        shell.installs[0]?.heard({ downloaded: 4_000_000, kind: 'downloaded', total: 12_000_000 });
        const downloading = flow.state();
        shell.installs[0]?.heard({ kind: 'installed' });

        // Then - the download counted, then relaunching
        expect(downloading).toStrictEqual({
            ...RELEASE,
            downloaded: 4_000_000,
            kind: 'installing',
            total: 12_000_000,
        });
        expect(flow.state()).toStrictEqual({ ...RELEASE, kind: 'relaunching' });
    });

    test('a refused check is refused in the shell words, and Check now asks again', async () => {
        // Given - a shell that cannot reach the releases, then finds the window current
        const shell = heldShell([
            {
                code: 'window.update-unreachable',
                kind: 'refused',
                message: 'error sending request',
            },
            { current: '0.21.0', kind: 'current' },
        ]);
        const flow = new WindowUpdateFlow(shell.port);
        flow.start();
        const refused = await settled(flow);

        // When - Check now is pressed
        flow.checkNow();
        const checking = flow.state();

        // Then - refused, a check that is not the first, then current
        expect(refused).toStrictEqual({
            code: 'window.update-unreachable',
            kind: 'refused',
            message: 'error sending request',
        });
        expect(checking).toStrictEqual({ first: false, kind: 'checking' });
        await expect(settled(flow)).resolves.toStrictEqual({ current: '0.21.0', kind: 'current' });
    });

    test('a refused install keeps the release, and Try again installs it with no new check', async () => {
        // Given - an install the shell refuses
        const shell = heldShell([OUT]);
        const flow = new WindowUpdateFlow(shell.port);
        flow.start();
        await settled(flow);
        flow.install();
        shell.installs[0]?.end(BROKEN);
        const failed = await settled(flow);

        // When - Try again is pressed
        flow.retry();

        // Then - failed with the release and the refusal, then a second install from one check
        expect(failed).toStrictEqual({
            ...RELEASE,
            code: 'window.update-install',
            kind: 'failed',
            message: 'Permission denied (os error 13)',
        });
        expect(flow.state()).toStrictEqual({ ...RELEASE, downloaded: 0, kind: 'installing' });
        expect([shell.installs.length, shell.checked()]).toStrictEqual([2, 1]);
    });

    test('a press while installing runs nothing, and Try again outside a failure runs nothing', async () => {
        // Given - an install under way
        const shell = heldShell([OUT]);
        const flow = new WindowUpdateFlow(shell.port);
        flow.start();
        await settled(flow);
        flow.install();

        // When - Install, Try again and Check now are pressed meanwhile
        flow.install();
        flow.retry();
        flow.checkNow();

        // Then - one install, one check, still installing
        expect([shell.installs.length, shell.checked(), flow.state().kind]).toStrictEqual([
            1,
            1,
            'installing',
        ]);
        shell.installs[0]?.end({ kind: 'installed' });
        await expect(settled(flow)).resolves.toStrictEqual({ ...RELEASE, kind: 'relaunching' });
    });
});

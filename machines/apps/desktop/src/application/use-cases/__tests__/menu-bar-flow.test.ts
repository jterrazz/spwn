import { waitUntil } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import type { MenuBarRead, MenuBarRefused } from '../../../domain/menu-bar';
import type { MenuBarPort } from '../../ports/menu-bar.port';
import { MenuBarFlow } from '../menu-bar-flow';
import type { MenuBarState } from '../menu-bar-flow';

type Answer = MenuBarRead | MenuBarRefused;

function portOf(reads: Answer[], shows: Answer[]) {
    const asked: boolean[] = [];
    const port: MenuBarPort = {
        read: async () => await Promise.resolve(reads.shift() ?? { kind: 'unsupported' }),
        show: async (shown) => {
            asked.push(shown);
            return await Promise.resolve(shows.shift() ?? { kind: 'unsupported' });
        },
    };
    return { asked, port };
}

// value: the flow once its last ask answered: neither before its first read nor mid-switch.
async function settled(flow: MenuBarFlow): Promise<MenuBarState> {
    await waitUntil(() => !['switching', 'unknown'].includes(flow.state().kind), {
        why: 'the flow answered',
    });
    return flow.state();
}

describe('menu bar flow', () => {
    test('a look reads what macOS holds, a press switches it, and the state is the read-back answer', async () => {
        // Given - a menu bar hidden, whose show answers that it waits for the owner's approval
        const { asked, port } = portOf([{ kind: 'hidden' }], [{ kind: 'approval' }]);
        const flow = new MenuBarFlow(port);

        // When - Settings looks, then the switch is pressed on
        const before = flow.state();
        flow.look();
        const looked = await settled(flow);
        flow.show(true);
        const switching = flow.state();
        const after = await settled(flow);

        // Then - unknown, hidden, switching from hidden, then approval as read back
        expect([before, looked, switching, after]).toStrictEqual([
            { kind: 'unknown' },
            { held: 'hidden', kind: 'held' },
            { held: 'hidden', kind: 'switching', to: true },
            { held: 'approval', kind: 'held' },
        ]);
        expect(asked).toStrictEqual([true]);
    });

    test('a refused press keeps what was held with the refusal, a refused read has nothing held, and a window with no switch is unsupported', async () => {
        // Given - a shown menu bar whose hide is refused, a read refused, and a shell with no switch
        const refused: MenuBarRefused = {
            code: 'menubar.not-switched',
            kind: 'refused',
            message: 'Operation not permitted',
        };
        const switched = new MenuBarFlow(portOf([{ kind: 'shown' }], [refused]).port);
        const unread = new MenuBarFlow(
            portOf([{ code: 'menubar.not-read', kind: 'refused', message: 'no answer' }], []).port,
        );
        const none = new MenuBarFlow(portOf([], []).port);

        // When - each looks, the first is pressed off, and none is pressed at all
        switched.look();
        await settled(switched);
        switched.show(false);
        unread.look();
        none.look();
        await settled(none);
        none.show(true);

        // Then - the refusal over the shown menu bar, the read's refusal, unsupported
        expect([await settled(switched), await settled(unread), await settled(none)]).toStrictEqual(
            [
                {
                    code: 'menubar.not-switched',
                    held: 'shown',
                    kind: 'failed',
                    message: 'Operation not permitted',
                },
                { code: 'menubar.not-read', kind: 'refused', message: 'no answer' },
                { kind: 'unsupported' },
            ],
        );
    });
});

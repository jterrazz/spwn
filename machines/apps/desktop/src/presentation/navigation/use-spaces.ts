import type { MachinesMacView } from '@jterrazz/station-api';
import { useCallback, useEffect, useMemo, useRef } from 'react';

import { useShortcuts } from './keys';
import type { Acts } from './keys';
import type { Route } from './route';
import type { Router } from './router';
import { Soon } from './soon';
import { inSpace, landingOf, opensNow, spaceOfRoute } from './spaces';
import type { Space } from './spaces';

export type KeysOn = { on: boolean; withGallery: boolean };

// invariant: a space opens where this window last was in it, kept per space for the window's life, so ⌘1 and the
// invariant: Machine word return to the page left for Home or Inbox; a space not built opens its Soon list and moves
// invariant: no route.
export function useSpaceKeys(
    router: Router,
    macs: readonly MachinesMacView[],
    { onFind, onColumn }: Pick<Acts, 'onColumn' | 'onFind'>,
    { on, withGallery }: KeysOn,
): { onSpace: (space: Space) => void; soon: Soon } {
    const { route, go } = router;
    const soon = useMemo(() => new Soon(), []);
    const last = useRef<Partial<Record<Space, Route>>>({});
    useEffect(() => {
        const space = spaceOfRoute(route);
        if (space !== undefined && inSpace(route)) {
            last.current = { ...last.current, [space]: route };
        }
    }, [route]);
    const firstMac = macs[0]?.Name;
    const onSpace = useCallback(
        (space: Space) => {
            if (!opensNow(space)) {
                soon.toggle(space);
                return;
            }
            soon.close();
            go(landingOf(last.current[space], firstMac));
        },
        [go, firstMac, soon],
    );
    const acts = useMemo(() => ({ onColumn, onFind, onSpace }), [onColumn, onFind, onSpace]);
    useShortcuts(router, acts, withGallery, on);
    return { onSpace, soon };
}

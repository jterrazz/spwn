import type { MachinesMacView } from '@jterrazz/station-api';
import { useCallback, useEffect, useMemo, useRef } from 'react';

import { useShortcuts } from './keys';
import type { Route } from './route';
import type { Router } from './router';
import { inSpace, landingOf } from './spaces';
import type { Space } from './spaces';

export type KeysOn = { on: boolean; withGallery: boolean };

// invariant: a space opens where this window last was in it, so ⌃1 and the Machine segment return to the page left
// invariant: for Home or Inbox.
export function useSpaceKeys(
    router: Router,
    macs: readonly MachinesMacView[],
    onFind: () => void,
    { on, withGallery }: KeysOn,
): (space: Space) => void {
    const { route, go } = router;
    const last = useRef<Route | undefined>(undefined);
    useEffect(() => {
        if (inSpace(route)) {
            last.current = route;
        }
    }, [route]);
    const firstMac = macs[0]?.Name;
    const onSpace = useCallback(
        (space: Space) => {
            if (space === 'machine') {
                go(landingOf(last.current, firstMac));
            }
        },
        [go, firstMac],
    );
    const acts = useMemo(() => ({ onFind, onSpace }), [onFind, onSpace]);
    useShortcuts(router, acts, withGallery, on);
    return onSpace;
}

import type { MachinesMacView } from '@jterrazz/station-api';
import { useEffect, useState } from 'react';

import { Button } from '../design/index';
import { findMachine } from '../model/machines';
import type { Route } from './route';
import type { Router } from './router';

import styles from './window.module.css';

function missingOf(route: Route, macs: readonly MachinesMacView[]): string | undefined {
    return route.page === 'machine' && findMachine(macs, route.machine) === undefined
        ? route.machine
        : undefined;
}

// invariant: the route the window opened on, when the engine's whole listing (its watch done) does not hold its
// invariant: machine, is replaced by Home. A first answer that lists this Mac alone is no proof another machine is
// invariant: gone, and a route taken since is never judged: a machine just made may not be listed yet.
export function useLost(router: Router, macs: readonly MachinesMacView[], listed: boolean) {
    const { route, replace } = router;
    const [lost, setLost] = useState<string | undefined>(undefined);
    const [opening, setOpening] = useState<Route | undefined>(route);
    const missing = route === opening && listed ? missingOf(route, macs) : undefined;
    if (missing !== undefined && lost !== missing) {
        setLost(missing);
    }
    if (listed && opening !== undefined && missing === undefined) {
        setOpening(undefined);
    }
    if (lost !== undefined && missing === undefined && route.page !== 'home') {
        setLost(undefined);
    }
    useEffect(() => {
        if (missing !== undefined) {
            replace({ page: 'home' });
        }
    }, [missing, replace]);
    return {
        dismiss: () => {
            setLost(undefined);
        },
        lost: route.page === 'home' ? lost : undefined,
    };
}

export function Lost({ machine, onDismiss }: { machine: string; onDismiss: () => void }) {
    return (
        <div className={styles.lost} role="status">
            <span>
                {`The window was on ${machine}, which the engine no longer lists, so it opened Home.`}
            </span>
            <Button onClick={onDismiss} variant="link">
                Dismiss
            </Button>
        </div>
    );
}

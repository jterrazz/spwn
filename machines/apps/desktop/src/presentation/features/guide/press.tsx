import type { CallName, CoreGuideGesture, MachinesMacView } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import { findMachine, targetOf } from '../../../domain/machines/machines';
import { useGesture } from '../../dependencies/use-gesture';
import { useStart } from '../../kit/gestures';
import { machineRoute } from '../../navigation/route';
import type { Route } from '../../navigation/route';
import type { GuideAction } from '../../ui/guide';
import { repairOf } from '../../words/provisional';
import type { Repair } from '../../words/provisional';
import { routeOfNav } from './guide-of';

const REPAIR_REREADS: readonly CallName[] = ['overview.read'];

type Ground = {
    macs: readonly MachinesMacView[];
    go: (to: Route) => void;
    leave: () => void;
};

type Run = Ground & {
    gesture: CoreGuideGesture;
    repair: Repair;
    children: (action: GuideAction | undefined) => ReactNode;
};

// invariant: a repair runs the call the engine named, at the machine it named, and asks the confirmation the call
// invariant: requires through the window's own dialog, the name typed being the engine's `confirm`; once made, the Guide
// invariant: is left for the page of the machine, or Home when the machine is gone.
function Repairing({ gesture, macs, go, leave, repair, children }: Run) {
    const on = gesture.on ?? '';
    const found = findMachine(macs, on);
    const start = useStart();
    const run = useGesture(repair.call, {
        input: repair.input,
        onDone: () => {
            leave();
            go(
                repair.call === 'machines.delete'
                    ? { page: 'home' }
                    : machineRoute(repair.input.VM ?? ''),
            );
        },
        rereads: REPAIR_REREADS,
        subject: repair.input.VM,
        target: found === undefined ? (on === '' ? undefined : { name: on }) : targetOf(found),
        typed: gesture.confirm,
    });
    return (
        <>
            {children({
                label: gesture.label,
                onPress: () => {
                    start(run, {
                        consequence: undefined,
                        destroys: gesture.confirm !== undefined,
                        held: undefined,
                        title: `${gesture.label}?`,
                        verb: gesture.label,
                    });
                },
            })}
        </>
    );
}

type Props = Ground & {
    gesture: CoreGuideGesture | undefined;
    children: (action: GuideAction | undefined) => ReactNode;
};

export function Pressing({ gesture, children, ...ground }: Props) {
    if (gesture === undefined) {
        return <>{children(undefined)}</>;
    }
    const route = routeOfNav(gesture.nav ?? '');
    if (route !== undefined) {
        return (
            <>
                {children({
                    label: gesture.label,
                    onPress: () => {
                        ground.leave();
                        ground.go(route);
                    },
                })}
            </>
        );
    }
    const repair = repairOf(gesture);
    if (repair === undefined) {
        return <>{children(undefined)}</>;
    }
    return (
        <Repairing {...ground} gesture={gesture} repair={repair}>
            {children}
        </Repairing>
    );
}

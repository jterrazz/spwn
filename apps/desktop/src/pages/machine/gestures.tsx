import type { CallName, MachinesMacView, MachinesVM } from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactNode } from 'react';

import { targetOf } from '../../domain/machines/machines';
import { machineKindOf } from '../../engine/provisional';
import { verdictShown } from '../../engine/shown';
import { useGesture } from '../../engine/use-gesture';
import type { Route } from '../../shell/route';
import { GestureButton, GestureOutcome, marked } from '../kit/gestures';

import styles from './machine.module.css';

const LIFECYCLE_REREADS: readonly CallName[] = ['machines.list', 'overview.read'];

type VmProps = { mac: MachinesMacView; vm: MachinesVM };

type Lifecycle = 'restart' | 'start' | 'stop';

// invariant: the header is drawn again when machines.list is read again, so the gesture it shows the outcome of is
// invariant: found again in the store, which outlives it.
const LIFECYCLES: readonly Lifecycle[] = ['restart', 'start', 'stop'];

const LABELS: Readonly<Record<Lifecycle, string>> = {
    restart: 'Restart',
    start: 'Start',
    stop: 'Stop',
};

function useLifecycle({ mac, vm }: VmProps) {
    const target = targetOf({ mac });
    const spec = {
        kind: machineKindOf(vm),
        rereads: LIFECYCLE_REREADS,
        subject: vm.Name,
        target,
    };
    return {
        restart: useGesture('machines.restart', { ...spec, input: { Name: vm.Name } }),
        start: useGesture('machines.start', { ...spec, input: { VM: vm.Name } }),
        stop: useGesture('machines.stop', { ...spec, input: { VM: vm.Name } }),
    };
}

export function LifecycleGestures({ mac, vm, terminal }: VmProps & { terminal?: ReactNode }) {
    const gestures = useLifecycle({ mac, vm });
    const [last, setLast] = useState<Lifecycle>(
        () => LIFECYCLES.find((which) => gestures[which].state.kind !== 'idle') ?? 'stop',
    );
    const shown = gestures[last];
    const done = shown.state.kind === 'done';
    const leads = vm.Health !== undefined && verdictShown(vm.Health).tone === 'bad';
    const button = (which: Lifecycle) => {
        const label = LABELS[which];
        const primary = which === 'restart' && leads;
        return (
            <GestureButton
                gesture={marked(gestures[which], () => {
                    setLast(which);
                })}
                key={which}
                label={label}
                title={`${label} ${vm.Name}?`}
                variant={primary ? 'primary' : 'plain'}
                yields={primary}
            />
        );
    };
    return (
        <div className={styles.gestures}>
            <div className={styles.actions}>
                {done ? <GestureOutcome gesture={shown} /> : null}
                {(vm.Gestures ?? []).map(button)}
                {terminal}
            </div>
            {done ? null : <GestureOutcome gesture={shown} />}
        </div>
    );
}

export function DeleteGesture({ mac, vm, go }: VmProps & { go: (to: Route) => void }) {
    const remove = useGesture('machines.delete', {
        input: { VM: vm.Name },
        onDone: () => {
            go({ page: 'home' });
        },
        rereads: LIFECYCLE_REREADS,
        subject: vm.Name,
        target: targetOf({ mac }),
    });
    return (
        <div className={styles.destructive}>
            <GestureButton
                gesture={remove}
                label={`Delete ${vm.Name}…`}
                title={`Delete ${vm.Name}?`}
                variant="danger"
                verb="Delete"
            />
            <GestureOutcome gesture={remove} />
        </div>
    );
}

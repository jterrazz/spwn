import type { CallName, MachinesMacView, MachinesVM } from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactNode } from 'react';

import { machineKindOf } from '../../engine/provisional';
import { useGesture } from '../../engine/use-gesture';
import { targetOf } from '../../model/machines';
import type { Route } from '../../shell/route';
import { GestureButton, GestureOutcome, marked } from '../kit/gestures';

import styles from './machine.module.css';

const LIFECYCLE_REREADS: readonly CallName[] = ['machines.list', 'overview.read'];

type VmProps = { mac: MachinesMacView; vm: MachinesVM };

type Lifecycle = 'restart' | 'start' | 'stop';

// invariant: the header is drawn again when machines.list is read again, so the gesture it shows the outcome of is
// invariant: found again in the store, which outlives it.
const LIFECYCLES: readonly Lifecycle[] = ['restart', 'start', 'stop'];

function useLifecycle({ mac, vm }: VmProps) {
    const target = targetOf({ mac });
    const spec = { rereads: LIFECYCLE_REREADS, subject: vm.Name, target };
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
    const button = (which: Lifecycle, label: string) => (
        <GestureButton
            gesture={marked(gestures[which], () => {
                setLast(which);
            })}
            label={label}
            title={`${label} ${vm.Name}?`}
        />
    );
    return (
        <div className={styles.gestures}>
            <div className={styles.actions}>
                {done ? <GestureOutcome gesture={shown} /> : null}
                {button('start', 'Start')}
                {button('stop', 'Stop')}
                {machineKindOf(vm) === 'server' ? button('restart', 'Restart') : null}
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
                verb={`Delete ${vm.Name}`}
            />
            <GestureOutcome gesture={remove} />
        </div>
    );
}

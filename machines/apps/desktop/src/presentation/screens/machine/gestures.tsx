import type { CallName, MachinesMacView, MachinesVM } from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactNode } from 'react';

import { isRunning, targetOf } from '../../../domain/machines/machines';
import { useGesture } from '../../dependencies/use-gesture';
import { GestureButton, GestureOutcome, marked, useStart } from '../../kit/gestures';
import type { Route } from '../../navigation/route';
import { wordOf } from '../../words/engine-words';
import { machineKindOf } from '../../words/provisional';
import { holdAgainst, verdictShown } from '../../words/shown';

import styles from './machine.module.css';

const LIFECYCLE_REREADS: readonly CallName[] = ['overview.read'];

type VmProps = { mac: MachinesMacView; vm: MachinesVM };

type Lifecycle = 'restart' | 'start' | 'stop' | 'try-again';

// invariant: the header is drawn again when machines.list is read again, so the gesture it shows the outcome of is
// invariant: found again in the store, which outlives it.
const LIFECYCLES: readonly Lifecycle[] = ['restart', 'start', 'stop', 'try-again'];

function useLifecycle({ mac, vm }: VmProps) {
    const target = targetOf({ mac });
    const spec = {
        kind: machineKindOf(vm),
        rereads: LIFECYCLE_REREADS,
        subject: vm.Name,
        target,
    };
    return {
        restart: useGesture('machines.restart', {
            ...spec,
            asks: vm.Asks?.Restart,
            input: { Name: vm.Name },
        }),
        start: useGesture('machines.start', { ...spec, input: { VM: vm.Name } }),
        stop: useGesture('machines.stop', { ...spec, asks: vm.Asks?.Stop, input: { VM: vm.Name } }),
        'try-again': useGesture('machines.start', { ...spec, input: { VM: vm.Name } }),
    };
}

export function LifecycleGestures({ mac, vm, beside }: VmProps & { beside?: ReactNode }) {
    const gestures = useLifecycle({ mac, vm });
    const [last, setLast] = useState<Lifecycle>(
        () => LIFECYCLES.find((which) => gestures[which].state.kind !== 'idle') ?? 'stop',
    );
    const shown = gestures[last];
    const done = shown.state.kind === 'done';
    const leads = vm.Health !== undefined && verdictShown(vm.Health).tone === 'bad';
    const button = (which: Lifecycle) => {
        const label = wordOf('MachinesGesture', which).Word;
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
                {beside}
            </div>
            {done ? null : <GestureOutcome gesture={shown} />}
        </div>
    );
}

// value: what a delete takes with it, said in its question: "Delete linux-1 and its 40 GB disk?".
export function deleteTitle(vm: MachinesVM): string {
    return vm.SizeGB > 0
        ? `Delete ${vm.Name} and its ${vm.SizeGB} GB disk?`
        : `Delete ${vm.Name} and its disk?`;
}

export function DeleteGesture({ mac, vm, go }: VmProps & { go: (to: Route) => void }) {
    const started = useStart();
    const target = targetOf({ mac });
    const stop = useGesture('machines.stop', {
        asks: vm.Asks?.Stop,
        input: { VM: vm.Name },
        kind: machineKindOf(vm),
        rereads: LIFECYCLE_REREADS,
        subject: vm.Name,
        target,
    });
    const remove = useGesture('machines.delete', {
        asks: vm.Asks?.Delete,
        input: { VM: vm.Name },
        onDone: () => {
            go({ page: 'home' });
        },
        rereads: LIFECYCLE_REREADS,
        subject: vm.Name,
        target,
    });
    // invariant: a VM that runs offers the safe way out of its delete: stop it, its disk kept.
    const instead = isRunning(vm)
        ? {
              label: 'Stop instead',
              run: () => {
                  started(stop, { destroys: false, title: `Stop ${vm.Name}?`, verb: 'Stop' });
              },
          }
        : undefined;
    return (
        <div className={styles.destructive}>
            <GestureButton
                gesture={remove}
                held={holdAgainst(vm.Hold)}
                instead={instead}
                label={`Delete ${vm.Name}…`}
                title={deleteTitle(vm)}
                variant="danger"
                verb="Delete"
            />
            <GestureOutcome gesture={remove} />
        </div>
    );
}

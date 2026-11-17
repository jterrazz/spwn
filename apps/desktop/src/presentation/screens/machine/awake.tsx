import type { AwakeChoice, CallName } from '@jterrazz/station-api';
import { useState } from 'react';

import { targetOf } from '../../../domain/machines/machines';
import type { Found } from '../../../domain/machines/machines';
import { useWatch } from '../../dependencies/dependencies';
import { useGesture } from '../../dependencies/use-gesture';
import { GestureButton, GestureOutcome } from '../../kit/gestures';
import { Reading, Unanswered } from '../../kit/readings';
import { Button, Dialog } from '../../ui/index';
import { AWAKE_GESTURE_WORDS } from '../../words/words';

import styles from './machine.module.css';

const AWAKE_REREADS: readonly CallName[] = ['awake.read', 'machines.list', 'overview.read'];

type SheetProps = { found: Found; onClose: () => void };

function Choice({ choice, found, onClose }: SheetProps & { choice: AwakeChoice }) {
    const keep = useGesture('awake.keep', {
        input: choice.For === undefined ? { Until: choice.Until ?? '' } : { For: choice.For },
        onDone: onClose,
        rereads: AWAKE_REREADS,
        subject: choice.Label,
        target: targetOf(found),
    });
    return (
        <div className={styles.choice}>
            <GestureButton
                gesture={keep}
                label={choice.Label}
                title={`Keep ${found.mac.Name} awake ${choice.Label}?`}
            />
            <GestureOutcome gesture={keep} />
        </div>
    );
}

function StopKeeping({ found, onClose }: SheetProps) {
    const stop = useGesture('awake.off', {
        input: {},
        onDone: onClose,
        rereads: AWAKE_REREADS,
        subject: found.mac.Name,
        target: targetOf(found),
    });
    return (
        <div className={styles.choice}>
            <GestureButton
                gesture={stop}
                label={AWAKE_GESTURE_WORDS.stop}
                title={`${AWAKE_GESTURE_WORDS.stop}?`}
                variant="link"
            />
            <GestureOutcome gesture={stop} />
        </div>
    );
}

function AwakeSheet({ found, onClose }: SheetProps) {
    const awake = useWatch('awake.read', {}, targetOf(found));
    const title = `Keep ${found.mac.Name} awake`;
    const close = <Button onClick={onClose}>{AWAKE_GESTURE_WORDS.close}</Button>;
    if (awake.kind !== 'read') {
        return (
            <Dialog actions={close} onClose={onClose} title={title}>
                {awake.kind === 'failed' ? (
                    <Unanswered error={awake.error} size="panel" />
                ) : (
                    <Reading size="tab" what="Awake" />
                )}
            </Dialog>
        );
    }
    return (
        <Dialog actions={close} onClose={onClose} title={title}>
            <div className={styles.choices}>
                {(awake.out.Choices ?? []).map((choice) => (
                    <Choice choice={choice} found={found} key={choice.Label} onClose={onClose} />
                ))}
                {awake.out.Session === null ? null : (
                    <StopKeeping found={found} onClose={onClose} />
                )}
            </div>
        </Dialog>
    );
}

export function AwakeButton({ found }: { found: Found }) {
    const [open, setOpen] = useState(false);
    return (
        <>
            <Button
                name={`Keep ${found.mac.Name} awake`}
                onClick={() => {
                    setOpen(true);
                }}
            >
                {AWAKE_GESTURE_WORDS.open}
            </Button>
            {open ? (
                <AwakeSheet
                    found={found}
                    onClose={() => {
                        setOpen(false);
                    }}
                />
            ) : null}
        </>
    );
}

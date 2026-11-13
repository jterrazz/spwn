import type { CallName, Target } from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactNode } from 'react';

import { AppIcon, Button } from '../../design/index';
import { looksSecret } from '../../design/secret';
import type { EngineError } from '../../engine/engine-error';
import type { GestureState } from '../../engine/gesture';
import type { Shown } from '../../engine/shown';
import { useGesture } from '../../engine/use-gesture';
import type { Gesture } from '../../engine/use-gesture';
import { GestureButton, GestureOutcome, GestureRow, marked } from '../kit/gestures';

import styles from './accounts.module.css';

const ACCOUNTS_REREADS: readonly CallName[] = ['accounts.list'];

// value: accounts.share's For is a Go time.Duration, nanoseconds on the wire.
const TWELVE_HOURS = 12 * 3600 * 1_000_000_000;

function saidAnything(error: EngineError): readonly string[] {
    return [error.message, JSON.stringify(error.wire ?? {})];
}

function holdsSecret(state: GestureState<unknown>): boolean {
    if (state.kind === 'failed') {
        return saidAnything(state.error).some(looksSecret);
    }
    if (state.kind === 'running') {
        const last = state.progress.at(-1);
        return [last?.detail, last?.error?.message].some(
            (said) => said !== undefined && looksSecret(said),
        );
    }
    return false;
}

export function ScreenedOutcome({ gesture }: { gesture: Gesture<unknown> }) {
    if (!holdsSecret(gesture.state)) {
        return <GestureOutcome gesture={gesture} />;
    }
    return (
        <span className={styles.withheld}>
            Withheld: the engine answered what looks like a secret, never drawn here.
            {gesture.state.kind === 'failed' ? (
                <Button onClick={gesture.dismiss} variant="link">
                    Dismiss
                </Button>
            ) : null}
        </span>
    );
}

export type Held = { account: string; shown: string; vm: string; target: Target | undefined };

export function useTakeBack({ account, shown, vm, target }: Held): Gesture<unknown> {
    return useGesture('accounts.take-back', {
        input: { Account: account, From: vm },
        rereads: ACCOUNTS_REREADS,
        subject: `${shown} from ${vm}`,
        target,
    });
}

type TakeBackProps = { gesture: Gesture<unknown>; shown: string; vm: string };

export function TakeBackButton({ gesture, shown, vm }: TakeBackProps) {
    return (
        <GestureButton
            gesture={gesture}
            label="Take back"
            name={`Take back ${shown}`}
            title={`Take back ${shown} from ${vm}?`}
            variant="link"
        />
    );
}

type ShareRowProps = Held & { holds: Shown; lead: ReactNode; sentence?: string | undefined };

export function ShareRow({ holds, lead, sentence, ...held }: ShareRowProps) {
    const { account, shown, vm, target } = held;
    const extend = useGesture('accounts.share', {
        input: { Account: account, For: TWELVE_HOURS, With: vm },
        rereads: ACCOUNTS_REREADS,
        subject: `${shown} with ${vm}`,
        target,
    });
    const back = useTakeBack(held);
    const [last, setLast] = useState<'back' | 'extend'>('extend');
    const shownGesture = last === 'extend' ? extend : back;
    const icon = <AppIcon name={looksSecret(account) ? '' : account} />;
    return (
        <GestureRow
            gesture={shownGesture}
            lead={
                lead === undefined ? (
                    icon
                ) : (
                    <span className={styles.leads}>
                        {icon}
                        {lead}
                    </span>
                )
            }
            leadSize={lead === undefined ? 'icon' : 'chip'}
            outcome={<ScreenedOutcome gesture={shownGesture} />}
            sentence={sentence ?? ''}
            state={{ ...holds, asWritten: true }}
            title={shown}
            trail={
                <span className={styles.trail}>
                    <GestureButton
                        gesture={marked(extend, () => {
                            setLast('extend');
                        })}
                        label="Extend 12 h"
                        name={`Extend ${shown} by 12 h`}
                        title={`Extend ${shown} with ${vm} by 12 h?`}
                    />
                    <TakeBackButton
                        gesture={marked(back, () => {
                            setLast('back');
                        })}
                        shown={shown}
                        vm={vm}
                    />
                </span>
            }
        />
    );
}

import type { AwakeChoice, AwakeReport, CallName, CoreVerdictVerdict } from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactElement } from 'react';

import { targetOf } from '../../../domain/machines/machines';
import type { Found } from '../../../domain/machines/machines';
import { useWatch } from '../../dependencies/dependencies';
import { useGesture } from '../../dependencies/use-gesture';
import type { Gesture } from '../../dependencies/use-gesture';
import { GestureButton, GestureOutcome, useStart } from '../../kit/gestures';
import { Unanswered } from '../../kit/readings';
import { Skeleton, SkeletonRows } from '../../kit/skeleton';
import { Button, Dialog, StateSign, Switch } from '../../ui/index';
import { AWAKE_GESTURE_WORDS } from '../../words/words';

import styles from './machine.module.css';

const AWAKE_REREADS: readonly CallName[] = ['awake.read', 'overview.read'];

type SheetProps = { found: Found; onClose: () => void };

type Kept = { choice: AwakeChoice; gesture: Gesture<unknown> };

function Keep({
    choice,
    found,
    onClose,
    children,
}: SheetProps & { choice: AwakeChoice; children: (kept: Kept) => ReactElement }) {
    const gesture = useGesture('awake.keep', {
        input: choice.For === undefined ? { Until: choice.Until ?? '' } : { For: choice.For },
        onDone: onClose,
        rereads: AWAKE_REREADS,
        subject: choice.Label,
        target: targetOf(found),
    });
    return children({ choice, gesture });
}

// invariant: one keep gesture per choice, however many the engine sends: each level holds the first choice's
// invariant: and hands the rest on, so the row is drawn once every choice has its own.
function Keeps({
    choices,
    found,
    onClose,
    children,
}: SheetProps & {
    choices: readonly AwakeChoice[];
    children: (kept: readonly Kept[]) => ReactElement;
}) {
    const [first, ...rest] = choices;
    if (first === undefined) {
        return children([]);
    }
    return (
        <Keep choice={first} found={found} onClose={onClose}>
            {(one) => (
                <Keeps choices={rest} found={found} onClose={onClose}>
                    {(others) => children([one, ...others])}
                </Keeps>
            )}
        </Keep>
    );
}

function AwakeState({ verdict }: { verdict: CoreVerdictVerdict | undefined }) {
    if (verdict === undefined) {
        return null;
    }
    const line = verdict.Line ?? '';
    return (
        <div className={styles.awakeState}>
            <p className={styles.awakeSentence}>
                <StateSign tone={verdict.Tone} />
                {verdict.Sentence}
            </p>
            {line === '' ? null : <p className={styles.awakeLine}>{line}</p>}
        </div>
    );
}

function Terms({ found, kept, held }: { found: Found; kept: readonly Kept[]; held: boolean }) {
    const start = useStart();
    if (kept.length === 0) {
        return null;
    }
    const picked = kept.findIndex(({ choice }) => choice.Picked ?? false);
    return (
        <Switch
            disabled={held}
            even
            label={AWAKE_GESTURE_WORDS.howLong}
            onChange={(at) => {
                const chosen = kept[Number(at)];
                if (chosen !== undefined) {
                    start(chosen.gesture, {
                        destroys: false,
                        title: `Keep ${found.mac.Name} awake ${chosen.choice.Label}?`,
                        verb: chosen.choice.Label,
                    });
                }
            }}
            options={kept.map(({ choice, gesture }, at) => ({
                busy: gesture.state.kind === 'running',
                id: String(at),
                label: choice.Label,
            }))}
            surface="panel"
            value={picked === -1 ? null : String(picked)}
        />
    );
}

function KeptSheet({ found, onClose, report }: SheetProps & { report: AwakeReport }) {
    const stop = useGesture('awake.off', {
        input: {},
        onDone: onClose,
        rereads: AWAKE_REREADS,
        subject: found.mac.Name,
        target: targetOf(found),
    });
    const actions = (
        <>
            {report.Session === null ? null : (
                <span className={styles.stopKeeping}>
                    <GestureButton
                        gesture={stop}
                        label={AWAKE_GESTURE_WORDS.stop}
                        title={`${AWAKE_GESTURE_WORDS.stop}?`}
                        variant="link"
                    />
                </span>
            )}
            <Button onClick={onClose}>{AWAKE_GESTURE_WORDS.close}</Button>
        </>
    );
    return (
        <Dialog actions={actions} onClose={onClose} title={`Keep ${found.mac.Name} awake`}>
            <AwakeState verdict={report.Verdict} />
            <Keeps choices={report.Choices ?? []} found={found} onClose={onClose}>
                {(kept) => {
                    const gestures = [...kept.map(({ gesture }) => gesture), stop];
                    return (
                        <>
                            <Terms
                                found={found}
                                held={gestures.some(({ state }) => state.kind === 'running')}
                                kept={kept}
                            />
                            <div className={styles.awakeOutcomes}>
                                {gestures.map((gesture) => (
                                    <GestureOutcome gesture={gesture} key={gesture.key} />
                                ))}
                            </div>
                        </>
                    );
                }}
            </Keeps>
        </Dialog>
    );
}

function AwakeSheet({ found, onClose }: SheetProps) {
    const awake = useWatch('awake.read', {}, targetOf(found));
    if (awake.kind === 'read') {
        return <KeptSheet found={found} onClose={onClose} report={awake.out} />;
    }
    return (
        <Dialog
            actions={<Button onClick={onClose}>{AWAKE_GESTURE_WORDS.close}</Button>}
            onClose={onClose}
            title={`Keep ${found.mac.Name} awake`}
        >
            {awake.kind === 'failed' ? (
                <Unanswered error={awake.error} size="panel" />
            ) : (
                <Skeleton what="Awake">
                    <SkeletonRows />
                </Skeleton>
            )}
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

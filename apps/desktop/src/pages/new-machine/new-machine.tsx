import type {
    CallName,
    MachinesCallsNewIn,
    MachinesCallsNewMachine,
    MachinesCreated,
} from '@jterrazz/station-api';
import { useEffect, useRef, useState } from 'react';
import type { ReactElement } from 'react';

import { Button, Dialog, FormField, Screen } from '../../design/index';
import type { Reading as Answered } from '../../engine/entry';
import { createRefusalPlaceOf } from '../../engine/provisional';
import type { RefusalPlace } from '../../engine/provisional';
import { useReread, useWatch } from '../../engine/react';
import { useGesture } from '../../engine/use-gesture';
import type { Gesture } from '../../engine/use-gesture';
import { machineRoute } from '../../shell/route';
import type { Route } from '../../shell/route';
import { GestureButton, GestureOutcome } from '../kit/gestures';
import { Reading, Unanswered } from '../kit/readings';
import {
    askedOf,
    chosenOnOf,
    EMPTY_DRAFT,
    memoryOf,
    nameOf,
    serverInputOf,
    targetOfOn,
    workspaceInputOf,
} from './model';
import type { Draft, Opening } from './model';
import { Lifetimes, Memory, OnRow, Options } from './parts';
import type { Change } from './parts';

import styles from './new-machine.module.css';

const CREATE_REREADS: readonly CallName[] = ['machines.list', 'overview.read'];

const LISTED: readonly CallName[] = ['machines.list'];
export type NewMachineProps = { go: (to: Route) => void; onClose: () => void; opening: Opening };

type SheetProps = NewMachineProps & {
    answer: MachinesCallsNewMachine;
    draft: Draft;
    change: Change;
};

function placeOf(gesture: Gesture<unknown>, answer: MachinesCallsNewMachine): RefusalPlace {
    const macs = (answer.On ?? []).map((on) => on.Mac);
    return gesture.state.kind === 'failed'
        ? createRefusalPlaceOf(gesture.state.error, macs)
        : 'foot';
}

// invariant: Launch hands the owner to the new machine's page once the engine has begun making it (its first
// invariant: progress frame) or answered, whichever comes first, and only once: a refusal comes before either, so
// invariant: it is drawn in the sheet.
function useLaunch({ go, onClose }: Pick<NewMachineProps, 'go' | 'onClose'>) {
    const reread = useReread();
    const launched = useRef(false);
    return (name: string) => {
        if (launched.current || name === '') {
            return;
        }
        launched.current = true;
        onClose();
        go(machineRoute(name));
        reread(LISTED);
    };
}

function useCreate(props: SheetProps) {
    const { answer, draft } = props;
    const on = chosenOnOf(answer);
    const memory = on === undefined ? undefined : memoryOf(answer, on, draft);
    const name = nameOf(answer, draft);
    const launch = useLaunch(props);
    const shared = {
        onDone: (out: MachinesCreated) => {
            launch(out.VM.Name);
        },
        rereads: CREATE_REREADS,
        subject: name,
        target: on === undefined ? undefined : targetOfOn(on),
    };
    const asked = { answer, draft, memory };
    const workspace = useGesture('machines.create-workspace', {
        ...shared,
        input: workspaceInputOf(asked),
    });
    const server = useGesture('machines.create-server', { ...shared, input: serverInputOf(asked) });
    const gesture = draft.kind === 'server' ? server : workspace;
    const begun = gesture.state.kind === 'running' && gesture.state.progress.length > 0;
    useEffect(() => {
        if (begun) {
            launch(name);
        }
    });
    const ready =
        on?.Ready === true &&
        memory !== undefined &&
        (draft.kind === 'workspace' || answer.Server !== null);
    return { gesture, memory, name, on, ready };
}

type ActionsProps = {
    gesture: Gesture<unknown>;
    onClose: () => void;
    ready: boolean;
    shown: string;
};

function Actions({ gesture, onClose, ready, shown }: ActionsProps) {
    return (
        <>
            <Button onClick={onClose}>Cancel</Button>
            {ready ? (
                <GestureButton
                    gesture={gesture}
                    label="Launch"
                    title={`Launch ${shown}?`}
                    variant="primary"
                />
            ) : (
                <Button disabled variant="primary">
                    Launch
                </Button>
            )}
        </>
    );
}

type PurposeProps = { draft: Draft; change: Change; refusal: null | ReactElement };

function Purpose({ draft, change, refusal }: PurposeProps) {
    return (
        <section aria-label="Purpose" className={styles.field}>
            <FormField
                label="What is it for?"
                look="boxed"
                onChange={(purpose) => {
                    change({ purpose });
                }}
                placeholder="Lot L6 · machine file views"
                value={draft.purpose}
            />
            {refusal}
        </section>
    );
}

function Sheet(props: SheetProps) {
    const { answer, draft, change, onClose } = props;
    const [open, setOpen] = useState(false);
    const { gesture, memory, name, on, ready } = useCreate(props);
    const place = placeOf(gesture, answer);
    const outcome = (at: RefusalPlace) =>
        place === at ? <GestureOutcome gesture={gesture} /> : null;
    const file = outcome('file');
    return (
        <Dialog
            actions={
                <Actions
                    gesture={gesture}
                    onClose={onClose}
                    ready={ready}
                    shown={name === '' ? `a ${draft.kind}` : name}
                />
            }
            consequence={`It will follow the machine file ${answer.From}`}
            onClose={onClose}
            title={`New machine on ${answer.Mac}`}
        >
            {file === null ? null : <div className={styles.outcome}>{file}</div>}
            <Purpose change={change} draft={draft} refusal={outcome('purpose')} />
            <FormField
                label="Name"
                look="boxed"
                onChange={(typed) => {
                    change({ name: typed });
                }}
                value={draft.name ?? answer.Name}
            />
            <OnRow answer={answer} change={change} refusal={outcome('on')} />
            {on === undefined ? null : (
                <Memory
                    answer={answer}
                    change={change}
                    memory={memory}
                    name={name}
                    on={on}
                    refusal={outcome('memory')}
                />
            )}
            <Lifetimes answer={answer} change={change} draft={draft} />
            <Options
                answer={answer}
                change={change}
                draft={draft}
                onToggle={() => {
                    setOpen((before) => !before);
                }}
                open={open}
            />
            <div className={styles.outcome}>
                {outcome('foot')}
                {on === undefined ? outcome('memory') : null}
            </div>
        </Dialog>
    );
}

// invariant: a Mac picked asks machines.new again with its On, and the answer drawn stays until the engine's next
// invariant: one comes, so the sheet is never redrawn as reading in between.
function useAnswer(input: MachinesCallsNewIn): Answered<MachinesCallsNewMachine> {
    const reading = useWatch('machines.new', input);
    const [held, setHeld] = useState<MachinesCallsNewMachine | undefined>(undefined);
    if (reading.kind === 'read' && reading.out !== held) {
        setHeld(reading.out);
    }
    if (reading.kind !== 'reading' || held === undefined) {
        return reading;
    }
    return { done: false, kind: 'read', out: held };
}

export function NewMachine(props: NewMachineProps) {
    const { opening, onClose } = props;
    const [draft, setDraft] = useState<Draft>(EMPTY_DRAFT);
    const reading = useAnswer(askedOf(opening, draft));
    const change: Change = (next) => {
        setDraft((before) => ({ ...before, ...next }));
    };
    if (reading.kind === 'read') {
        return (
            <Screen>
                <Sheet {...props} answer={reading.out} change={change} draft={draft} />
            </Screen>
        );
    }
    return (
        <Screen>
            <Dialog
                actions={<Button onClick={onClose}>Cancel</Button>}
                onClose={onClose}
                title="New machine"
            >
                {reading.kind === 'failed' ? (
                    <Unanswered error={reading.error} size="panel" />
                ) : (
                    <Reading size="tab" what="What a new machine can be" />
                )}
            </Dialog>
        </Screen>
    );
}

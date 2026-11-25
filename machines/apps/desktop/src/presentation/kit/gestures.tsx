import {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useMemo,
    useRef,
    useState,
    useSyncExternalStore,
} from 'react';
import type { ReactNode } from 'react';

import type { EngineError } from '../../domain/engine-error';
import { useGestureStore } from '../dependencies/dependencies';
import { useGesture } from '../dependencies/use-gesture';
import type { Gesture } from '../dependencies/use-gesture';
import type { ButtonVariant } from '../ui/button';
import { Button, ConfirmDialog, Panel, ProgressLine, Row, StatePill } from '../ui/index';
import type { RowProps } from '../ui/row';
import { shownOrWithheld } from '../ui/secret';
import { forgetInputOf } from '../words/provisional';
import { ownState } from '../words/shown';
import { Unanswered } from './readings';

import styles from '../ui/page.module.css';

type GestureButtonProps = {
    gesture: Gesture<unknown>;
    label: string;
    title: string;
    verb?: string | undefined;
    consequence?: string | undefined;
    held?: string | undefined;
    variant?: ButtonVariant;
    name?: string;
    destroys?: boolean;
    yields?: boolean;
};

export function marked(gesture: Gesture<unknown>, mark: () => void): Gesture<unknown> {
    return {
        ...gesture,
        start: () => {
            mark();
            gesture.start();
        },
    };
}

type Asking = {
    title: string;
    verb: string;
    consequence?: string | undefined;
    held?: string | undefined;
    destroys: boolean;
};

export function useStart(): (gesture: Gesture<unknown>, asking: Asking) => void {
    const gestures = useGestureStore();
    return (gesture, { title, verb, consequence, held, destroys }) => {
        gestures.askedBy({
            cancel: gesture.cancel,
            confirm: gesture.confirm,
            consequence: (ask) => consequence ?? gesture.consequence(ask),
            destroys,
            held,
            key: gesture.key,
            title,
            verb,
        });
        gesture.start();
    };
}

export function GestureButton({
    gesture,
    label,
    title,
    verb = label,
    consequence,
    held,
    variant = 'plain',
    name,
    destroys = variant === 'danger',
    yields = false,
}: GestureButtonProps) {
    const started = useStart();
    const start = () => {
        started(gesture, { consequence, destroys, held, title, verb });
    };
    return (
        <Button
            busy={gesture.state.kind === 'running'}
            name={name}
            onClick={start}
            variant={variant}
            yields={yields}
        >
            {label}
        </Button>
    );
}

export function GestureConfirm() {
    const gestures = useGestureStore();
    const asked = useSyncExternalStore(gestures.subscribeConfirming, gestures.confirming);
    if (asked === undefined) {
        return null;
    }
    const { ask } = asked;
    return (
        <ConfirmDialog
            destroys={asked.destroys}
            held={asked.held ?? ask.refusal?.wire?.detail?.holder}
            key={asked.key}
            onCancel={asked.cancel}
            onConfirm={asked.confirm}
            said={asked.consequence(ask) ?? ask.refusal?.wire?.message}
            title={asked.title}
            typed={ask.form === 'name' ? ask.name : undefined}
            verb={asked.verb}
        />
    );
}

const UNKNOWN_SAID =
    'No answer came before the connection ended: the engine may have done it. The page below shows what it now says.';

export function Unknown({ error, onDismiss }: { error: EngineError; onDismiss: () => void }) {
    return (
        <span className={styles.outcome}>
            <span className={styles.unanswered}>
                <StatePill {...ownState('unknown')} />
                <span className={styles.said}>
                    {UNKNOWN_SAID}
                    <span className={styles.hint}>{shownOrWithheld(error.message)}</span>
                </span>
            </span>
            <Button onClick={onDismiss} variant="link">
                Dismiss
            </Button>
        </span>
    );
}

function ForgetFix({ name }: { name: string }) {
    const forget = useGesture('machines.forget', {
        input: { Name: name },
        rereads: ['machines.list', 'overview.read'],
        subject: name,
    });
    return (
        <GestureButton
            gesture={forget}
            label={`Forget ${name} first`}
            title={`Forget ${name}?`}
            verb="Forget"
        />
    );
}

function fixOf(error: EngineError): ReactNode {
    const fix = error.wire?.fix;
    const forget = fix ? forgetInputOf(fix) : undefined;
    return forget?.Name === undefined ? undefined : <ForgetFix name={forget.Name} />;
}

export function GestureOutcome({ gesture }: { gesture: Gesture<unknown> }) {
    const { state } = gesture;
    if (state.kind === 'running') {
        const last = state.progress.at(-1);
        if (!last) {
            return null;
        }
        return last.error ? (
            <ProgressLine detail={last.error.message} failed task={last.task} />
        ) : (
            <ProgressLine detail={last.detail} task={last.task} />
        );
    }
    if (state.kind === 'failed') {
        return (
            <span className={styles.outcome}>
                <Unanswered action={fixOf(state.error)} error={state.error} size="panel" />
                <Button onClick={gesture.dismiss} variant="link">
                    Dismiss
                </Button>
            </span>
        );
    }
    if (state.kind === 'unknown') {
        return <Unknown error={state.error} onDismiss={gesture.dismiss} />;
    }
    return state.kind === 'done' ? <StatePill {...ownState('done')} /> : null;
}

export type Held = { title: string; shown: ReactNode; sentence?: string };

type Holding = { hold: (held: Held) => void; release: (title: string) => void };

const Holder = createContext<Holding | undefined>(undefined);

// invariant: "Done here" holds the outcome of a row the engine's answer no longer lists; a row still listed
// invariant: carries its own mark, since the page already shows what changed (no toast for what is on screen).
export function HeldOutcomes({ children }: { children: ReactNode }) {
    const [held, setHeld] = useState<readonly Held[]>([]);
    const hold = useCallback((one: Held) => {
        setHeld((before) => [...before.filter((it) => it.title !== one.title), one]);
    }, []);
    const release = useCallback((title: string) => {
        setHeld((before) =>
            before.some((it) => it.title === title)
                ? before.filter((it) => it.title !== title)
                : before,
        );
    }, []);
    const holding = useMemo(() => ({ hold, release }), [hold, release]);
    return (
        <Holder value={holding}>
            {held.length === 0 ? null : (
                <Panel note={String(held.length)} title="Done here">
                    {held.map((one) => (
                        <Row
                            key={one.title}
                            sentence={one.sentence ?? ''}
                            size="slim"
                            title={one.title}
                            trail={one.shown}
                        />
                    ))}
                </Panel>
            )}
            {children}
        </Holder>
    );
}

export function HoldsWhenDone({ held }: { held: readonly Held[] | undefined }) {
    const holding = useContext(Holder);
    const latest = useRef(held);
    useEffect(() => {
        latest.current = held;
    });
    const done = held !== undefined;
    useEffect(() => {
        if (done && holding !== undefined) {
            for (const one of latest.current ?? []) {
                holding.hold(one);
            }
        }
    }, [done, holding]);
    return null;
}

type GestureRowProps = RowProps & { gesture: Gesture<unknown>; outcome?: ReactNode };

function useHeldWhenGone(holding: Holding | undefined, held: Held, done: boolean): void {
    const latest = useRef({ done, held });
    useEffect(() => {
        latest.current = { done, held };
    });
    const { title } = held;
    useEffect(() => {
        if (done) {
            holding?.release(title);
        }
    }, [done, holding, title]);
    useEffect(
        () => () => {
            if (latest.current.done) {
                holding?.hold(latest.current.held);
            }
        },
        [holding],
    );
}

export function GestureRow({ gesture, outcome, ...row }: GestureRowProps) {
    const holding = useContext(Holder);
    const { kind } = gesture.state;
    const settled = outcome ?? <GestureOutcome gesture={gesture} />;
    const title = row.tag === undefined ? row.title : `${row.title} ${row.tag}`;
    useHeldWhenGone(holding, { shown: settled, title }, kind === 'done');
    const shown = kind === 'idle' || kind === 'asking' || kind === 'done' ? null : settled;
    const trail =
        kind === 'done' ? (
            <>
                {settled}
                {row.trail}
            </>
        ) : (
            row.trail
        );
    return (
        <>
            <Row {...row} trail={trail} />
            {shown === null ? null : <div className={styles.rowOutcome}>{shown}</div>}
        </>
    );
}

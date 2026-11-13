import type { CallName, DiskOutcome, DiskRow, Target } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import type { ButtonVariant } from '../../design/button';
import { Button, Row } from '../../design/index';
import { diskGestureOf } from '../../engine/provisional';
import { useGesture } from '../../engine/use-gesture';
import type { Gesture } from '../../engine/use-gesture';
import { DISK_GESTURE_LABELS, DISK_GESTURE_VERBS } from '../../engine/words';
import { GestureButton, GestureOutcome, GestureRow, HoldsWhenDone } from '../kit/gestures';
import { FreedFigure, heldOf, OutcomePill } from './outcomes';

import pageStyles from '../page.module.css';
import styles from './disk.module.css';

const FREE_REREADS: readonly CallName[] = ['disk.read'];
const MOVE_REREADS: readonly CallName[] = ['disk.read', 'overview.read'];
const ERASE_REREADS: readonly CallName[] = ['disk.read'];

export type DiskGesture = Gesture<DiskOutcome | DiskOutcome[] | null>;

function outcomesOf(gesture: DiskGesture): readonly DiskOutcome[] | undefined {
    const { state } = gesture;
    if (state.kind !== 'done') {
        return undefined;
    }
    const { out } = state;
    return Array.isArray(out) ? out : out === null ? [] : [out];
}

function Settled({ gesture, frees }: { gesture: DiskGesture; frees: boolean }) {
    const outcomes = outcomesOf(gesture);
    if (outcomes === undefined) {
        return <GestureOutcome gesture={gesture} />;
    }
    return (
        <>
            {outcomes.map((outcome) => (
                <span className={pageStyles.outcome} key={outcome.Name}>
                    {frees ? <FreedFigure outcome={outcome} /> : null}
                    <OutcomePill outcome={outcome} />
                </span>
            ))}
        </>
    );
}

type LineProps = {
    row: DiskRow;
    target: Target | undefined;
    lead: ReactNode;
    sentence: string;
    size?: string | undefined;
    chips?: ReactNode;
};

function Trail({ chips, children }: { chips: ReactNode; children?: ReactNode }) {
    return (
        <span className={styles.trail}>
            {chips}
            {children}
        </span>
    );
}

type Worded = { label: string; verb: string; variant: ButtonVariant };

type ShownProps = { line: LineProps; gesture: DiskGesture; worded: Worded; frees: boolean };

function GestureLine({ line, gesture, worded, frees }: ShownProps) {
    const { row, lead, sentence, size, chips } = line;
    const { label, verb, variant } = worded;
    return (
        <GestureRow
            gesture={gesture}
            lead={lead}
            outcome={<Settled frees={frees} gesture={gesture} />}
            sentence={sentence}
            title={row.App}
            trail={
                <Trail chips={chips}>
                    <GestureButton
                        gesture={gesture}
                        label={label}
                        name={`${label} ${row.App}`}
                        title={`${verb} ${row.App}?`}
                        variant={variant}
                        verb={verb}
                    />
                </Trail>
            }
            value={size}
        />
    );
}

function PlainLine({ line, button }: { line: LineProps; button?: ReactNode }) {
    const { row, lead, sentence, size, chips } = line;
    const trail = <Trail chips={chips}>{button}</Trail>;
    return (
        <Row
            lead={lead}
            sentence={sentence}
            title={row.App}
            trail={chips === undefined && button === undefined ? undefined : trail}
            value={size}
        />
    );
}

function FreeLine({ line, worded }: { line: LineProps; worded: Worded }) {
    const free = useGesture('disk.free', {
        input: { Apps: [line.row.App] },
        rereads: FREE_REREADS,
        subject: line.row.App,
        target: line.target,
    });
    return <GestureLine frees gesture={free} line={line} worded={worded} />;
}

function MoveLine({ line, worded, folder }: { line: LineProps; worded: Worded; folder: string }) {
    const move = useGesture('disk.move', {
        input: { Folder: folder },
        rereads: MOVE_REREADS,
        subject: folder,
        target: line.target,
    });
    return <GestureLine frees={false} gesture={move} line={line} worded={worded} />;
}

function EraseLine({ line, worded }: { line: LineProps; worded: Worded }) {
    const erase = useGesture('disk.erase', {
        input: { App: line.row.App },
        rereads: ERASE_REREADS,
        subject: line.row.App,
        target: line.target,
    });
    return <GestureLine frees gesture={erase} line={line} worded={worded} />;
}

export function DiskLine(line: LineProps) {
    const { row } = line;
    const gesture = diskGestureOf(row.Group);
    if (gesture === undefined) {
        return <PlainLine line={line} />;
    }
    const worded: Worded = {
        label: DISK_GESTURE_LABELS[gesture.call],
        variant: gesture.danger ? 'danger' : 'plain',
        verb: DISK_GESTURE_VERBS[gesture.call],
    };
    const [folder] = row.Paths ?? [];
    switch (gesture.call) {
        case 'disk.free': {
            return <FreeLine line={line} worded={worded} />;
        }
        case 'disk.move': {
            return folder === undefined ? (
                <PlainLine button={<Button disabled>{worded.label}</Button>} line={line} />
            ) : (
                <MoveLine folder={folder} line={line} worded={worded} />
            );
        }
        case 'disk.erase': {
            return <EraseLine line={line} worded={worded} />;
        }
    }
}

export function EraseLink({ row, target }: { row: DiskRow; target: Target | undefined }) {
    const erase = useGesture('disk.erase', {
        input: { App: row.App },
        rereads: ERASE_REREADS,
        subject: row.App,
        target,
    });
    const label = DISK_GESTURE_LABELS['disk.erase'];
    const verb = DISK_GESTURE_VERBS['disk.erase'];
    return (
        <span className={pageStyles.outcome}>
            <Settled frees gesture={erase} />
            <GestureButton
                gesture={erase}
                label={label}
                name={`${label} ${row.App}`}
                title={`${verb} ${row.App}?`}
                variant="danger"
                verb={verb}
            />
        </span>
    );
}

export function useFreeAll(target: Target | undefined): DiskGesture {
    return useGesture('disk.free', { input: {}, rereads: FREE_REREADS, subject: '', target });
}

type FreeAllProps = { gesture: DiskGesture; label: string; name: string; variant?: ButtonVariant };

export function FreeAllButton({ gesture, label, name, variant = 'plain' }: FreeAllProps) {
    return (
        <span className={pageStyles.outcome}>
            <GestureOutcome gesture={gesture} />
            <GestureButton
                gesture={gesture}
                label={label}
                title={`${label} on ${name}?`}
                variant={variant}
            />
        </span>
    );
}

export function Freed({ gesture }: { gesture: DiskGesture }) {
    return <HoldsWhenDone held={outcomesOf(gesture)?.map(heldOf)} />;
}

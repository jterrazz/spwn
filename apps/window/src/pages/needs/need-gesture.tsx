import type { CallName, MachinesMacView, OverviewNeed, Target } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import { CopyButton } from '../../design/copy-button';
import { Row, Value } from '../../design/index';
import type { RowProps } from '../../design/row';
import type { In, ReadName } from '../../engine/client';
import { wordOf } from '../../engine/engine-words';
import { moveInputOf, removeInputOf, startInputOf, updateInputOf } from '../../engine/provisional';
import type { NeedCall } from '../../engine/provisional';
import { needState } from '../../engine/shown';
import { useGesture } from '../../engine/use-gesture';
import { sentence } from '../../model/format';
import { findMachine, targetOf } from '../../model/machines';
import { GestureButton, GestureRow } from '../kit/gestures';
import { gestureOf } from './needs';

import styles from './needs.module.css';

const UPDATE_REREADS: readonly CallName[] = ['overview.read', 'apps.list'];
const REMOVE_REREADS: readonly CallName[] = ['overview.read', 'background.list'];
const MOVE_REREADS: readonly CallName[] = ['overview.read', 'disk.read'];
const START_REREADS: readonly CallName[] = ['engine.read', 'machines.list', 'overview.read'];

type Props = {
    need: OverviewNeed;
    macs: readonly MachinesMacView[];
    onOpen: () => void;
};

function rowOf({ need, onOpen }: Props, gesture: ReactNode): RowProps {
    return {
        link: gesture,
        onOpen,
        sentence: sentence([need.Machine, wordOf('CoreSectionID', need.Section).Word]),
        state: needState(need),
        stateAt: 'start',
        title: need.Text,
    };
}

function verbOf(need: OverviewNeed): string {
    return wordOf('OverviewVerb', need.Verb).Word;
}

function needTarget(need: OverviewNeed, macs: readonly MachinesMacView[]): Target | undefined {
    const found = findMachine(macs, need.Machine);
    return found ? targetOf(found) : { name: need.Machine };
}

type FixProps<N extends NeedCall & ReadName> = Props & {
    call: N;
    input: In<N>;
    rereads: readonly CallName[];
};

function Fix<N extends NeedCall & ReadName>(props: FixProps<N>) {
    const { need, macs, call, input, rereads } = props;
    const fix = useGesture(call, {
        input,
        rereads,
        subject: need.Subject,
        target: needTarget(need, macs),
    });
    const verb = verbOf(need);
    return (
        <GestureRow
            gesture={fix}
            {...rowOf(
                props,
                <GestureButton
                    gesture={fix}
                    label={verb}
                    name={`${verb} ${need.Subject}`}
                    title={`${verb} ${need.Subject}?`}
                />,
            )}
        />
    );
}

function Call(props: Props) {
    const { need } = props;
    const update = updateInputOf(need);
    if (update) {
        return <Fix {...props} call="apps.update" input={update} rereads={UPDATE_REREADS} />;
    }
    const remove = removeInputOf(need);
    if (remove) {
        return <Fix {...props} call="background.remove" input={remove} rereads={REMOVE_REREADS} />;
    }
    const move = moveInputOf(need);
    if (move) {
        return <Fix {...props} call="disk.move" input={move} rereads={MOVE_REREADS} />;
    }
    const start = startInputOf(need);
    if (start) {
        return <Fix {...props} call="engine.start" input={start} rereads={START_REREADS} />;
    }
    return <Said {...props} />;
}

function Command({ text }: { text: string }) {
    return (
        <span className={styles.command}>
            <Value>{text}</Value>
            <CopyButton name={`Copy ${text}`} text={text} />
        </span>
    );
}

function Said(props: Props) {
    const gesture = gestureOf(props.need);
    const said =
        gesture !== undefined && 'text' in gesture ? (
            <Command text={gesture.text} />
        ) : (
            <span className={styles.verb}>{verbOf(props.need)}</span>
        );
    return <Row {...rowOf(props, said)} />;
}

export function NeedLine(props: Props) {
    return props.need.Call === '' ? <Said {...props} /> : <Call {...props} />;
}

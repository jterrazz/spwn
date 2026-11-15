import type {
    AccountsProgram,
    CallName,
    MachinesMacView,
    OverviewNeed,
    Target,
} from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactNode } from 'react';

import type { In, ReadName } from '../../../application/ports/engine.port';
import type { ConfirmKind } from '../../../application/use-cases/confirms';
import { sentence } from '../../../domain/format';
import { findMachine, targetOf } from '../../../domain/machines/machines';
import type { Found } from '../../../domain/machines/machines';
import { useGesture } from '../../dependencies/use-gesture';
import { GestureButton, GestureRow } from '../../kit/gestures';
import { LoginSheet } from '../../kit/login-sheet';
import { CopyButton } from '../../ui/copy-button';
import { Button, Row, Value } from '../../ui/index';
import type { RowProps } from '../../ui/row';
import { wordOf } from '../../words/engine-words';
import {
    deleteInputOf,
    loginProgramOf,
    machineKindOf,
    moveInputOf,
    removeInputOf,
    restartInputOf,
    startInputOf,
    updateInputOf,
} from '../../words/provisional';
import type { NeedCall } from '../../words/provisional';
import { needState } from '../../words/shown';
import { gestureOf } from './needs';

import styles from './needs.module.css';

const UPDATE_REREADS: readonly CallName[] = ['overview.read', 'apps.list'];
const REMOVE_REREADS: readonly CallName[] = ['overview.read', 'background.list'];
const MOVE_REREADS: readonly CallName[] = ['overview.read', 'disk.read'];
const START_REREADS: readonly CallName[] = ['engine.read', 'machines.list', 'overview.read'];

const LIFECYCLE_REREADS: readonly CallName[] = ['machines.list', 'overview.read'];

type Props = {
    need: OverviewNeed;
    primary: boolean;
    macs: readonly MachinesMacView[];
    onOpen: () => void;
};

function rowOf({ need, onOpen }: Props, gesture: ReactNode): RowProps {
    return {
        link: gesture,
        onOpen,
        sentence: sentence([need.Machine, need.Noun]),
        state: needState(need),
        stateAt: 'start',
        title: need.Text,
    };
}

function verbOf(need: OverviewNeed): string {
    return wordOf('OverviewVerb', need.Verb).Word;
}

function needTarget(need: OverviewNeed, macs: readonly MachinesMacView[]): Target | undefined {
    const found = findMachine(macs, need.On);
    return found ? targetOf(found) : { name: need.On };
}

type FixProps<N extends NeedCall & ReadName> = Props & {
    call: N;
    input: In<N>;
    rereads: readonly CallName[];
    subject?: string | undefined;
    kind?: ConfirmKind | undefined;
};

function Fix<N extends NeedCall & ReadName>(props: FixProps<N>) {
    const { need, macs, call, input, rereads, primary } = props;
    const subject = props.subject ?? need.Subject;
    const fix = useGesture(call, {
        input,
        kind: props.kind,
        rereads,
        subject,
        target: needTarget(need, macs),
        typed: need.Confirm === '' ? undefined : need.Confirm,
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
                    name={`${verb} ${subject}`}
                    title={`${verb} ${subject}?`}
                    variant={primary ? 'primary' : 'plain'}
                />,
            )}
        />
    );
}

function Lifecycle(props: Props) {
    const { need, macs } = props;
    const found = findMachine(macs, need.Machine);
    const remove = deleteInputOf(need);
    if (found?.vm && remove) {
        return (
            <Fix
                {...props}
                call="machines.delete"
                input={remove}
                rereads={LIFECYCLE_REREADS}
                subject={remove.VM}
            />
        );
    }
    const restart = restartInputOf(need);
    if (found?.vm && restart) {
        return (
            <Fix
                {...props}
                call="machines.restart"
                input={restart}
                kind={machineKindOf(found.vm)}
                rereads={LIFECYCLE_REREADS}
                subject={restart.Name}
            />
        );
    }
    return <Said {...props} />;
}

function Call(props: Props) {
    const { need } = props;
    if (deleteInputOf(need) || restartInputOf(need)) {
        return <Lifecycle {...props} />;
    }
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
    const program = loginProgramOf(need);
    const found = findMachine(props.macs, need.On);
    if (program && found) {
        return <Login {...props} found={found} program={program} />;
    }
    return <Said {...props} />;
}

function Login(props: Props & { found: Found; program: AccountsProgram }) {
    const { found, need, primary, program } = props;
    const [asked, setAsked] = useState(false);
    const verb = verbOf(need);
    return (
        <>
            <Row
                {...rowOf(
                    props,
                    <Button
                        name={`${verb} ${program}`}
                        onClick={() => {
                            setAsked(true);
                        }}
                        variant={primary ? 'primary' : 'plain'}
                    >
                        {verb}
                    </Button>,
                )}
            />
            {asked ? (
                <LoginSheet
                    found={found}
                    onClose={() => {
                        setAsked(false);
                    }}
                    program={program}
                    title={`${verb} ${program}`}
                />
            ) : null}
        </>
    );
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

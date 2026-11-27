import type {
    AccountsProgram,
    CallName,
    MachinesMacView,
    MachinesStartInput,
    OverviewNeed,
    Target,
} from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactNode } from 'react';

import type { In, ReadName } from '../../../application/ports/engine.port';
import type { ConfirmForm, ConfirmKind } from '../../../application/use-cases/confirms';
import { sentence } from '../../../domain/format';
import { findMachine, targetOf } from '../../../domain/machines/machines';
import type { Found } from '../../../domain/machines/machines';
import { useGesture } from '../../dependencies/use-gesture';
import { GestureButton, GestureRow } from '../../kit/gestures';
import { LoginSheet } from '../../kit/login-sheet';
import { Openable } from '../../kit/openable';
import { OpenButton } from '../../kit/opens';
import { carriesCommand } from '../../ui/command';
import { CopyButton } from '../../ui/copy-button';
import { Button, Row } from '../../ui/index';
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
    startVmInputOf,
    updateInputOf,
    upgradeInputOf,
    wakeInputOf,
} from '../../words/provisional';
import type { NeedCall } from '../../words/provisional';
import { needState } from '../../words/shown';

import styles from './needs.module.css';

const UPDATE_REREADS: readonly CallName[] = ['overview.read', 'apps.list'];
const REMOVE_REREADS: readonly CallName[] = ['overview.read', 'background.list'];
const MOVE_REREADS: readonly CallName[] = ['overview.read', 'disk.read'];
const START_REREADS: readonly CallName[] = ['engine.read', 'overview.read'];
const UPGRADE_REREADS: readonly CallName[] = ['overview.read', 'machinefile.drift'];
const LIFECYCLE_REREADS: readonly CallName[] = ['overview.read'];

type Props = {
    need: OverviewNeed;
    primary: boolean;
    macs: readonly MachinesMacView[];
    onOpen: () => void;
    named?: boolean;
};

function rowOf({ need, onOpen, named = false }: Props, gesture: ReactNode): RowProps {
    return {
        link: gesture,
        onOpen,
        sentence: sentence([need.Machine, need.Noun]),
        state: named ? { ...needState(need), said: 'name' } : needState(need),
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
    asks?: '' | ConfirmForm | undefined;
};

function Fix<N extends NeedCall & ReadName>(props: FixProps<N>) {
    const { need, macs, call, input, rereads, primary } = props;
    const subject = props.subject ?? need.Subject;
    const fix = useGesture(call, {
        asks: props.asks,
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
    const restart = restartInputOf(need);
    if (found?.vm && restart) {
        return (
            <Fix
                {...props}
                asks={found.vm.Asks?.Restart}
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

function Retry(props: Props & { input: MachinesStartInput }) {
    const { need, macs, input } = props;
    const found = findMachine(macs, need.Machine);
    if (found?.vm) {
        return (
            <Fix
                {...props}
                call="machines.start"
                input={input}
                kind={machineKindOf(found.vm)}
                rereads={LIFECYCLE_REREADS}
                subject={input.VM}
            />
        );
    }
    return <Said {...props} />;
}

function deleteAsksOf({ macs, need }: Props): '' | ConfirmForm | undefined {
    return findMachine(macs, need.Machine)?.vm?.Asks?.Delete;
}

function Call(props: Props) {
    const { need } = props;
    const retry = startVmInputOf(need);
    if (retry) {
        return <Retry {...props} input={retry} />;
    }
    if (restartInputOf(need)) {
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
    const erase = deleteInputOf(need);
    if (erase) {
        return (
            <Fix
                {...props}
                asks={deleteAsksOf(props)}
                call="machines.delete"
                input={erase}
                rereads={LIFECYCLE_REREADS}
                subject={erase.VM}
            />
        );
    }
    const upgrade = upgradeInputOf(need);
    if (upgrade) {
        return (
            <Fix {...props} call="machinefile.upgrade" input={upgrade} rereads={UPGRADE_REREADS} />
        );
    }
    const wake = wakeInputOf(need);
    if (wake) {
        return <Fix {...props} call="machines.wake" input={wake} rereads={LIFECYCLE_REREADS} />;
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

function Opened(props: Props & { label: string; openable: Openable }) {
    const { label, openable, primary } = props;
    const variant = primary ? 'primary' : 'plain';
    return (
        <Row
            {...rowOf(
                props,
                <OpenButton held="" label={label} openable={openable} variant={variant} />,
            )}
        />
    );
}

function Said(props: Props) {
    const { need } = props;
    const { Command, Path } = need;
    const onThisMac = findMachine(props.macs, need.Machine)?.mac.ThisMac === true;
    const pane = onThisMac ? Openable.ofPane(need) : undefined;
    if (pane) {
        return <Opened {...props} label={`${verbOf(need)} ${need.Object}`} openable={pane} />;
    }
    const line = Openable.ofFixLine(need);
    if (line) {
        return <Opened {...props} label={verbOf(need)} openable={line} />;
    }
    if (Command !== '') {
        return (
            <Row
                {...rowOf(
                    props,
                    <CopyButton label="Copy the command" name="Copy the command" text={Command} />,
                )}
            />
        );
    }
    return (
        <Row
            {...rowOf(props, null)}
            below={
                Path === '' || carriesCommand(Path) ? null : (
                    <span className={styles.path}>{Path}</span>
                )
            }
        />
    );
}

export function NeedLine(props: Props) {
    return props.need.Call === '' ? <Said {...props} /> : <Call {...props} />;
}

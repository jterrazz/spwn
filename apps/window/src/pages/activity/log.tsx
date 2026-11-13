import type { MachinesLogInput, MachinesLogOutput, Target } from '@jterrazz/station-api';
import { useCallback, useEffect, useState } from 'react';

import { Button, Missing, Panel, SearchField, Switch } from '../../design/index';
import type { EngineError } from '../../engine/engine-error';
import { useRead } from '../../engine/react';
import { clock } from '../../model/format';
import { targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';
import { NotGiven, Unanswered } from '../kit/readings';
import { LineList } from './line-list';

import styles from './log.module.css';

type Source = 'console' | 'setup';

type Heard =
    | { kind: 'failed'; error: EngineError }
    | { kind: 'read'; out: MachinesLogOutput; at: Date };

const FOOT = 'a reading, not a live feed: press Refresh for the newest lines';

type ReaderProps = {
    input: MachinesLogInput;
    target: Target | undefined;
    heard: (heard: Heard) => void;
};

function Reader({ input, target, heard }: ReaderProps) {
    const reading = useRead('machines.log', input, target);
    useEffect(() => {
        if (reading.kind === 'read') {
            heard({ at: new Date(), kind: 'read', out: reading.out });
        } else if (reading.kind === 'failed') {
            heard(reading);
        }
    }, [reading, heard]);
    return null;
}

function SourceSwitch({ value, onChange }: { value: Source; onChange: (next: Source) => void }) {
    return (
        <Switch
            label="Which log"
            onChange={onChange}
            options={[
                { id: 'console', label: 'Console' },
                { id: 'setup', label: 'Setup run' },
            ]}
            surface="panel"
            value={value}
        />
    );
}

type ConsoleProps = {
    vm: string;
    lines: readonly string[];
    at: Date;
    onRefresh: () => void;
};

function Console({ vm, lines, at, onRefresh }: ConsoleProps) {
    const [query, setQuery] = useState('');
    return (
        <>
            <div className={styles.tools}>
                <SearchField
                    label="Filter the lines"
                    onChange={setQuery}
                    placeholder="Filter the lines"
                    value={query}
                />
                <Button onClick={onRefresh}>Refresh</Button>
                <Button
                    onClick={() => {
                        void navigator.clipboard.writeText(lines.join('\n'));
                    }}
                >
                    Copy all
                </Button>
            </div>
            <LineList
                foot={`read at ${clock(at) ?? ''} · ${FOOT}`}
                lines={lines}
                memory={vm}
                query={query}
            />
        </>
    );
}

function useLog(vm: string, target: Target | undefined) {
    const [round, setRound] = useState(0);
    const input: MachinesLogInput = { VM: vm };
    const [last, setLast] = useState<undefined | { vm: string; heard: Heard }>(undefined);
    const heard = useCallback(
        (next: Heard) => {
            setLast({ heard: next, vm });
        },
        [vm],
    );
    const refresh = useCallback(() => {
        setRound((value) => value + 1);
    }, []);
    return {
        reader: <Reader heard={heard} input={input} key={round} target={target} />,
        refresh,
        shown: last?.vm === vm ? last.heard : undefined,
    };
}

function ConsoleBody({ vm, target }: { vm: string; target: Target | undefined }) {
    const { reader, refresh, shown } = useLog(vm, target);
    let body;
    if (shown?.kind === 'failed') {
        body = <Unanswered error={shown.error} size="panel" />;
    } else if (shown) {
        body = <Console at={shown.at} lines={shown.out.Lines ?? []} onRefresh={refresh} vm={vm} />;
    } else {
        body = <p className={styles.text}>Reading its console…</p>;
    }
    return (
        <>
            {reader}
            {body}
        </>
    );
}

function VmLog({ vm, target }: { vm: string; target: Target | undefined }) {
    const [source, setSource] = useState<Source>('console');
    const action = <SourceSwitch onChange={setSource} value={source} />;
    if (source === 'setup') {
        return (
            <Panel
                action={action}
                note="what the file installed and wrote"
                title="Its last setup run"
            >
                <Missing>What its file installed and wrote on its last run.</Missing>
            </Panel>
        );
    }
    return (
        <Panel action={action} note="its console, then its hypervisor" title={`What ${vm} prints`}>
            <ConsoleBody target={target} vm={vm} />
        </Panel>
    );
}

export function Log({ found }: { found: Found }) {
    if (!found.vm) {
        return (
            <NotGiven
                title="What station did on this Mac"
                what="The engine's own lines and its setup runs."
            />
        );
    }
    return <VmLog key={found.vm.Name} target={targetOf({ mac: found.mac })} vm={found.vm.Name} />;
}

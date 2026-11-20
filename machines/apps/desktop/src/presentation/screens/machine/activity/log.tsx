import type {
    EngineLog,
    EngineLogInput,
    MachinesLogInput,
    MachinesLogOutput,
    Target,
} from '@jterrazz/station-api';
import { useCallback, useEffect, useState } from 'react';

import type { EngineError } from '../../../../domain/engine-error';
import { clock } from '../../../../domain/format';
import { targetOf } from '../../../../domain/machines/machines';
import type { Found } from '../../../../domain/machines/machines';
import { SEPARATOR } from '../../../../domain/separator';
import { useRead } from '../../../dependencies/dependencies';
import { Unanswered } from '../../../kit/readings';
import { Button, Missing, Panel, SearchField, Switch } from '../../../ui/index';
import { LineList } from './line-list';

import styles from './log.module.css';

type Source = 'console' | 'setup';

type Heard =
    | { kind: 'failed'; error: EngineError }
    | { kind: 'read'; out: EngineLog | MachinesLogOutput; at: Date };

const FOOT = 'a reading, not a live feed: press Refresh for the newest lines';

type Asked =
    | { call: 'engine.log'; input: EngineLogInput }
    | { call: 'machines.log'; input: MachinesLogInput };

type ReaderProps = Asked & {
    target: Target | undefined;
    heard: (heard: Heard) => void;
};

function Reader({ call, input, target, heard }: ReaderProps) {
    const reading = useRead(call, input, target);
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
    memory: string;
    lines: readonly string[];
    none: string | undefined;
    at: Date;
    onRefresh: () => void;
};

function Console({ memory, lines, none, at, onRefresh }: ConsoleProps) {
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
                foot={`read at ${clock(at) ?? ''}${SEPARATOR}${FOOT}`}
                lines={lines}
                memory={memory}
                none={none}
                query={query}
            />
        </>
    );
}

function useLog(of: string) {
    const [round, setRound] = useState(0);
    const [last, setLast] = useState<undefined | { of: string; heard: Heard }>(undefined);
    const heard = useCallback(
        (next: Heard) => {
            setLast({ heard: next, of });
        },
        [of],
    );
    const refresh = useCallback(() => {
        setRound((value) => value + 1);
    }, []);
    return { heard, refresh, round, shown: last?.of === of ? last.heard : undefined };
}

type LinesProps = {
    memory: string;
    reading: string;
    shown: Heard | undefined;
    onRefresh: () => void;
};

function Lines({ memory, reading, shown, onRefresh }: LinesProps) {
    if (shown?.kind === 'failed') {
        return <Unanswered error={shown.error} size="panel" />;
    }
    if (shown) {
        const none = 'Empty' in shown.out ? shown.out.Empty : undefined;
        return (
            <Console
                at={shown.at}
                lines={shown.out.Lines ?? []}
                memory={memory}
                none={none}
                onRefresh={onRefresh}
            />
        );
    }
    return <p className={styles.text}>{reading}</p>;
}

function ConsoleBody({ vm, target }: { vm: string; target: Target | undefined }) {
    const { heard, refresh, round, shown } = useLog(vm);
    return (
        <>
            <Reader
                call="machines.log"
                heard={heard}
                input={{ VM: vm }}
                key={round}
                target={target}
            />
            <Lines memory={vm} onRefresh={refresh} reading="Reading its console…" shown={shown} />
        </>
    );
}

function MacLog({ mac, target }: { mac: string; target: Target | undefined }) {
    const { heard, refresh, round, shown } = useLog(mac);
    return (
        <Panel title="What Machines did on this Mac">
            <Reader call="engine.log" heard={heard} input={{}} key={round} target={target} />
            <Lines memory={mac} onRefresh={refresh} reading="Reading its lines…" shown={shown} />
        </Panel>
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
    const target = targetOf({ mac: found.mac });
    if (!found.vm) {
        return <MacLog key={found.mac.Name} mac={found.mac.Name} target={target} />;
    }
    return <VmLog key={found.vm.Name} target={target} vm={found.vm.Name} />;
}

import type { MachinesLogInput, MachinesLogOutput, Target } from '@jterrazz/station-api';
import { useCallback, useEffect, useState } from 'react';

import { Button, Missing, Panel, SearchField, Switch, Verdict } from '../../design/index';
import type { EngineError } from '../../engine/engine-error';
import { useRead } from '../../engine/react';
import { ownState } from '../../engine/shown';
import { clock } from '../../model/format';
import { targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';
import { NotGiven, Reading, Unanswered } from '../kit/readings';
import { Page } from '../page';
import { LineList } from './line-list';
import { countOf } from './lines';

import styles from './logs.module.css';

type Source = 'console' | 'setup';

type Heard =
    | { kind: 'failed'; error: EngineError }
    | { kind: 'read'; out: MachinesLogOutput; at: Date };

const FEED = 'A reading, not a live feed · Refresh reads the newest lines';

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

function SetupRun({ source, onSource }: { source: Source; onSource: (next: Source) => void }) {
    return (
        <>
            <Verdict
                line="Its console and its hypervisor are read, not yet its setup run"
                size="tab"
                title="No setup run from the engine."
                {...ownState('not-given')}
            />
            <Panel
                action={<SourceSwitch onChange={onSource} value={source} />}
                note="what the file installed and wrote"
                title="Its last setup run"
            >
                <Missing>What its file installed and wrote on its last run.</Missing>
            </Panel>
        </>
    );
}

type ConsoleProps = {
    vm: string;
    lines: readonly string[];
    at: Date;
    source: Source;
    onSource: (next: Source) => void;
    onRefresh: () => void;
};

function Console({ vm, lines, at, source, onSource, onRefresh }: ConsoleProps) {
    const [query, setQuery] = useState('');
    return (
        <>
            <Verdict
                line={FEED}
                size="tab"
                title={`${countOf(lines.length, lines.length, '')}, read at ${clock(at) ?? ''}.`}
                {...ownState('read-once')}
            />
            <Panel
                action={<SourceSwitch onChange={onSource} value={source} />}
                note="its console, then its hypervisor"
                title={`What ${vm} prints`}
            >
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
            </Panel>
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

function VmLogs({ vm, target }: { vm: string; target: Target | undefined }) {
    const [source, setSource] = useState<Source>('console');
    const { reader, refresh, shown } = useLog(vm, target);
    let body;
    if (source === 'setup') {
        body = <SetupRun onSource={setSource} source={source} />;
    } else if (shown?.kind === 'failed') {
        body = <Unanswered error={shown.error} size="tab" />;
    } else if (shown) {
        body = (
            <Console
                at={shown.at}
                lines={shown.out.Lines ?? []}
                onRefresh={refresh}
                onSource={setSource}
                source={source}
                vm={vm}
            />
        );
    } else {
        body = <Reading size="tab" what={`the log of ${vm}`} />;
    }
    return (
        <Page>
            {reader}
            {body}
        </Page>
    );
}

function MacLogs({ found }: { found: Found }) {
    return (
        <Page>
            <Verdict
                line={`The engine reads the log of a VM, not yet the log of ${found.mac.Name}`}
                size="tab"
                title="No log of this Mac from the engine."
                {...ownState('not-given')}
            />
            <NotGiven
                title="What station did on this Mac"
                what="The engine's own lines and its setup runs."
            />
        </Page>
    );
}

export function LogsTab({ found }: { found: Found }) {
    if (!found.vm) {
        return <MacLogs found={found} />;
    }
    return <VmLogs target={targetOf({ mac: found.mac })} vm={found.vm.Name} />;
}

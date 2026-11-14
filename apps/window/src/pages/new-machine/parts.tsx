import type { ReactElement, ReactNode } from 'react';

import { capitalised, identityOf, MemoryPick, Missing, Picks, Switch } from '../../design/index';
import type { MemoryHolder, Pick as Option } from '../../design/index';
import { wordOf } from '../../engine/engine-words';
import type { MachinesCallsNewMachine, MachinesCallsNewOn } from '../../engine/machines-new';
import { FIT_ALLOWS } from '../../engine/provisional';
import { useWatch } from '../../engine/react';
import { machineKindWord } from '../../engine/shown';
import { FIT_WORDS, PRESET_WORDS, ROOM_FOR_WORDS } from '../../engine/words';
import { bytes } from '../../model/format';
import { machineNamesOf, macsOf } from '../../model/machines';
import { useIdentityOf } from '../kit/identity';
import { GB, gbOf } from './model';
import type { Draft, Kind } from './model';

import styles from './new-machine.module.css';

export type Change = (next: Partial<Draft>) => void;

type FieldProps = { label: string; children: ReactNode; after?: ReactNode };

export function Field({ label, children, after }: FieldProps) {
    return (
        <section aria-label={label} className={styles.field}>
            <span className={styles.label}>{label}</span>
            {children}
            {after}
        </section>
    );
}

function onPick(on: MachinesCallsNewOn): Option<string> {
    const free = on.Room === null ? [] : [`${String(gbOf(on.Room.Capacity.Free))} GB free`];
    return {
        disabled: !on.Ready,
        id: on.Mac,
        lines: on.Ready ? free : [on.Verdict.Sentence],
        title: on.Mac,
    };
}

type OnRowProps = { answer: MachinesCallsNewMachine; change: Change; refusal: null | ReactElement };

export function OnRow({ answer, change, refusal }: OnRowProps) {
    const offered = answer.On ?? [];
    if (offered.length < 2) {
        return refusal;
    }
    return (
        <Field after={refusal} label="On">
            <Picks
                label="On"
                look="cards"
                onChange={(on) => {
                    change({ memory: null, on });
                }}
                options={offered.map(onPick)}
                value={answer.Mac}
            />
        </Field>
    );
}

function useNewIdentity(name: string) {
    const listing = useWatch('machines.list', {});
    const listed = machineNamesOf(macsOf(listing.kind === 'read' ? listing.out : undefined));
    return identityOf(name, [...listed, name]);
}

type MemoryProps = {
    answer: MachinesCallsNewMachine;
    on: MachinesCallsNewOn;
    memory: number | undefined;
    name: string;
    change: Change;
};

function Presets({ on, memory, change }: Pick<MemoryProps, 'change' | 'memory' | 'on'>) {
    const presets = on.Room?.Presets ?? [];
    return (
        <Picks
            label="Memory presets"
            look="chips"
            onChange={(id) => {
                const preset = presets.find((one) => one.Preset === id);
                if (preset !== undefined) {
                    change({ memory: preset.MemoryBytes });
                }
            }}
            options={presets.map((preset) => ({
                disabled: !FIT_ALLOWS[preset.Fit],
                id: preset.Preset,
                title: `${String(gbOf(preset.MemoryBytes))} GB · ${
                    FIT_ALLOWS[preset.Fit] ? PRESET_WORDS[preset.Preset] : FIT_WORDS[preset.Fit]
                }`,
            }))}
            value={presets.find((preset) => preset.MemoryBytes === memory)?.Preset ?? null}
        />
    );
}

function Range({ answer, on, memory, change }: Omit<MemoryProps, 'name'>) {
    if (memory === undefined) {
        return null;
    }
    return (
        <input
            aria-label="Memory, GB"
            className={styles.range}
            max={gbOf(on.MaxMemory)}
            min={gbOf(answer.MinMemory)}
            onChange={(event) => {
                change({ memory: Number(event.target.value) * GB });
            }}
            step={1}
            type="range"
            value={gbOf(memory)}
        />
    );
}

export function Memory(props: MemoryProps) {
    const { on, memory, name } = props;
    const identityOfListed = useIdentityOf();
    const identity = useNewIdentity(name);
    if (on.Room === null) {
        return (
            <Field label="Memory">
                <Missing>machines.new measures no memory on this Mac.</Missing>
            </Field>
        );
    }
    const { Capacity } = on.Room;
    const holders: MemoryHolder[] = [
        { identity: 'system', key: 'macos', label: 'macOS', squares: gbOf(Capacity.MacOS) },
        ...(Capacity.VMs ?? []).map((vm) => ({
            identity: identityOfListed(vm.Name),
            key: vm.Name,
            label: vm.Name,
            squares: gbOf(vm.Bytes),
        })),
    ];
    const chosen = memory === undefined ? 0 : gbOf(memory);
    return (
        <MemoryPick
            chosen={chosen}
            free={Math.max(0, gbOf(Capacity.Free) - chosen)}
            holders={holders}
            identity={identity}
            line={ROOM_FOR_WORDS[on.Room.RoomFor]}
            name={on.Mac}
            title={memory === undefined ? 'Memory' : `Memory · ${String(chosen)} GB`}
        >
            <Range {...props} />
            <Presets {...props} />
        </MemoryPick>
    );
}

type LifetimesProps = { answer: MachinesCallsNewMachine; draft: Draft; change: Change };

export function Lifetimes({ answer, draft, change }: LifetimesProps) {
    if (draft.kind === 'server') {
        return null;
    }
    return (
        <Field label="Lifetime">
            <Picks
                label="Lifetime"
                look="cards"
                onChange={(lifetime) => {
                    change({ lifetime });
                }}
                options={(answer.Lifetimes ?? []).map((id) => {
                    const word = wordOf('MachinesLifetime', id);
                    return {
                        id,
                        lines: word.Detail === undefined ? [] : [word.Detail],
                        title: word.Word,
                    };
                })}
                value={draft.lifetime ?? answer.Lifetime}
            />
        </Field>
    );
}

type OptionsProps = LifetimesProps & { open: boolean; onToggle: () => void };

function summaryOf(answer: MachinesCallsNewMachine, kind: Kind): string {
    const size = kind === 'server' && answer.Server !== null ? answer.Server.Size : answer.Size;
    const from = kind === 'server' && answer.Server !== null ? answer.Server.Image : answer.From;
    return [machineKindWord(kind), from, `${String(size.CPUs)} CPU`, bytes(size.DiskBytes)].join(
        ' · ',
    );
}

export function Options({ answer, draft, change, open, onToggle }: OptionsProps) {
    const kinds: readonly { id: Kind; label: string }[] = [
        { id: 'workspace', label: capitalised(machineKindWord('workspace')) },
        ...(answer.Server === null
            ? []
            : [{ id: 'server' as const, label: capitalised(machineKindWord('server')) }]),
    ];
    return (
        <div className={styles.options}>
            <div className={styles.fold}>
                <button
                    aria-expanded={open}
                    className={styles.toggle}
                    onClick={onToggle}
                    type="button"
                >
                    <span aria-hidden="true">{open ? '⌄' : '›'}</span> Options
                </button>
                <span className={styles.summary}>{summaryOf(answer, draft.kind)}</span>
            </div>
            {open ? (
                <Switch
                    label="Kind"
                    onChange={(kind: Kind) => {
                        change({ kind, memory: null });
                    }}
                    options={kinds}
                    surface="panel"
                    value={draft.kind}
                />
            ) : null}
        </div>
    );
}

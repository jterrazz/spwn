import { useId } from 'react';

import type { Tone } from '../../domain/tone';
import { Button } from './button';
import type { Identity } from './identity';
import { MemoryTrack } from './memory-track';
import type { TrackSegment } from './memory-track';
import { isCalm } from './state-pill';
import { StateSign } from './state-sign';

import styles from './fleet-register.module.css';

export type FleetState = { word: string; tone: Tone };

// value: where a share sits in its Mac's memory, as shares of the Mac's total; `used` is a share of the share.
export type FleetShare = { at: number; width: number; used?: number | undefined };

export type FleetMachine = {
    name: string;
    identity?: Identity | undefined;
    purpose: string;
    state: FleetState;
    cpu: string;
    memory: string;
    share?: FleetShare | undefined;
    sentence: string;
    line?: string | undefined;
};

// value: `system` is macOS's share of the Mac, and its presence says the Mac's memory was measured: a Mac with
// value: none draws no track, on its line nor on its machines'.
export type FleetMac = {
    name: string;
    thisMac: boolean;
    kind: string;
    state: FleetState;
    cpu: string;
    free: string;
    system?: FleetShare | undefined;
    machines: readonly FleetMachine[];
};

type Opening = {
    opened: null | string;
    onToggle: (name: string) => void;
    onOpen: (name: string) => void;
};

export function Glow() {
    return <i aria-hidden="true" className={styles.glow} />;
}

function segmentOf(machine: FleetMachine): TrackSegment[] {
    return machine.share === undefined
        ? []
        : [{ ...machine.share, identity: machine.identity, key: machine.name }];
}

function segmentsOf(mac: FleetMac, system: FleetShare): TrackSegment[] {
    return [{ ...system, identity: 'system', key: 'macos' }, ...mac.machines.flatMap(segmentOf)];
}

function Track({
    name,
    segments,
}: {
    name: string;
    segments: readonly TrackSegment[] | undefined;
}) {
    if (segments === undefined) {
        return <span />;
    }
    return <MemoryTrack label={`Memory of ${name}`} segments={segments} />;
}

function MacLine({ mac, onOpenMac }: { mac: FleetMac; onOpenMac: (name: string) => void }) {
    return (
        <div className={`${styles.line} ${styles.mac}`}>
            <span className={styles.name}>
                {mac.thisMac ? <Glow /> : null}
                <button
                    className={styles.macName}
                    onClick={() => {
                        onOpenMac(mac.name);
                    }}
                    type="button"
                >
                    {mac.name}
                </button>
            </span>
            <span />
            <span className={`${styles.state} ${styles[mac.state.tone]}`}>
                {isCalm(mac.state.tone) ? null : mac.state.word}
            </span>
            <span className={styles.figure}>{mac.cpu}</span>
            <Track
                name={mac.name}
                segments={mac.system === undefined ? undefined : segmentsOf(mac, mac.system)}
            />
            <span className={styles.figure}>{mac.free}</span>
        </div>
    );
}

type MachineLineProps = Opening & { machine: FleetMachine; measured: boolean };

function Detail({
    id,
    machine,
    onOpen,
}: Pick<Opening, 'onOpen'> & { id: string; machine: FleetMachine }) {
    return (
        <div className={styles.detail} id={id}>
            <span className={styles.said}>
                <span>{machine.sentence}</span>
                {machine.line === undefined || machine.line === '' ? null : (
                    <span className={styles.saidLine}>{machine.line}</span>
                )}
            </span>
            <Button
                onClick={() => {
                    onOpen(machine.name);
                }}
            >
                {`Open ${machine.name}`}
            </Button>
        </div>
    );
}

function MachineLine({ machine, measured, opened, onToggle, onOpen }: MachineLineProps) {
    const id = `${useId()}-detail`;
    const open = opened === machine.name;
    return (
        <>
            <div className={`${styles.line} ${styles.machine}`} data-open={open || undefined}>
                <span className={styles.name}>
                    <i className={styles.swatch} data-identity={machine.identity} />
                    <button
                        aria-controls={id}
                        aria-expanded={open}
                        className={styles.opener}
                        onClick={() => {
                            onToggle(machine.name);
                        }}
                        type="button"
                    >
                        {machine.name}
                    </button>
                </span>
                <span className={styles.what}>{machine.purpose}</span>
                <span className={`${styles.state} ${styles[machine.state.tone]}`}>
                    <StateSign tone={machine.state.tone} />
                    {machine.state.word}
                </span>
                <span className={styles.figure}>{machine.cpu}</span>
                <Track name={machine.name} segments={measured ? segmentOf(machine) : undefined} />
                <span className={styles.figure}>{machine.memory}</span>
            </div>
            {open ? <Detail id={id} machine={machine} onOpen={onOpen} /> : null}
        </>
    );
}

export type FleetRegisterProps = Opening & {
    macs: readonly FleetMac[];
    onOpenMac: (name: string) => void;
};

export function FleetRegister({ macs, onOpenMac, ...opening }: FleetRegisterProps) {
    return (
        <div className={styles.register}>
            {macs.map((mac) => (
                <div className={styles.group} key={mac.name}>
                    <MacLine mac={mac} onOpenMac={onOpenMac} />
                    {mac.machines.map((machine) => (
                        <MachineLine
                            key={machine.name}
                            machine={machine}
                            measured={mac.system !== undefined}
                            {...opening}
                        />
                    ))}
                </div>
            ))}
        </div>
    );
}

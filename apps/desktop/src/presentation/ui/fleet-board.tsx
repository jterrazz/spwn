import { useState } from 'react';
import type { ReactNode } from 'react';

import { SEPARATOR } from '../../domain/separator';
import { FleetRegister, Glow } from './fleet-register';
import type { FleetMac } from './fleet-register';
import { Panel } from './panel';
import { StateSign } from './state-sign';

import styles from './fleet-board.module.css';

export type { FleetMac, FleetMachine, FleetShare, FleetState } from './fleet-register';

export type FleetBoardProps = {
    macs: readonly FleetMac[];
    summary: string;
    thisMac: string;
    onOpen: (name: string) => void;
    onOpenMac: (name: string) => void;
    foot?: ReactNode;
    chosenAtFirst?: string | undefined;
    openedAtFirst?: string | undefined;
};

const CARD_LOOKS = {
    bad: styles.ringBad,
    new: '',
    ok: '',
    plain: styles.quiet,
    warn: styles.ringWarn,
};

function MacCard({
    mac,
    chosen,
    onChoose,
}: {
    mac: FleetMac;
    chosen: boolean;
    onChoose: () => void;
}) {
    return (
        <button
            aria-label={[mac.name, mac.kind, mac.state.word].join(SEPARATOR)}
            aria-pressed={chosen}
            className={`${styles.card} ${CARD_LOOKS[mac.state.tone]}`}
            onClick={onChoose}
            type="button"
        >
            <b className={styles.cardName}>
                <span className={styles.cut}>{mac.name}</span>
                {mac.thisMac ? <Glow /> : null}
            </b>
            <small className={styles.kind}>{mac.kind}</small>
            <span className={`${styles.said} ${styles[mac.state.tone]}`}>
                <StateSign tone={mac.state.tone} />
                {mac.state.word}
            </span>
        </button>
    );
}

type StageProps = {
    macs: readonly FleetMac[];
    chosen: null | string;
    onChoose: (name: string) => void;
    summary: string;
    thisMac: string;
};

function Stage({ macs, chosen, onChoose, summary, thisMac }: StageProps) {
    return (
        <div aria-label="Macs" className={styles.stage} role="group">
            <div className={styles.deck}>
                {macs.map((mac) => (
                    <MacCard
                        chosen={chosen === mac.name}
                        key={mac.name}
                        mac={mac}
                        onChoose={() => {
                            onChoose(mac.name);
                        }}
                    />
                ))}
            </div>
            <div className={styles.foot}>
                <span>{summary}</span>
                {macs.some((mac) => mac.thisMac) ? (
                    <span className={styles.here}>
                        <Glow />
                        {thisMac}
                    </span>
                ) : null}
            </div>
        </div>
    );
}

function toggled(current: null | string, name: string): null | string {
    return current === name ? null : name;
}

export function FleetBoard({
    macs,
    summary,
    thisMac,
    onOpen,
    onOpenMac,
    foot,
    chosenAtFirst,
    openedAtFirst,
}: FleetBoardProps) {
    const [chosen, setChosen] = useState<null | string>(chosenAtFirst ?? null);
    const [opened, setOpened] = useState<null | string>(openedAtFirst ?? null);
    const shown = chosen === null ? macs : macs.filter((mac) => mac.name === chosen);
    const count = macs.reduce((sum, mac) => sum + mac.machines.length, 0);
    return (
        <Panel note={String(count)} padded title="Machines">
            <div className={styles.board}>
                <Stage
                    chosen={chosen}
                    macs={macs}
                    onChoose={(name) => {
                        setChosen(toggled(chosen, name));
                        setOpened(null);
                    }}
                    summary={summary}
                    thisMac={thisMac}
                />
                <FleetRegister
                    macs={shown}
                    onOpen={onOpen}
                    onOpenMac={onOpenMac}
                    onToggle={(name) => {
                        setOpened(toggled(opened, name));
                    }}
                    opened={opened}
                />
                {foot === undefined ? null : <div className={styles.end}>{foot}</div>}
            </div>
        </Panel>
    );
}

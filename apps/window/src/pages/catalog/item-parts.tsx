import type { MachinesMacView } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import type { Fact } from '../../design/facts';
import {
    AppIcon,
    Button,
    Facts,
    identityOf,
    MachineChip,
    MissingMark,
    TypeTag,
    Value,
} from '../../design/index';
import { machineKindOf } from '../../engine/provisional';
import { SYSTEM_WORDS } from '../../engine/words';
import { findMachine } from '../../model/machines';
import { machineRoute } from '../../shell/route';
import type { Route } from '../../shell/route';
import type { InstallLine, System } from './item-model';

import styles from './item.module.css';

export type Go = (to: Route) => void;

export type ItemProps = {
    go: Go;
    routeOf: (name: string) => Route;
    macs: readonly MachinesMacView[];
};

export function ItemHead({
    name,
    kind,
    version,
    kindWord,
    about,
}: {
    name: string;
    kind: string;
    version: string;
    kindWord: string;
    about: string;
}) {
    return (
        <div className={styles.head}>
            <AppIcon kind={kind} name={name} size="page" />
            <div className={styles.named}>
                <h1 className={styles.name}>
                    {name}
                    {version === '' ? null : <Value look="raised">{version}</Value>}
                    {kindWord === '' ? null : <TypeTag>{kindWord}</TypeTag>}
                </h1>
                {about === '' ? null : <p className={styles.about}>{about}</p>}
            </div>
            <Button disabled title="Not wired yet" variant="primary">
                Add to a machine
            </Button>
        </div>
    );
}

export function systemsWord(systems: readonly System[]): string {
    return systems.map((system) => SYSTEM_WORDS[system]).join(', ');
}

export function Installs({ lines }: { lines: readonly InstallLine[] }) {
    return (
        <div className={styles.installs}>
            {lines.map((line) => (
                <div className={styles.install} key={`${line.method} ${line.target}`}>
                    <span className={styles.systems}>{systemsWord(line.systems)}</span>
                    {line.method === '' ? <span /> : <Value look="raised">{line.method}</Value>}
                    <span className={styles.target}>{line.target}</span>
                </div>
            ))}
        </div>
    );
}

export function ItemChips({
    names,
    routeOf,
    go,
}: Pick<ItemProps, 'go' | 'routeOf'> & { names: readonly string[] }) {
    if (names.length === 0) {
        return null;
    }
    return (
        <span className={styles.chips}>
            {names.map((name) => (
                <button
                    className={styles.chip}
                    key={name}
                    onClick={() => {
                        go(routeOf(name));
                    }}
                    type="button"
                >
                    <AppIcon name={name} />
                    {name}
                </button>
            ))}
        </span>
    );
}

export function MachineChips({
    names,
    macs,
    go,
}: Pick<ItemProps, 'go' | 'macs'> & { names: readonly string[] }) {
    return (
        <span className={styles.chips}>
            {names.map((name) => {
                const found = findMachine(macs, name);
                if (!found) {
                    return <Value key={name}>{name}</Value>;
                }
                return (
                    <MachineChip
                        identity={found.vm ? identityOf(name) : undefined}
                        key={name}
                        kind={found.vm ? machineKindOf(found.vm) : 'mac'}
                        name={name}
                        onOpen={() => {
                            go(machineRoute(name, 'Packages'));
                        }}
                    />
                );
            })}
        </span>
    );
}

export function factOf(label: string, value: ReactNode): Fact {
    return { label, value: value ?? <MissingMark /> };
}

export function Rail({ name, facts }: { name: string; facts: readonly Fact[] }) {
    return (
        <section aria-label={`${name} in the catalog`} className={styles.rail}>
            <Facts facts={facts} layout="column" />
        </section>
    );
}

export function Two({ main, rail }: { main: ReactNode; rail: ReactNode }) {
    return (
        <div className={styles.two}>
            <div className={styles.main}>{main}</div>
            {rail}
        </div>
    );
}

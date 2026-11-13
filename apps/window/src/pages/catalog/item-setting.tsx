import type { SettingsMatrix, SettingsRow } from '@jterrazz/station-api';

import { Facts, Panel, Row, StatePill } from '../../design/index';
import { reversibleState } from '../../engine/shown';
import { CATALOG_KIND_NAMES, CATALOG_KIND_WORDS, CATALOG_LEVEL_WORDS } from '../../engine/words';
import { needsOf } from './item-model';
import { factOf, ItemChips, ItemHead, MachineChips, Rail, systemsWord, Two } from './item-parts';
import type { ItemProps } from './item-parts';
import { categoryWord } from './shelves';

import styles from './item.module.css';

function HowApplied({ row, go, routeOf }: ItemProps & { row: SettingsRow }) {
    const setting = row.Package;
    const undone = reversibleState(setting.Reversible);
    const needs = needsOf(setting);
    const asks = (setting.Inputs ?? []).map((input) =>
        input.Label === '' ? input.Name : input.Label,
    );
    const systems = systemsWord(setting.Kinds ?? []);
    const facts = [
        { label: 'On', value: systems === '' ? null : systems },
        { label: 'Asks you for', value: asks.length === 0 ? null : asks.join(', ') },
    ].filter((fact) => fact.value !== null);
    const needed = {
        label: needs.oneOf ? 'Needs one of' : 'Needs',
        value: <ItemChips go={go} names={needs.names} routeOf={routeOf} />,
    };
    return (
        <Panel
            action={undone ? <StatePill tone={undone.tone} word={undone.word} /> : undefined}
            padded
            title="How it is applied"
        >
            {facts.length === 0 ? null : <Facts facts={facts} layout="line" />}
            {needs.names.length === 0 ? null : <Facts facts={[needed]} layout="line" />}
        </Panel>
    );
}

function Changes({ row }: { row: SettingsRow }) {
    const writes = row.Package.Writes ?? [];
    const sets = row.Package.Sets ?? [];
    if (writes.length === 0 && sets.length === 0) {
        return null;
    }
    return (
        <Panel note="and nothing else" title="What it changes">
            <div className={styles.rows}>
                {writes.map((stored) => (
                    <Row
                        key={stored.Path}
                        kind={CATALOG_LEVEL_WORDS[stored.Level]}
                        sentence={stored.Note}
                        size="slim"
                        title={stored.Path}
                    />
                ))}
                {sets.map((value) => (
                    <Row
                        key={value.Key}
                        kind="value"
                        sentence={`set to ${value.Value}`}
                        size="slim"
                        title={value.Key}
                    />
                ))}
            </div>
        </Panel>
    );
}

function machinesWord(count: number): string {
    return count === 1 ? '1 machine' : `${count} machines`;
}

function countedOn(name: string, matrix: SettingsMatrix | undefined): string[] | undefined {
    const held = (matrix?.Rows ?? []).find((one) => one.Name === name);
    if (!held || !matrix) {
        return undefined;
    }
    const columns = matrix.Columns ?? [];
    return (held.Counts ?? []).flatMap((counted, index) => {
        const column = columns[index];
        return counted && column ? [column.Name] : [];
    });
}

export function SettingItem({
    row,
    matrix,
    ...props
}: ItemProps & { row: SettingsRow; matrix: SettingsMatrix | undefined }) {
    const setting = row.Package;
    const on = countedOn(row.Name, matrix);
    return (
        <>
            <ItemHead
                about={setting.Description}
                kind="setting"
                kindWord={CATALOG_KIND_WORDS.setting}
                name={row.Name}
                version=""
            />
            <Two
                main={
                    <>
                        <HowApplied {...props} row={row} />
                        <Changes row={row} />
                    </>
                }
                rail={
                    <Rail
                        facts={[
                            factOf('Kind', CATALOG_KIND_NAMES.setting),
                            factOf('Category', categoryWord(setting.Category)),
                            factOf('Declared in', null),
                            factOf(
                                on === undefined ? 'Machines' : machinesWord(on.length),
                                on === undefined ? null : (
                                    <MachineChips go={props.go} macs={props.macs} names={on} />
                                ),
                            ),
                            factOf('Comes from', null),
                        ]}
                        name={row.Name}
                    />
                }
            />
        </>
    );
}

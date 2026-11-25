import type { MachinefileFileView, MachinefileHead } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import type { Opening } from '../../navigation/window';
import { Icon } from '../../ui/icon';
import { Button, Facts, Switch, Value } from '../../ui/index';
import type { Identity } from '../../ui/index';
import { identityAt, lifetimeOf, makesOf, sizeOf, systemOf } from './model';

import styles from './machine-files.module.css';

const MIXIN = 'A mixin makes no machine by itself: it is a piece other files extend';

export type Identities = ReadonlyMap<string, Identity>;

export type Shown = 'fields' | 'local' | 'yaml';

type Choose = (file: string) => void;

function FileChips({
    names,
    choose,
    identities,
}: {
    names: readonly string[];
    choose: Choose;
    identities: Identities;
}) {
    return (
        <span className={styles.extends}>
            {names.map((name, index) => (
                <button
                    className={styles.extended}
                    key={name}
                    onClick={() => {
                        choose(name);
                    }}
                    type="button"
                >
                    <Value identity={identities.get(name) ?? identityAt(index)} look="raised">
                        {name}
                    </Value>
                </button>
            ))}
        </span>
    );
}

function madeFacts(head: MachinefileHead): { label: string; value: ReactNode }[] {
    const from = head.VM?.From ?? '';
    return [
        { label: 'Makes', value: makesOf(head) },
        { label: 'System', value: systemOf(head) },
        { label: 'Size', value: sizeOf(head) },
        { label: 'Lifetime', value: lifetimeOf(head) },
        { label: 'Starts from', value: from === '' ? '' : <Value>{from}</Value> },
    ];
}

function factsOf(
    view: MachinefileFileView,
    choose: Choose,
    identities: Identities,
): { label: string; value: ReactNode }[] {
    const extended = view.Extends ?? [];
    const own = view.Head === undefined ? [] : madeFacts(view.Head);
    return [
        ...own,
        {
            label: 'Extends',
            value:
                extended.length === 0 ? (
                    ''
                ) : (
                    <FileChips choose={choose} identities={identities} names={extended} />
                ),
        },
    ].filter((fact) => fact.value !== '');
}

export function FileHead({
    view,
    choose,
    identities,
    onNewMachine,
    shown,
    show,
}: {
    view: MachinefileFileView;
    choose: Choose;
    identities: Identities;
    onNewMachine?: ((opening: Opening) => void) | undefined;
    shown: Shown;
    show: (shown: Shown) => void;
}) {
    const named = view.Path === '' ? view.Name : (view.Path.split('/').at(-1) ?? view.Name);
    const facts = factsOf(view, choose, identities);
    return (
        <section aria-label={named} className={styles.file}>
            <div className={styles.head}>
                <span className={styles.icon}>
                    <Icon name="file" />
                </span>
                <div className={styles.named}>
                    <h2 className={styles.name}>{named}</h2>
                </div>
                <span className={styles.actions}>
                    <Switch
                        label="How the file is shown"
                        onChange={show}
                        options={[
                            { id: 'fields', label: 'Fields' },
                            { id: 'yaml', label: 'YAML' },
                        ]}
                        surface="bare"
                        value={shown === 'local' ? 'yaml' : shown}
                    />
                    {view.Head?.VM !== undefined && onNewMachine !== undefined ? (
                        <Button
                            onClick={() => {
                                onNewMachine({ from: view.Name });
                            }}
                            variant="primary"
                        >
                            New machine
                        </Button>
                    ) : null}
                </span>
            </div>
            {view.Description === '' ? null : <p className={styles.lead}>{view.Description}</p>}
            {view.Head === undefined ? <p className={styles.lead}>{MIXIN}</p> : null}
            {facts.length === 0 ? null : <Facts facts={facts} />}
        </section>
    );
}

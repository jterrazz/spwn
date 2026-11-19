import type { AccountsVaultEntry, AccountsVaultGrant } from '@jterrazz/station-api';
import { useState } from 'react';

import { useGesture } from '../../dependencies/use-gesture';
import { GestureButton, GestureOutcome, GestureRow, HeldOutcomes } from '../../kit/gestures';
import type { Route } from '../../navigation/route';
import { AppIcon, Button, Facts, Panel } from '../../ui/index';
import { Page } from '../../ui/page';
import { WITHHELD } from '../../ui/secret';
import { GrantSheet } from './grant-sheet';
import { aboutEntry, grantedTo, kindWord, VAULT_REREADS } from './passwords-of';

import styles from './passwords.module.css';

type Go = (to: Route) => void;

function GrantRow({ entry, grant }: { entry: AccountsVaultEntry; grant: AccountsVaultGrant }) {
    const remove = useGesture('grants.remove', {
        input: { Entry: entry.ID, To: grant.To },
        rereads: VAULT_REREADS,
        subject: grant.To,
    });
    const name = `Remove ${entry.Label} from ${grant.To}`;
    return (
        <GestureRow
            gesture={remove}
            lead={<AppIcon name={entry.Program} />}
            title={grant.To}
            trail={
                <GestureButton
                    gesture={remove}
                    label="Remove"
                    name={name}
                    title={`${name}?`}
                    verb="Remove"
                />
            }
        />
    );
}

function RemoveEntry({ entry, go }: { entry: AccountsVaultEntry; go: Go }) {
    const remove = useGesture('vault.remove', {
        input: { ID: entry.ID },
        onDone: () => {
            go({ page: 'passwords' });
        },
        rereads: VAULT_REREADS,
        subject: entry.Label,
    });
    return (
        <div className={styles.destructive}>
            <GestureButton
                consequence={entry.RemoveSentence}
                gesture={remove}
                label="Remove password"
                title={`Remove ${entry.Label}?`}
                variant="danger"
                verb="Remove"
            />
            <GestureOutcome gesture={remove} />
        </div>
    );
}

function Withheld() {
    return <span className={styles.withheld}>{WITHHELD}</span>;
}

export function EntryPage({ entry, go }: { entry: AccountsVaultEntry; go: Go }) {
    const [granting, setGranting] = useState(false);
    const grants = entry.Grants ?? [];
    const grant = (
        <Button
            onClick={() => {
                setGranting(true);
            }}
        >
            Grant…
        </Button>
    );
    return (
        <Page>
            <header className={styles.head}>
                <span className={styles.mark}>
                    <AppIcon name={entry.Program} />
                </span>
                <div className={styles.named}>
                    <h1 className={styles.title}>{entry.Label}</h1>
                    <p className={styles.about}>{aboutEntry(entry)}</p>
                </div>
                <RemoveEntry entry={entry} go={go} />
            </header>
            <Panel padded title="This password">
                <Facts
                    facts={[
                        { label: 'Value', value: <Withheld /> },
                        { label: 'Kind', value: kindWord(entry) },
                        { label: 'Program', value: entry.Program },
                        ...(entry.Who === undefined
                            ? []
                            : [{ label: 'Account', value: entry.Who }]),
                    ]}
                />
            </Panel>
            <HeldOutcomes>
                <Panel
                    action={grant}
                    title="Granted to"
                    {...(grants.length === 0 ? {} : { note: grantedTo(entry) })}
                >
                    {grants.map((one) => (
                        <GrantRow entry={entry} grant={one} key={one.To} />
                    ))}
                </Panel>
            </HeldOutcomes>
            {granting ? (
                <GrantSheet
                    entry={entry}
                    onClose={() => {
                        setGranting(false);
                    }}
                />
            ) : null}
        </Page>
    );
}

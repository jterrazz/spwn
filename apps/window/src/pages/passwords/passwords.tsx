import type { AccountsVaultEntry, AccountsVaultRead } from '@jterrazz/station-api';

import { Icon } from '../../design/icon';
import { AppIcon, asSentence, Panel, Row, RowHead, Verdict } from '../../design/index';
import { useWatch } from '../../engine/react';
import { verdictShown } from '../../engine/shown';
import type { Route } from '../../shell/route';
import { HeldOutcomes } from '../kit/gestures';
import { Reading, Unanswered } from '../kit/readings';
import { Page } from '../page';
import { EntryPage } from './entry';
import { addedDay, grantedTo, kindWord } from './passwords-of';

import styles from './passwords.module.css';

type Go = (to: Route) => void;

const COLUMNS = [
    { label: 'Password' },
    { label: 'Kind' },
    { label: 'Account' },
    { label: 'Added' },
    { label: 'Granted to', numeric: true },
];

const NO_VALUE_SHOWN =
    'No value is shown, copied or revealed here: a password is only granted, and removed.';

function EntryRow({ entry, go }: { entry: AccountsVaultEntry; go: Go }) {
    return (
        <Row
            cells={[entry.Who ?? '', addedDay(entry)]}
            kind={kindWord(entry)}
            lead={<AppIcon name={entry.Program} />}
            onOpen={() => {
                go({ item: entry.ID, page: 'passwords' });
            }}
            sentence={entry.Program}
            template="passwords"
            title={entry.Label}
            trail={
                <>
                    <span className={styles.granted}>{grantedTo(entry)}</span>
                    <span className={styles.chevron}>
                        <Icon name="chevron" />
                    </span>
                </>
            }
        />
    );
}

function NoValueShown() {
    return (
        <p className={styles.note}>
            <span className={styles.lock}>
                <Icon name="lock" />
            </span>
            {NO_VALUE_SHOWN}
        </p>
    );
}

function Vault({ read, go }: { read: AccountsVaultRead; go: Go }) {
    const entries = read.Entries ?? [];
    return (
        <Page>
            <Verdict
                line={asSentence(read.Verdict.Sentence)}
                size="page"
                title={`Passwords on ${read.Mac}`}
                {...verdictShown(read.Verdict)}
            />
            <HeldOutcomes>
                <Panel note={String(entries.length)} title="Passwords">
                    {entries.length === 0 ? null : (
                        <RowHead columns={COLUMNS} lead template="passwords" />
                    )}
                    {entries.map((entry) => (
                        <EntryRow entry={entry} go={go} key={entry.ID} />
                    ))}
                    <NoValueShown />
                </Panel>
            </HeldOutcomes>
        </Page>
    );
}

// invariant: the vault is This Mac's alone, so it is asked of the socket's own engine and names no target.
export function PasswordsPage({ opened, go }: { opened: string | undefined; go: Go }) {
    const listed = useWatch('vault.list', {}, undefined);
    if (listed.kind === 'failed') {
        return (
            <Page>
                <Unanswered error={listed.error} size="page" />
            </Page>
        );
    }
    if (listed.kind === 'reading') {
        return (
            <Page>
                <Reading size="page" what="the passwords of this Mac" />
            </Page>
        );
    }
    const entry = (listed.out.Entries ?? []).find((one) => one.ID === opened);
    return entry === undefined ? (
        <Vault go={go} read={listed.out} />
    ) : (
        <EntryPage entry={entry} go={go} />
    );
}

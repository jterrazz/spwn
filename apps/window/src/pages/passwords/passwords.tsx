import type { AccountsVaultEntry, AccountsVaultRead } from '@jterrazz/station-api';

import { AppIcon, asSentence, Panel, Row, Verdict } from '../../design/index';
import { useWatch } from '../../engine/react';
import { verdictShown } from '../../engine/shown';
import { sentence } from '../../model/format';
import type { Route } from '../../shell/route';
import { HeldOutcomes } from '../kit/gestures';
import { Reading, Unanswered } from '../kit/readings';
import { Page } from '../page';
import { EntryPage } from './entry';
import { addedOn, grantedTo, kindWord } from './passwords-of';

import styles from './passwords.module.css';

type Go = (to: Route) => void;

function EntryRow({ entry, go }: { entry: AccountsVaultEntry; go: Go }) {
    return (
        <Row
            kind={kindWord(entry)}
            lead={<AppIcon name={entry.Program} />}
            onOpen={() => {
                go({ item: entry.ID, page: 'passwords' });
            }}
            sentence={entry.Program}
            title={entry.Label}
            trail={<span className={styles.granted}>{grantedTo(entry)}</span>}
            trailSize="wide"
            value={sentence([entry.Who, addedOn(entry)])}
            valueSize="long"
        />
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
                    {entries.map((entry) => (
                        <EntryRow entry={entry} go={go} key={entry.ID} />
                    ))}
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

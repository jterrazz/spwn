import type { AccountsVaultEntry, AccountsVaultRead } from '@jterrazz/station-api';

import { sentence } from '../../../domain/format';
import { useWatch } from '../../dependencies/dependencies';
import { HeldOutcomes } from '../../kit/gestures';
import { Unanswered } from '../../kit/readings';
import { Skeleton, SkeletonPanel } from '../../kit/skeleton';
import type { Route } from '../../navigation/route';
import { Icon } from '../../ui/icon';
import { AppIcon, asSentence, Panel, Row, RowHead, Verdict, VerdictSkeleton } from '../../ui/index';
import { Page } from '../../ui/page';
import { verdictShown } from '../../words/shown';
import { EntryPage } from './entry';
import { addedDay, grantedTo, kindWord } from './passwords-of';

import styles from './passwords.module.css';

type Go = (to: Route) => void;

const TITLE = 'Passwords';

const COLUMNS = [
    { label: 'Password' },
    { label: 'Kind' },
    { label: 'Account' },
    { label: 'Added' },
    { label: 'Granted to', numeric: true },
];

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

function Vault({ read, go }: { read: AccountsVaultRead; go: Go }) {
    const entries = read.Entries ?? [];
    return (
        <Page>
            <Verdict
                line={asSentence(sentence([read.Verdict.Sentence, read.Verdict.Line]))}
                size="page"
                title={`Passwords on ${read.Mac}`}
                {...verdictShown(read.Verdict)}
            />
            <HeldOutcomes>
                <Panel note={String(entries.length)} title={TITLE}>
                    {entries.length === 0 ? null : (
                        <RowHead columns={COLUMNS} lead template="passwords" />
                    )}
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
                <Skeleton what="the passwords of this Mac">
                    <VerdictSkeleton size="page" />
                    <SkeletonPanel
                        count={3}
                        head={<RowHead columns={COLUMNS} lead template="passwords" />}
                        lead="icon"
                        template="passwords"
                        title={TITLE}
                    />
                </Skeleton>
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

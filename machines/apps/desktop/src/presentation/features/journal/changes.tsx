import type { CoreJournalGroup, CoreJournalReading, MachinesMacView } from '@jterrazz/station-api';

import { useRead } from '../../dependencies/dependencies';
import { Unanswered } from '../../kit/readings';
import type { Route } from '../../navigation/route';
import { Button, Panel } from '../../ui/index';
import { JournalLineRow } from './journal-line';
import { keyOfLine } from './model';

import pageStyles from '../../ui/page.module.css';
import styles from './changes.module.css';

type Go = (to: Route) => void;

const NIGHT_LINES = 3;

const HISTORY = 'History';

type GroupProps = { group: CoreJournalGroup; macs: readonly MachinesMacView[]; go: Go };

function Lines({ group, macs, go, most }: GroupProps & { most?: number }) {
    return (group.Lines ?? [])
        .slice(0, most)
        .map((line) => <JournalLineRow go={go} key={keyOfLine(line)} line={line} macs={macs} />);
}

function Day(props: GroupProps) {
    const { Title, Sum } = props.group;
    return (
        <section aria-label={Title}>
            <h3 className={styles.dayHead}>
                {Title}
                {Sum === '' ? null : <span className={styles.sum}>{Sum}</span>}
            </h3>
            <Lines {...props} />
        </section>
    );
}

function Days({
    reading,
    macs,
    go,
}: {
    reading: CoreJournalReading;
    macs: readonly MachinesMacView[];
    go: Go;
}) {
    const groups = reading.Groups ?? [];
    return (
        <>
            {groups.length === 0 && reading.Empty !== '' ? (
                <p className={pageStyles.none}>{reading.Empty}</p>
            ) : null}
            {groups.map((group) => (
                <Day go={go} group={group} key={`${group.Day}/${group.Title}`} macs={macs} />
            ))}
            {(reading.Unheard ?? []).map((unheard) => (
                <p className={pageStyles.none} key={unheard.Mac}>
                    {unheard.Why}
                </p>
            ))}
        </>
    );
}

type HistoryProps = { macs: readonly MachinesMacView[]; go: Go; chosen: null | string };

export function History({ macs, go, chosen }: HistoryProps) {
    const journal = useRead('journal.read', chosen === null ? {} : { Mac: chosen });
    if (journal.kind === 'reading') {
        return null;
    }
    return (
        <Panel note="what happened, newest first" title={HISTORY}>
            {journal.kind === 'failed' ? (
                <Unanswered error={journal.error} size="panel" />
            ) : (
                <Days go={go} macs={macs} reading={journal.out} />
            )}
        </Panel>
    );
}

type NightProps = { macs: readonly MachinesMacView[]; go: Go };

export function LastNight({ macs, go }: NightProps) {
    const journal = useRead('journal.read', { Night: true });
    if (journal.kind === 'reading') {
        return null;
    }
    if (journal.kind === 'failed') {
        return (
            <Panel title="Last night">
                <Unanswered error={journal.error} size="panel" />
            </Panel>
        );
    }
    const [night] = journal.out.Groups ?? [];
    if (night === undefined) {
        return null;
    }
    return (
        <Panel
            action={
                <Button
                    onClick={() => {
                        go({ page: 'inbox' });
                    }}
                    variant="link"
                >
                    All history
                </Button>
            }
            note={night.Sum}
            title={night.Title}
        >
            <Lines go={go} group={night} macs={macs} most={NIGHT_LINES} />
        </Panel>
    );
}

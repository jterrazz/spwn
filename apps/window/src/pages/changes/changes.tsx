import type { MachinesMacView } from '@jterrazz/station-api';

import { Button, Panel } from '../../design/index';
import { useJournal } from '../../engine/kept-journal/journal';
import type { JournalGroup } from '../../engine/kept-journal/journal';
import type { Route } from '../../shell/route';
import { NotGiven } from '../kit/readings';
import { JournalLineRow } from './journal-line';
import { keyOfLine } from './model';

import pageStyles from '../page.module.css';
import styles from './changes.module.css';

type Go = (to: Route) => void;

const NIGHT_LINES = 3;

type GroupProps = { group: JournalGroup; macs: readonly MachinesMacView[]; go: Go };

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

type HistoryProps = { macs: readonly MachinesMacView[]; go: Go; chosen: null | string };

export function History({ macs, go, chosen }: HistoryProps) {
    const journal = useJournal(chosen === null ? {} : { Mac: chosen });
    if (journal.kind === 'reading') {
        return null;
    }
    if (journal.kind !== 'read') {
        return <NotGiven title="History" what="What happened, kept by day" />;
    }
    const groups = journal.out.Groups ?? [];
    const none =
        chosen === null
            ? 'No change is kept yet.'
            : `None of the changes kept concerns ${chosen} or its machines.`;
    return (
        <Panel note="what happened, newest first" title="History">
            {groups.length === 0 ? (
                <p className={pageStyles.none}>{none}</p>
            ) : (
                groups.map((group) => <Day go={go} group={group} key={group.Title} macs={macs} />)
            )}
        </Panel>
    );
}

type NightProps = { macs: readonly MachinesMacView[]; go: Go };

export function LastNight({ macs, go }: NightProps) {
    const journal = useJournal({});
    if (journal.kind === 'reading') {
        return null;
    }
    if (journal.kind !== 'read') {
        return <NotGiven title="Last night" what="What moved, held or failed overnight" />;
    }
    const night = (journal.out.Groups ?? []).find((group) => group.Night);
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

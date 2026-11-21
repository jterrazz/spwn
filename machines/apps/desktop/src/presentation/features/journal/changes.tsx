import type { CoreJournalGroup, CoreJournalReading, MachinesMacView } from '@jterrazz/station-api';
import { useState } from 'react';

import { useRead } from '../../dependencies/dependencies';
import { Unanswered } from '../../kit/readings';
import type { Route } from '../../navigation/route';
import { Button, Panel } from '../../ui/index';
import { SHOW_OLDER_WORD } from '../../words/words';
import { JournalLineRow } from './journal-line';
import { keyOfLine } from './model';
import {
    askedOf,
    olderAsked,
    olderOffered,
    pageInput,
    pagingOf,
    saidAfter,
    unheardUnsaid,
} from './pages';

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

type PageProps = {
    macs: readonly MachinesMacView[];
    go: Go;
    chosen: null | string;
    asked: readonly string[];
    said: readonly string[];
    onOlder: (before: string) => void;
};

function Days({ reading, said, macs, go }: PageProps & { reading: CoreJournalReading }) {
    const groups = reading.Groups ?? [];
    return (
        <>
            {groups.length === 0 && reading.Empty !== '' ? (
                <p className={pageStyles.none}>{reading.Empty}</p>
            ) : null}
            {groups.map((group) => (
                <Day go={go} group={group} key={`${group.Day}/${group.Title}`} macs={macs} />
            ))}
            {unheardUnsaid(said, reading).map((unheard) => (
                <p className={pageStyles.none} key={unheard.Mac}>
                    {unheard.Why}
                </p>
            ))}
        </>
    );
}

function Pages({ reading, ...props }: PageProps & { reading: CoreJournalReading }) {
    const next = olderOffered(reading);
    let older = null;
    if (next !== undefined && props.asked.includes(next)) {
        older = <Older {...props} before={next} said={saidAfter(props.said, reading)} />;
    } else if (next !== undefined) {
        older = (
            <div className={styles.older}>
                <Button
                    onClick={() => {
                        props.onOlder(next);
                    }}
                    variant="link"
                >
                    {SHOW_OLDER_WORD}
                </Button>
            </div>
        );
    }
    return (
        <>
            <Days {...props} reading={reading} />
            {older}
        </>
    );
}

function Older({ before, ...props }: PageProps & { before: string }) {
    const page = useRead('journal.read', pageInput(props.chosen, before));
    if (page.kind === 'reading') {
        return null;
    }
    if (page.kind === 'failed') {
        return <Unanswered error={page.error} size="panel" />;
    }
    return <Pages {...props} reading={page.out} />;
}

type HistoryProps = { macs: readonly MachinesMacView[]; go: Go; chosen: null | string };

export function History({ macs, go, chosen }: HistoryProps) {
    const journal = useRead('journal.read', pageInput(chosen));
    const [paging, setPaging] = useState(() => pagingOf(chosen));
    if (journal.kind === 'reading') {
        return null;
    }
    const onOlder = (before: string) => {
        setPaging((was) => olderAsked(was, chosen, before));
    };
    return (
        <Panel note="what happened, newest first" title={HISTORY}>
            {journal.kind === 'failed' ? (
                <Unanswered error={journal.error} size="panel" />
            ) : (
                <Pages
                    asked={askedOf(paging, chosen)}
                    chosen={chosen}
                    go={go}
                    macs={macs}
                    onOlder={onOlder}
                    reading={journal.out}
                    said={[]}
                />
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

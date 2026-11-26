import type { WikiServedBlock, WikiServedDecision } from '@jterrazz/station-api';

import { StateDot } from '../../ui/index';
import { WIKI_DECISION_WORDS } from '../../words/wiki';
import { DotIcon } from './dot-icon';
import { Findings } from './fields';
import type { Reader } from './fields';

import styles from './wiki.module.css';

function Record({ decision, reader }: { decision: WikiServedDecision; reader: Reader }) {
    if (!decision.Resolves) {
        return <span className={styles.decisionUnread}>{WIKI_DECISION_WORDS.unread}</span>;
    }
    return (
        <button
            className={styles.decisionRecord}
            onClick={() => {
                reader.open(decision.Page);
            }}
            type="button"
        >
            {decision.Title}
        </button>
    );
}

export function DecisionLook({
    block,
    decision,
    reader,
}: {
    block: WikiServedBlock;
    decision: WikiServedDecision;
    reader: Reader;
}) {
    const amended = decision.AmendedBy ?? [];
    return (
        <article aria-label={`${block.Title} ${block.Word}`} className={styles.decision}>
            <header className={styles.blockHead}>
                <DotIcon grids={reader.grids} name={block.Icon} size="small" />
                <span className={styles.kind}>{block.Title}</span>
                <span className={styles.word}>{block.Word}</span>
            </header>
            <div className={styles.decisionHead}>
                <span className={styles.decisionNumber}>
                    {decision.Number === '' ? block.Word : decision.Number}
                </span>
                {decision.Status === '' ? null : (
                    <span className={styles.decisionStatus}>
                        <StateDot tone={decision.Accepted ? 'ok' : 'warn'} word={decision.Status} />
                        {decision.Status}
                    </span>
                )}
                {decision.Date === '' ? null : (
                    <span className={styles.decisionDate}>{decision.Date}</span>
                )}
            </div>
            {amended.length === 0 ? null : (
                <p className={styles.decisionAmended}>
                    {WIKI_DECISION_WORDS.amendedBy} {amended.join(', ')}
                </p>
            )}
            <dl className={styles.fields}>
                <div className={styles.field}>
                    <dt>{WIKI_DECISION_WORDS.record}</dt>
                    <dd>
                        <Record decision={decision} reader={reader} />
                    </dd>
                </div>
                <div className={styles.field}>
                    <dt>{WIKI_DECISION_WORDS.decides}</dt>
                    <dd className={styles.decisionDecides}>{decision.Decides}</dd>
                </div>
            </dl>
            <Findings findings={block.Finding} />
        </article>
    );
}

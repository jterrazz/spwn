import type { WikiServedField, WikiServedFinding, WikiServedRef } from '@jterrazz/station-api';

import { StateDot } from '../../ui/index';
import { WIKI_LEVEL_TONES, WIKI_LEVEL_WORDS } from '../../words/wiki';
import type { Grids } from './dot-icon';

import styles from './wiki.module.css';

export type Reader = { grids: Grids; open: (page: string) => void; seat: string };

// value: the element a served address is drawn as, so a finding can lead the reader to it.
export function placeOf(address: string): string {
    return `wiki-${address.replaceAll(/[^\w-]/gu, '-')}`;
}

// value: the mark of a block or a group the engine says needs the owner.
export function attentionOf(needsYou: boolean): string {
    return needsYou ? (styles.needsYou ?? '') : '';
}

// value: the page a ref opens: its address up to the block it names.
function pageOf(address: string): string {
    const [page = address] = address.split('#');
    return page;
}

export function Ref({ to, reader }: { to: WikiServedRef; reader: Reader }) {
    if (!to.Resolves) {
        return (
            <span className={`${styles.chip} ${styles.unresolved}`} title={to.Address}>
                {to.Word}
            </span>
        );
    }
    return (
        <button
            className={styles.chip}
            onClick={() => {
                reader.open(pageOf(to.Address));
            }}
            title={to.Address}
            type="button"
        >
            {to.Word}
        </button>
    );
}

export function FieldValue({ field, reader }: { field: WikiServedField; reader: Reader }) {
    const refs = field.Refs ?? [];
    if (refs.length === 0) {
        return <span className={styles.fieldText}>{field.Text}</span>;
    }
    return (
        <span className={styles.refs}>
            {refs.map((ref) => (
                <Ref key={ref.Address} reader={reader} to={ref} />
            ))}
        </span>
    );
}

// invariant: a field that says the block's line again is not drawn twice: the line heads the block.
export function Fields({
    fields,
    line,
    reader,
}: {
    fields: null | readonly WikiServedField[];
    line: string;
    reader: Reader;
}) {
    const shown = (fields ?? []).filter((field) => field.Text !== line || field.Text === '');
    if (shown.length === 0) {
        return null;
    }
    return (
        <dl className={styles.fields}>
            {shown.map((field) => (
                <div className={styles.field} key={field.Name}>
                    <dt>{field.Name.replaceAll('_', ' ')}</dt>
                    <dd>
                        <FieldValue field={field} reader={reader} />
                    </dd>
                </div>
            ))}
        </dl>
    );
}

export function Findings({ findings }: { findings: null | readonly WikiServedFinding[] }) {
    const listed = findings ?? [];
    if (listed.length === 0) {
        return null;
    }
    return (
        <ul aria-label="Findings" className={styles.findings}>
            {listed.map((finding) => (
                <li
                    className={styles.finding}
                    data-tone={WIKI_LEVEL_TONES[finding.Level]}
                    key={`${finding.Rule}-${String(finding.Line)}`}
                >
                    <StateDot
                        tone={WIKI_LEVEL_TONES[finding.Level]}
                        word={WIKI_LEVEL_WORDS[finding.Level]}
                    />
                    <span className={styles.findingLevel}>{WIKI_LEVEL_WORDS[finding.Level]}</span>
                    <span>{finding.Says}</span>
                </li>
            ))}
        </ul>
    );
}

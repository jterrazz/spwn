import type {
    WikiServed,
    WikiServedFinding,
    WikiServedFloor,
    WikiServedSection,
} from '@jterrazz/station-api';

import { Picks, StateDot } from '../../ui/index';
import { countOf } from '../../words/engine-words';
import {
    WIKI_ATTENTION_WORDS,
    WIKI_LEVEL_TONES,
    WIKI_LEVEL_WORDS,
    WIKI_WORDS,
} from '../../words/wiki';
import { DotIcon } from './dot-icon';
import { Findings, placeOf } from './fields';
import type { Reader } from './fields';
import { Bento, isLight, runsOf } from './items';
import { Prose } from './prose';

import styles from './wiki.module.css';

const LISTED = 5;

// value: each finding of the page, in the engine's order, as a way to where it sits.
function Attention({ page, findings }: { page: string; findings: readonly WikiServedFinding[] }) {
    if (findings.length === 0) {
        return null;
    }
    const listed = findings.slice(0, LISTED);
    const rest = findings.length - listed.length;
    return (
        <ul aria-label={WIKI_ATTENTION_WORDS.title} className={styles.attention}>
            {listed.map((finding) => (
                <li key={`${finding.At}-${finding.Rule}-${String(finding.Line)}`}>
                    <button
                        className={styles.attentionRow}
                        data-tone={WIKI_LEVEL_TONES[finding.Level]}
                        onClick={() => {
                            document
                                .querySelector(`#${placeOf(finding.At === '' ? page : finding.At)}`)
                                ?.scrollIntoView({ behavior: 'smooth', block: 'center' });
                        }}
                        title={finding.Says}
                        type="button"
                    >
                        <StateDot
                            tone={WIKI_LEVEL_TONES[finding.Level]}
                            word={WIKI_LEVEL_WORDS[finding.Level]}
                        />
                        <span className={styles.attentionWord}>
                            {finding.Word === '' ? WIKI_ATTENTION_WORDS.page : finding.Word}
                        </span>
                        <span className={styles.attentionLevel}>
                            {WIKI_LEVEL_WORDS[finding.Level]}
                        </span>
                    </button>
                </li>
            ))}
            {rest === 0 ? null : (
                <li className={styles.sideFoot}>
                    +{rest} {WIKI_ATTENTION_WORDS.others}
                </li>
            )}
        </ul>
    );
}

function Side({ served, onBudget }: { served: WikiServed; onBudget: (budget: string) => void }) {
    const budgets = (served.Budgets ?? []).map((budget) => ({ id: budget, title: budget }));
    const { NeedsYou: needs, Unshown: folded } = served;
    return (
        <aside aria-label={WIKI_WORDS.page} className={styles.side}>
            <span className={styles.sideLabel}>{WIKI_WORDS.budget}</span>
            <Picks
                label={WIKI_WORDS.budget}
                look="chips"
                onChange={onBudget}
                options={budgets}
                value={served.Budget}
            />
            {needs === 0 ? null : (
                <span className={styles.needs}>
                    <StateDot
                        tone={WIKI_LEVEL_TONES['needs-you']}
                        word={WIKI_LEVEL_WORDS['needs-you']}
                    />
                    <b>{needs}</b> {WIKI_LEVEL_WORDS['needs-you']}
                </span>
            )}
            <Attention findings={served.Attention ?? []} page={served.Page.ID} />
            {folded === 0 ? null : (
                <span className={styles.sideFoot}>
                    {countOf('block', folded)} {WIKI_WORDS.folded}
                </span>
            )}
        </aside>
    );
}

function keyOf(section: undefined | WikiServedSection): string {
    return section === undefined || section.Anchor === '' ? 'top' : section.Anchor;
}

function shelved(section: WikiServedSection): boolean {
    return section.Heading !== '' && !section.Rests && isLight(section.Items ?? []);
}

// invariant: light sections that follow each other stand side by side as tiles of one shelf; any other section is
// invariant: its own row.
function shelvesOf(sections: readonly WikiServedSection[]): WikiServedSection[][] {
    const shelves: WikiServedSection[][] = [];
    for (const section of sections) {
        const last = shelves.at(-1);
        const before = last?.at(-1);
        if (last !== undefined && before !== undefined && shelved(before) && shelved(section)) {
            last.push(section);
        } else {
            shelves.push([section]);
        }
    }
    return shelves;
}

function Section({
    section,
    reader,
    page,
    tiled,
}: {
    section: WikiServedSection;
    reader: Reader;
    page: string;
    tiled: boolean;
}) {
    const items = section.Items ?? [];
    if (section.Rests) {
        return (
            <section aria-label={section.Heading} className={styles.rests}>
                <h2 className={styles.restsTitle}>
                    <DotIcon grids={reader.grids} name="decide" size="small" />
                    {section.Heading}
                </h2>
                <Bento items={items} reader={reader} />
            </section>
        );
    }
    return (
        <section
            aria-label={section.Heading === '' ? page : section.Heading}
            className={tiled ? styles.sectionTile : styles.section}
        >
            {section.Heading === '' ? null : (
                <h2 className={styles.sectionTitle}>{section.Heading}</h2>
            )}
            <Bento items={items} reader={reader} />
        </section>
    );
}

function Ladder({ floors }: { floors: readonly WikiServedFloor[] }) {
    return (
        <ol aria-label={WIKI_WORDS.ladder} className={styles.ladder}>
            {floors.map((floor) => (
                <li
                    aria-current={floor.Here ? 'step' : undefined}
                    className={floor.Here ? styles.rungHere : styles.rung}
                    key={floor.Index}
                    title={floor.Question}
                >
                    <span className={styles.rungDot} />
                    <span className={styles.rungName}>
                        {WIKI_WORDS.floorShort}
                        {floor.Index} {floor.Name}
                    </span>
                    <span className={styles.rungCount}>{floor.Blocks}</span>
                </li>
            ))}
        </ol>
    );
}

function Head({ served, onBudget }: { served: WikiServed; onBudget: (budget: string) => void }) {
    const { Floor, Page } = served;
    return (
        <header className={styles.pageHead} id={placeOf(Page.ID)}>
            <div className={styles.headMain}>
                <Ladder floors={served.Floors ?? []} />
                <div className={styles.pageMeta}>
                    <span className={styles.question}>{Floor.Question}</span>
                </div>
                <h1 className={styles.pageTitle}>{Page.Word}</h1>
                <div className={styles.face}>
                    <Prose paragraphs={served.Paragraph ?? []} />
                </div>
                <Findings findings={served.Findings} />
            </div>
            <Side onBudget={onBudget} served={served} />
        </header>
    );
}

export function Reading({
    served,
    reader,
    onBudget,
}: {
    served: WikiServed;
    reader: Reader;
    onBudget: (budget: string) => void;
}) {
    const { Page } = served;
    const sections = (served.Sections ?? []).filter(
        (section) => runsOf(section.Items ?? []).length > 0,
    );
    return (
        <article aria-label={Page.Word} className={styles.reading}>
            <Head onBudget={onBudget} served={served} />
            {shelvesOf(sections).map((shelf) =>
                shelf.length === 1 ? (
                    shelf.map((section) => (
                        <Section
                            key={keyOf(section)}
                            page={Page.Word}
                            reader={reader}
                            section={section}
                            tiled={false}
                        />
                    ))
                ) : (
                    <div className={styles.shelf} key={keyOf(shelf[0])}>
                        {shelf.map((section) => (
                            <Section
                                key={keyOf(section)}
                                page={Page.Word}
                                reader={reader}
                                section={section}
                                tiled
                            />
                        ))}
                    </div>
                ),
            )}
        </article>
    );
}

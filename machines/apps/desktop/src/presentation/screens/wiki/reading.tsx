import type {
    WikiServed,
    WikiServedFinding,
    WikiServedItem,
    WikiServedSection,
} from '@jterrazz/station-api';

import { SEPARATOR } from '../../../domain/separator';
import { Picks, StateDot } from '../../ui/index';
import { countOf } from '../../words/engine-words';
import { WIKI_LEVEL_TONES, WIKI_LEVEL_WORDS, WIKI_WORDS } from '../../words/wiki';
import { DotIcon } from './dot-icon';
import { Findings } from './fields';
import type { Reader } from './fields';
import { Bento, isLight, runsOf } from './items';
import { Prose } from './prose';

import styles from './wiki.module.css';

type Tally = { needs: number; folded: number };

function needsIn(findings: null | readonly WikiServedFinding[]): number {
    return (findings ?? []).filter((finding) => finding.Level === 'needs-you').length;
}

function tallyOf(items: readonly WikiServedItem[], into: Tally): void {
    for (const { Block, Group } of items) {
        if (Block !== null) {
            into.needs += needsIn(Block.Finding);
            into.folded += Block.Shown === 'none' ? 1 : 0;
        }
        if (Group !== null) {
            const inner: Tally = { folded: 0, needs: 0 };
            tallyOf(Group.Items ?? [], inner);
            into.needs += inner.needs;
            into.folded += Group.Folded === '' ? inner.folded : Group.Blocks;
        }
    }
}

// value: what the head counts of the page as served: the findings that need the owner, and the blocks this reading
// value: time does not show.
function tally(served: WikiServed): Tally {
    const into: Tally = { folded: 0, needs: needsIn(served.Findings) };
    for (const section of served.Sections ?? []) {
        tallyOf(section.Items ?? [], into);
    }
    return into;
}

function Side({ served, onBudget }: { served: WikiServed; onBudget: (budget: string) => void }) {
    const budgets = (served.Budgets ?? []).map((budget) => ({ id: budget, title: budget }));
    const { needs, folded } = tally(served);
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

function Head({ served, onBudget }: { served: WikiServed; onBudget: (budget: string) => void }) {
    const { Floor, Page } = served;
    return (
        <header className={styles.pageHead}>
            <div className={styles.headMain}>
                <div className={styles.pageMeta}>
                    <span className={styles.floor}>
                        {WIKI_WORDS.floor} {Floor.Index}
                        {SEPARATOR}
                        {Floor.Name}
                    </span>
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

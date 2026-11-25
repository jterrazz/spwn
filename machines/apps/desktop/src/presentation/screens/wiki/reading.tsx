import type { WikiServed } from '@jterrazz/station-api';

import { SEPARATOR } from '../../../domain/separator';
import { Picks } from '../../ui/index';
import { WIKI_WORDS } from '../../words/wiki';
import { Findings } from './fields';
import type { Reader } from './fields';
import { Items } from './items';
import { Prose } from './prose';

import styles from './wiki.module.css';

export function Reading({
    served,
    reader,
    onBudget,
}: {
    served: WikiServed;
    reader: Reader;
    onBudget: (budget: string) => void;
}) {
    const { Floor, Page } = served;
    const budgets = (served.Budgets ?? []).map((budget) => ({ id: budget, title: budget }));
    const sections = (served.Sections ?? []).filter(
        (section) => section.Heading !== '' || (section.Items ?? []).length > 0,
    );
    return (
        <article aria-label={Page.Word} className={styles.reading}>
            <header className={styles.pageHead}>
                <div className={styles.pageMeta}>
                    <span className={styles.floor}>
                        {WIKI_WORDS.floor} {Floor.Index}
                        {SEPARATOR}
                        {Floor.Name}
                    </span>
                    <span className={styles.question}>{Floor.Question}</span>
                    <span className={styles.budget}>
                        <Picks
                            label={WIKI_WORDS.budget}
                            look="chips"
                            onChange={onBudget}
                            options={budgets}
                            value={served.Budget}
                        />
                    </span>
                </div>
                <h1 className={styles.pageTitle}>{Page.Word}</h1>
                <div className={styles.face}>
                    <Prose paragraphs={served.Paragraph ?? []} />
                </div>
                <Findings findings={served.Findings} />
            </header>
            {sections.map((section) => (
                <section
                    aria-label={section.Heading === '' ? Page.Word : section.Heading}
                    className={styles.section}
                    key={section.Anchor}
                >
                    {section.Heading === '' ? null : (
                        <h2 className={styles.sectionTitle}>{section.Heading}</h2>
                    )}
                    <Items items={section.Items ?? []} reader={reader} />
                </section>
            ))}
        </article>
    );
}

import type { WikiPageHead, WikiServed } from '@jterrazz/station-api';

import { SEPARATOR } from '../../../domain/separator';
import { countOf } from '../../words/engine-words';
import { WIKI_WORDS } from '../../words/wiki';
import { DotIcon } from './dot-icon';
import type { Reader } from './fields';

import styles from './wiki.module.css';

function counts(page: WikiPageHead): string {
    return [
        `${WIKI_WORDS.floor} ${String(page.Floor)}`,
        page.Hard > 0 ? countOf('hard-finding', page.Hard) : '',
        page.Soft > 0 ? countOf('remark', page.Soft) : '',
    ]
        .filter((part) => part !== '')
        .join(SEPARATOR);
}

function PageTile({ page, reader, icon }: { page: WikiPageHead; reader: Reader; icon: string }) {
    return (
        <button
            className={styles.pageTile}
            onClick={() => {
                reader.open(page.ID);
            }}
            type="button"
        >
            <DotIcon grids={reader.grids} name={page.Icon === '' ? icon : page.Icon} size="mid" />
            <span className={styles.pageTileWord}>{page.Word}</span>
            <span className={styles.pageTileLine}>{page.Line}</span>
            <span className={styles.pageTileFoot}>{counts(page)}</span>
        </button>
    );
}

export function WikiHome({ served, reader }: { served: WikiServed; reader: Reader }) {
    const pages = served.Pages ?? [];
    const seats = served.Seats ?? [];
    return (
        <article aria-label={WIKI_WORDS.home} className={styles.reading}>
            <header className={styles.pageHead}>
                <h1 className={styles.pageTitle}>{WIKI_WORDS.home}</h1>
                <p className={styles.homeLine}>{WIKI_WORDS.homeLine}</p>
                <p className={styles.homeCounts}>
                    {[
                        countOf('wiki-page', pages.length),
                        countOf('hard-finding', served.Verdict.Hard),
                        countOf('remark', served.Verdict.Soft),
                    ].join(SEPARATOR)}
                </p>
            </header>
            {seats.map((seat) => {
                const held = pages.filter((page) => page.Seat === seat.ID);
                return held.length === 0 ? null : (
                    <section aria-label={seat.Name} className={styles.section} key={seat.ID}>
                        <h2 className={styles.seatTitle}>
                            <DotIcon grids={reader.grids} name={seat.Icon} size="small" />
                            {seat.Name}
                        </h2>
                        <div className={styles.pageTiles}>
                            {held.map((page) => (
                                <PageTile
                                    icon={seat.Icon}
                                    key={page.ID}
                                    page={page}
                                    reader={reader}
                                />
                            ))}
                        </div>
                    </section>
                );
            })}
        </article>
    );
}

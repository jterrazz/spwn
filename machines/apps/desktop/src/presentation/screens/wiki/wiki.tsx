import type { WikiReadInput, WikiServed } from '@jterrazz/station-api';
import { useState } from 'react';

import { useRead } from '../../dependencies/dependencies';
import { Unanswered } from '../../kit/readings';
import { Placeheld, Skeleton } from '../../kit/skeleton';
import type { Route } from '../../navigation/route';
import { Bone } from '../../ui/index';
import { WIKI_WORDS } from '../../words/wiki';
import type { Reader } from './fields';
import { WikiHome } from './home';
import { Reading } from './reading';
import { SeatsBar } from './seats';

import styles from './wiki.module.css';

function inputOf(page: string | undefined, budget: string | undefined): WikiReadInput {
    return {
        ...(page === undefined ? {} : { Page: page }),
        ...(budget === undefined ? {} : { Budget: budget }),
    };
}

function readerOf(served: WikiServed, go: (to: Route) => void): Reader {
    const active = (served.Seats ?? []).find((seat) => seat.Active);
    return {
        grids: served.Icons ?? {},
        open: (page) => {
            go({ item: page, page: 'wiki' });
        },
        seat: active?.Name ?? '',
    };
}

function Bones() {
    return (
        <Skeleton what={WIKI_WORDS.home}>
            <div className={styles.bones}>
                <Bone width="wide" />
                <Bone width="short" />
                <Bone width="long" />
                <Bone width="long" />
                <Bone width="mid" />
            </div>
        </Skeleton>
    );
}

// invariant: the engine served every tier, fold and order the page shows at the budget it was asked; the window draws
// invariant: what it was served and keeps only the budget asked, which a page opened anew forgets.
export function WikiPage({ item, go }: { item: string | undefined; go: (to: Route) => void }) {
    const [budget, setBudget] = useState<string | undefined>(undefined);
    const reading = useRead('wiki.read', inputOf(item, budget), undefined, { placeholder: true });
    if (reading.kind === 'reading') {
        return <Bones />;
    }
    if (reading.kind === 'failed') {
        return (
            <div className={styles.reading}>
                <Unanswered error={reading.error} size="page" />
            </div>
        );
    }
    const served = reading.out;
    const reader = readerOf(served, go);
    return (
        <div className={styles.wiki}>
            <SeatsBar
                grids={reader.grids}
                home={item === undefined}
                open={(page) => {
                    go(page === undefined ? { page: 'wiki' } : { item: page, page: 'wiki' });
                }}
                seats={served.Seats ?? []}
            />
            <Placeheld held={reading.placeholder === true}>
                {item === undefined ? (
                    <WikiHome reader={reader} served={served} />
                ) : (
                    <Reading onBudget={setBudget} reader={reader} served={served} />
                )}
            </Placeheld>
        </div>
    );
}

import { useOffers, useRead } from '../dependencies/dependencies';
import { StateDot } from '../ui/index';
import { verdictShown } from '../words/shown';
import { RELEASE_WORDS } from '../words/words';
import type { Route } from './route';

import styles from './sidebar.module.css';

type Go = (to: Route) => void;

function Behind({ go }: { go: Go }) {
    const reading = useRead('engine.read-release', {});
    if (reading.kind !== 'read' || !reading.out.Behind || reading.out.Latest === undefined) {
        return null;
    }
    const { Head, Latest } = reading.out;
    return (
        <button
            className={styles.release}
            onClick={() => {
                go({ page: 'engine' });
            }}
            type="button"
        >
            <span aria-hidden="true">
                <StateDot {...verdictShown(Head)} />
            </span>
            <span className={styles.releaseText}>
                {Latest} {RELEASE_WORDS.available}
            </span>
            <span className={styles.releaseGo}>{RELEASE_WORDS.update}</span>
        </button>
    );
}

export function ReleaseRow({ go }: { go: Go }) {
    return useOffers('engine.read-release') ? <Behind go={go} /> : null;
}

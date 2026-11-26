import type { ReactNode } from 'react';

import { useOffers, useRead, useWindowUpdate } from '../dependencies/dependencies';
import { StateDot } from '../ui/index';
import { ownState, verdictShown } from '../words/shown';
import type { Shown } from '../words/shown';
import { RELEASE_WORDS, WINDOW_RELEASE_WORDS, windowAvailableWord } from '../words/words';
import type { Route } from './route';

import styles from './column.module.css';

type Go = (to: Route) => void;

type FootProps = { go: () => void; dot: Shown; said: ReactNode; verb: string };

function Foot({ go, dot, said, verb }: FootProps) {
    return (
        <button className={styles.release} onClick={go} type="button">
            <span aria-hidden="true">
                <StateDot {...dot} />
            </span>
            <span className={styles.releaseText}>{said}</span>
            <span className={styles.releaseGo}>{verb}</span>
        </button>
    );
}

function WindowOut({ go }: { go: Go }) {
    const { state } = useWindowUpdate();
    if (state.kind !== 'out' && state.kind !== 'failed') {
        return null;
    }
    return (
        <Foot
            dot={ownState('window-out')}
            go={() => {
                go({ page: 'settings' });
            }}
            said={windowAvailableWord(state.version)}
            verb={WINDOW_RELEASE_WORDS.update}
        />
    );
}

function Behind({ go }: { go: Go }) {
    const reading = useRead('engine.read-release', {});
    if (reading.kind !== 'read' || !reading.out.Behind || reading.out.Latest === undefined) {
        return <WindowOut go={go} />;
    }
    const { Head, Latest } = reading.out;
    return (
        <Foot
            dot={verdictShown(Head)}
            go={() => {
                go({ page: 'engine' });
            }}
            said={
                <>
                    {Latest} {RELEASE_WORDS.available}
                </>
            }
            verb={RELEASE_WORDS.update}
        />
    );
}

// invariant: one row at the foot: the engine's release while it is behind, else the window's own while one is out.
export function ReleaseRow({ go }: { go: Go }) {
    return useOffers('engine.read-release') ? <Behind go={go} /> : <WindowOut go={go} />;
}

import type { Unfit } from '../../../domain/fit';
import { useReach, useUpdate, useWindowUpdate } from '../../dependencies/dependencies';
import { WindowRelease } from '../../features/window-release/window-release';
import { Button, StateMark } from '../../ui/index';
import {
    FIT_APP_UPDATE_WORD,
    FIT_LEAD_WORDS,
    FIT_PANEL_WORD,
    FIT_REMEDY_WORDS,
    FIT_UPDATE_TO,
    fitState,
} from '../../words/fit';
import { RELEASE_WORDS } from '../../words/words';
import { Acts, Command, Fine, Head } from './core-parts';
import { UpdatePanel } from './update';

import styles from './core.module.css';

function EngineOlder({ executable }: { executable: string | undefined }) {
    const { flow } = useUpdate();
    return (
        <>
            <Command />
            <Acts>
                <Button
                    onClick={() => {
                        flow.update({ executable, to: FIT_UPDATE_TO });
                    }}
                    variant="primary"
                >
                    {RELEASE_WORDS.update}
                </Button>
            </Acts>
        </>
    );
}

function AppOlder() {
    const { state } = useWindowUpdate();
    if (state.kind === 'unknown' || state.kind === 'unsupported') {
        return <Fine>{FIT_APP_UPDATE_WORD}</Fine>;
    }
    return <WindowRelease />;
}

function Verdict({ fit, executable }: { fit: Unfit; executable: string | undefined }) {
    return (
        <section aria-label={FIT_PANEL_WORD} className={styles.panel}>
            <span className={styles.pill}>
                <StateMark {...fitState(fit)} />
            </span>
            <Head title={FIT_REMEDY_WORDS[fit]}>{FIT_LEAD_WORDS[fit]}</Head>
            {fit === 'engine-older' ? <EngineOlder executable={executable} /> : <AppOlder />}
        </section>
    );
}

// invariant: the frame's content while the engine it reaches is judged incompatible: the verdict and its remedy,
// invariant: and nothing of the engine's but its hello.
export function VerdictPage() {
    const reach = useReach();
    const update = useUpdate();
    if (update.state.kind !== 'idle') {
        return (
            <div className={styles.center}>
                <UpdatePanel flow={update.flow} state={update.state} />
            </div>
        );
    }
    return (
        <div className={styles.center}>
            {reach.kind === 'incompatible' ? (
                <Verdict executable={reach.executable} fit={reach.fit} />
            ) : null}
        </div>
    );
}

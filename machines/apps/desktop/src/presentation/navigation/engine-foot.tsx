import type { EngineReadOutput } from '@jterrazz/station-api';

import { useCore, useReach } from '../dependencies/dependencies';
import type { Reading } from '../dependencies/dependencies';
import { StateMark } from '../ui/index';
import { fitFootWord, fitState } from '../words/fit';
import { coreState, ownState } from '../words/shown';
import { CORE_FOOT_WORDS } from '../words/words';
import { useEngineLine } from './engine-line';
import type { EngineLine } from './engine-line';
import type { Route } from './route';

import styles from './path-bar.module.css';

export type EngineSaid = { engine: Reading<EngineReadOutput>; quiet: boolean };

// value: the engine's line as the frame says it: with no engine, what the core or the verdict says, else the
// value: engine's own line.
export function useFrameLine({ engine, quiet }: EngineSaid): EngineLine {
    const { state } = useCore();
    const reach = useReach();
    const line = useEngineLine(engine);
    if (!quiet) {
        return line;
    }
    if (reach.kind === 'incompatible') {
        return {
            dot: fitState(reach.fit),
            opens: false,
            said: fitFootWord(reach.fit),
            version: '',
        };
    }
    const dot = state.kind === 'installed' ? ownState('not-started') : coreState(state.kind);
    return { dot, opens: false, said: CORE_FOOT_WORDS[state.kind], version: '' };
}

type Go = { go: (to: Route) => void };

function FootLine({ line, go }: Go & { line: EngineLine }) {
    if (!line.opens) {
        return (
            <div className={styles.foot}>
                <StateMark said="name" tone={line.dot.tone} word={line.dot.word} />
                <span>{line.said}</span>
            </div>
        );
    }
    return (
        <button
            className={styles.foot}
            onClick={() => {
                go({ page: 'engine' });
            }}
            type="button"
        >
            <span aria-hidden="true">
                <StateMark said="name" tone={line.dot.tone} word={line.dot.word} />
            </span>
            <span>{line.said}</span>
        </button>
    );
}

// value: the engine's line at the column's foot: its state's dot and its words, opening the Engine page, in a region
// value: of its own so a line of Needs you that says the same is told from it.
export function EngineFoot({ engine, quiet, go }: EngineSaid & Go) {
    const line = useFrameLine({ engine, quiet });
    return (
        <section aria-label="Engine state" className={styles.footing}>
            <FootLine go={go} line={line} />
        </section>
    );
}

// invariant: the bar and the card mark the engine only when it is not fine: absent, connecting, refused,
// invariant: incompatible or in a state of its own that is not ok; a read still under way marks nothing.
export function engineMarked({ engine, quiet }: EngineSaid, line: EngineLine): boolean {
    if (quiet || !line.opens || engine.kind === 'failed') {
        return true;
    }
    return engine.kind === 'read' && line.dot.tone !== 'ok';
}

// value: the engine's mark at the bar's right, so its absence is said with the column hidden; it names itself as
// value: the column's foot does, and opens the Engine page when there is one to open.
export function EngineMark({ engine, quiet, go }: EngineSaid & Go) {
    const line = useFrameLine({ engine, quiet });
    if (!engineMarked({ engine, quiet }, line)) {
        return null;
    }
    const dot = (
        <span aria-hidden="true">
            <StateMark said="name" tone={line.dot.tone} word={line.dot.word} />
        </span>
    );
    if (!line.opens) {
        return (
            <span aria-label={line.said} className={styles.mark} role="status" title={line.said}>
                {dot}
            </span>
        );
    }
    return (
        <button
            aria-label={line.said}
            className={styles.mark}
            onClick={() => {
                go({ page: 'engine' });
            }}
            title={line.said}
            type="button"
        >
            {dot}
        </button>
    );
}

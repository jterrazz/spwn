import type { EngineReadOutput } from '@jterrazz/station-api';

import { SEPARATOR } from '../../domain/separator';
import { useReach } from '../dependencies/dependencies';
import type { Reading } from '../dependencies/dependencies';
import { failedState } from '../words/failed';
import { engineState, ownState } from '../words/shown';
import type { Shown } from '../words/shown';

export type EngineLine = { dot: Shown; said: string; version: string; opens: boolean };

// value: the engine's one line, read where the window names its engine: the sidebar's foot, the bar's mark and Settings.
export function useEngineLine(engine: Reading<EngineReadOutput>): EngineLine {
    const reach = useReach();
    if (reach.kind === 'unanswered') {
        return { dot: ownState('engine-absent'), opens: false, said: reach.said, version: '' };
    }
    if (reach.kind === 'connecting') {
        const dot = ownState('connecting');
        return { dot, opens: false, said: `Engine ${dot.word}`, version: '' };
    }
    if (engine.kind === 'failed') {
        return {
            dot: failedState(engine.error),
            opens: true,
            said: engine.error.message,
            version: '',
        };
    }
    if (engine.kind === 'reading') {
        return { dot: ownState('reading'), opens: true, said: 'Engine', version: '' };
    }
    const { Version } = engine.out;
    const dot = engineState(engine.out);
    const said = Version === '' ? `Engine ${dot.word}` : `Engine ${dot.word}${SEPARATOR}${Version}`;
    return { dot, opens: true, said, version: Version };
}

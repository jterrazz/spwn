import type { EngineError } from './engine-error';
import { engineShown, ownState } from './shown';
import type { Shown } from './shown';

export function failedState({ wire }: EngineError): Shown {
    if (!wire) {
        return ownState('no-answer');
    }
    return wire.type === undefined ? ownState('untyped') : engineShown('CoreFailType', wire.type);
}

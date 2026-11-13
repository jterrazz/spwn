import type { EngineError } from './engine-error';
import { ownState } from './shown';
import type { Shown } from './shown';
import { ERROR_TYPE_TONES } from './tones';
import { ERROR_TYPE_WORDS } from './words';

export function failedState({ wire }: EngineError): Shown {
    if (!wire) {
        return ownState('no-answer');
    }
    return { tone: ERROR_TYPE_TONES[wire.type ?? ''], word: ERROR_TYPE_WORDS[wire.type ?? ''] };
}

import type { CoreCallWireError } from '@jterrazz/station-api';

// value: answered, the engine's own error; unsent, the request reached no engine; lost, it was sent and its answer
// value: never came (the connection dropped, or the call ran past its time), so the engine may have done it.
export type Fate = 'answered' | 'lost' | 'unsent';

export class EngineError extends Error {
    readonly wire: CoreCallWireError | undefined;
    readonly fate: Fate;

    constructor(
        wire: CoreCallWireError | undefined,
        reason = wire?.message ?? 'no answer',
        fate: Fate = wire ? 'answered' : 'unsent',
    ) {
        super(reason);
        this.name = 'EngineError';
        this.wire = wire;
        this.fate = fate;
    }
}

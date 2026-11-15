import type { CoreCallWireError } from '@jterrazz/station-api';

// value: answered, the engine's own error; unsent, the request reached no engine; lost, it was sent and its answer
// value: never came (the connection dropped, or the call ran past its time), so the engine may have done it.
export type Fate = 'answered' | 'lost' | 'unsent';

// value: the refusals the window makes of its own, its shell's (`src-tauri/src/opener.rs`) and its opener's outside
// value: Tauri: drawn like the engine's, never on the engine's wire, so never one of its codes.
export const WINDOW_FAIL_CODES = [
    'opener.not-a-web-address',
    'opener.not-one-line',
    'opener.not-opened',
    'opener.outside-tauri',
] as const;

export type WindowFailCode = (typeof WINDOW_FAIL_CODES)[number];

export type WindowRefusal = Omit<CoreCallWireError, 'code'> & { code: WindowFailCode };

export type Refusal = CoreCallWireError | WindowRefusal;

export class EngineError extends Error {
    readonly wire: Refusal | undefined;
    readonly fate: Fate;

    constructor(
        wire: Refusal | undefined,
        reason = wire?.message ?? 'no answer',
        fate: Fate = wire ? 'answered' : 'unsent',
    ) {
        super(reason);
        this.name = 'EngineError';
        this.wire = wire;
        this.fate = fate;
    }
}

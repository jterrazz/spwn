import type { OpenerPort, Opens } from '../../../application/ports/opener.port';
import { EngineError, WINDOW_FAIL_CODES } from '../../../domain/engine-error';
import type { WindowFailCode } from '../../../domain/engine-error';
import { isRecord } from '../../../domain/request-key';
import type { Ipc } from '../engine-tauri/tauri';

const COMMANDS = {
    terminal: { argument: 'command', command: 'open_in_terminal' },
    url: { argument: 'url', command: 'open_url' },
} as const satisfies Record<Opens, { argument: string; command: string }>;

function windowCodeOf(code: unknown): undefined | WindowFailCode {
    return WINDOW_FAIL_CODES.find((one) => one === code);
}

function refusalOf(error: unknown): EngineError {
    const code = isRecord(error) ? windowCodeOf(error.code) : undefined;
    if (isRecord(error) && code !== undefined && typeof error.message === 'string') {
        return new EngineError({
            code,
            message: error.message,
            subject: 'window',
            type: 'refused',
        });
    }
    return new EngineError({ message: String(error), subject: 'window', type: 'failed' });
}

// invariant: the window's two Tauri commands, each handed the engine's value and nothing else; whether it is a web
// invariant: address or one line is the Rust side's to check (`src-tauri/src/opener.rs`), and a refusal comes back
// invariant: drawn like the engine's.
export function tauriOpener(ipc: Pick<Ipc, 'invoke'>): OpenerPort {
    return async (opening) => {
        const { argument, command } = COMMANDS[opening.opens];
        try {
            await ipc.invoke(command, { [argument]: opening.value });
            return { kind: 'opened' };
        } catch (error) {
            return { error: refusalOf(error), kind: 'refused' };
        }
    };
}

export const NO_OPENER: OpenerPort = async () =>
    await Promise.resolve({
        error: new EngineError({
            code: 'opener.outside-tauri',
            message: 'The window opens a browser or a terminal from its Tauri shell alone',
            subject: 'window',
            type: 'refused',
        }),
        kind: 'refused',
    });

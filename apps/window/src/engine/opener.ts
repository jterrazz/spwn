import type { AccountsAccount, MachinesSSHTarget, PortsSharedPort } from '@jterrazz/station-api';

import { EngineError, WINDOW_FAIL_CODES } from './engine-error';
import type { WindowFailCode } from './engine-error';
import { isRecord } from './frame';
import { shareOpensFromHere } from './provisional';
import type { Ipc } from './tauri';

type Opens = 'terminal' | 'url';

// invariant: an Openable is made from an engine answer alone, by the three functions below: its constructor is
// invariant: private, so no text a page holds can reach a terminal or a browser.
export class Openable {
    readonly opens: Opens;
    readonly value: string;

    private constructor(opens: Opens, value: string) {
        this.opens = opens;
        this.value = value;
    }

    static ofSsh(target: MachinesSSHTarget): Openable | undefined {
        return target.Command === '' ? undefined : new Openable('terminal', target.Command);
    }

    static ofFix(account: AccountsAccount): Openable | undefined {
        const hint = account.hint ?? '';
        return hint === '' ? undefined : new Openable('terminal', hint);
    }

    static ofShare(share: PortsSharedPort, onThisMac: boolean): Openable | undefined {
        const url = shareOpensFromHere(share, onThisMac);
        return share.http && url !== '' ? new Openable('url', url) : undefined;
    }
}

export function shareAddressOf(share: PortsSharedPort): string {
    return share.urls.public === '' ? share.urls.local : share.urls.public;
}

export type Opened = { kind: 'opened' } | { kind: 'refused'; error: EngineError };

export type Opener = (openable: Openable) => Promise<Opened>;

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
export function tauriOpener(ipc: Pick<Ipc, 'invoke'>): Opener {
    return async (openable) => {
        const { argument, command } = COMMANDS[openable.opens];
        try {
            await ipc.invoke(command, { [argument]: openable.value });
            return { kind: 'opened' };
        } catch (error) {
            return { error: refusalOf(error), kind: 'refused' };
        }
    };
}

export const NO_OPENER: Opener = async () =>
    await Promise.resolve({
        error: new EngineError({
            code: 'opener.outside-tauri',
            message: 'The window opens a browser or a terminal from its Tauri shell alone',
            subject: 'window',
            type: 'refused',
        }),
        kind: 'refused',
    });

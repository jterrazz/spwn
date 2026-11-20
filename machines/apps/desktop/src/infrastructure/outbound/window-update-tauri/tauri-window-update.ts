import type {
    HeardProgress,
    WindowUpdatePort,
} from '../../../application/ports/window-update.port';
import { isRecord } from '../../../domain/request-key';
import { isWindowUpdateCode } from '../../../domain/window-update';
import type {
    WindowChecked,
    WindowInstalled,
    WindowProgress,
    WindowRefused,
    WindowUpdateCode,
} from '../../../domain/window-update';
import type { Ipc } from '../engine-tauri/tauri';

type UpdateIpc = Pick<Ipc, 'channel' | 'invoke'>;

const UNREAD = 'the shell answered in a shape the window does not read';

function refusalOf(error: unknown, otherwise: WindowUpdateCode): WindowRefused {
    if (isRecord(error) && isWindowUpdateCode(error.code) && typeof error.message === 'string') {
        return { code: error.code, kind: 'refused', message: error.message };
    }
    const message =
        isRecord(error) && typeof error.message === 'string' ? error.message : String(error);
    return { code: otherwise, kind: 'refused', message };
}

function textOf(value: unknown): string | undefined {
    return typeof value === 'string' && value !== '' ? value : undefined;
}

function checkedOf(answer: unknown): WindowChecked | WindowRefused {
    if (!isRecord(answer)) {
        return refusalOf(UNREAD, 'window.update-unreachable');
    }
    const current = textOf(answer.current);
    const version = textOf(answer.version);
    const date = textOf(answer.date);
    if (answer.kind === 'unsupported') {
        return { kind: 'unsupported' };
    }
    if (answer.kind === 'current' && current !== undefined) {
        return { current, kind: 'current' };
    }
    if (answer.kind === 'out' && current !== undefined && version !== undefined) {
        return date === undefined
            ? { current, kind: 'out', version }
            : { current, date, kind: 'out', version };
    }
    return refusalOf(UNREAD, 'window.update-unreachable');
}

function counted(value: unknown): number | undefined {
    return typeof value === 'number' && Number.isFinite(value) && value >= 0 ? value : undefined;
}

function progressOf(message: unknown): undefined | WindowProgress {
    if (!isRecord(message)) {
        return undefined;
    }
    if (message.installed === true) {
        return { kind: 'installed' };
    }
    const downloaded = counted(message.downloaded);
    const total = counted(message.total);
    if (downloaded === undefined) {
        return undefined;
    }
    return total === undefined
        ? { downloaded, kind: 'downloaded' }
        : { downloaded, kind: 'downloaded', total };
}

async function installed(
    ipc: UpdateIpc,
    heard: HeardProgress,
): Promise<WindowInstalled | WindowRefused> {
    const progress = ipc.channel((message: unknown) => {
        const step = progressOf(message);
        if (step) {
            heard(step);
        }
    });
    try {
        await ipc.invoke('window_update_install', { progress });
        return { kind: 'installed' };
    } catch (error: unknown) {
        return refusalOf(error, 'window.update-install');
    }
}

export function tauriWindowUpdate(ipc: UpdateIpc): WindowUpdatePort {
    return {
        check: async () => {
            try {
                return checkedOf(await ipc.invoke('window_update_check'));
            } catch (error: unknown) {
                return refusalOf(error, 'window.update-unreachable');
            }
        },
        install: async (heard) => await installed(ipc, heard),
    };
}

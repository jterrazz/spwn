import type { MenuBarPort } from '../../../application/ports/menu-bar.port';
import { isMenuBarCode } from '../../../domain/menu-bar';
import type { MenuBarCode, MenuBarRead, MenuBarRefused } from '../../../domain/menu-bar';
import { isRecord } from '../../../domain/request-key';
import type { Ipc } from '../engine-tauri/tauri';

type MenuBarIpc = Pick<Ipc, 'invoke'>;

const UNREAD = 'the shell answered in a shape the window does not read';
const KINDS = ['approval', 'hidden', 'shown', 'unsupported'] as const;

function refusalOf(error: unknown, otherwise: MenuBarCode): MenuBarRefused {
    if (isRecord(error) && isMenuBarCode(error.code) && typeof error.message === 'string') {
        return { code: error.code, kind: 'refused', message: error.message };
    }
    const message =
        isRecord(error) && typeof error.message === 'string' ? error.message : String(error);
    return { code: otherwise, kind: 'refused', message };
}

function readOf(answer: unknown, otherwise: MenuBarCode): MenuBarRead | MenuBarRefused {
    const kind = KINDS.find((known) => isRecord(answer) && answer.kind === known);
    return kind === undefined ? refusalOf(UNREAD, otherwise) : { kind };
}

async function asked(
    ipc: MenuBarIpc,
    otherwise: MenuBarCode,
    command: string,
    args?: Record<string, unknown>,
): Promise<MenuBarRead | MenuBarRefused> {
    try {
        return readOf(await ipc.invoke(command, args), otherwise);
    } catch (error: unknown) {
        return refusalOf(error, otherwise);
    }
}

export function tauriMenuBar(ipc: MenuBarIpc): MenuBarPort {
    return {
        read: async () => await asked(ipc, 'menubar.not-read', 'menu_bar_read'),
        show: async (shown) => await asked(ipc, 'menubar.not-switched', 'menu_bar_show', { shown }),
    };
}

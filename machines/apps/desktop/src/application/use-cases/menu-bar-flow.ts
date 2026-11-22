import type { MenuBarCode, MenuBarRead, MenuBarRefused } from '../../domain/menu-bar';
import type { MenuBarPort } from '../ports/menu-bar.port';

type Held = Exclude<MenuBarRead, { kind: 'unsupported' }>['kind'];

export type MenuBarState =
    | { kind: 'failed'; held: Held; code: MenuBarCode; message: string }
    | { kind: 'held'; held: Held }
    | { kind: 'refused'; code: MenuBarCode; message: string }
    | { kind: 'switching'; held: Held; to: boolean }
    | { kind: 'unknown' }
    | { kind: 'unsupported' };

export const NO_MENU_BAR: MenuBarPort = {
    read: async () => await Promise.resolve({ kind: 'unsupported' }),
    show: async () => await Promise.resolve({ kind: 'unsupported' }),
};

function refusedOf(code: MenuBarCode, error: unknown): MenuBarRefused {
    return { code, kind: 'refused', message: String(error) };
}

function heldOf(state: MenuBarState): Held | undefined {
    return 'held' in state ? state.held : undefined;
}

// invariant: the switch shows what macOS holds, read each time Settings is drawn and again after each press, never
// invariant: what the press asked: a press that needs the owner's approval reads back as `approval`.
export class MenuBarFlow {
    private current: MenuBarState = { kind: 'unknown' };
    private readonly listeners = new Set<() => void>();
    private readonly port: MenuBarPort;

    constructor(port: MenuBarPort) {
        this.port = port;
    }

    readonly state = (): MenuBarState => this.current;

    readonly subscribe = (listener: () => void): (() => void) => {
        this.listeners.add(listener);
        return () => {
            this.listeners.delete(listener);
        };
    };

    look(): void {
        if (this.current.kind !== 'switching') {
            void this.answered(this.port.read(), 'menubar.not-read', undefined);
        }
    }

    show(shown: boolean): void {
        const held = heldOf(this.current);
        if (held === undefined || this.current.kind === 'switching') {
            return;
        }
        this.tell({ held, kind: 'switching', to: shown });
        void this.answered(this.port.show(shown), 'menubar.not-switched', held);
    }

    private tell(next: MenuBarState): void {
        this.current = next;
        for (const listener of this.listeners) {
            listener();
        }
    }

    private async answered(
        asked: Promise<MenuBarRead | MenuBarRefused>,
        code: MenuBarCode,
        held: Held | undefined,
    ): Promise<void> {
        const answer = await asked.catch((error: unknown) => refusedOf(code, error));
        if (answer.kind === 'unsupported') {
            this.tell({ kind: 'unsupported' });
            return;
        }
        if (answer.kind !== 'refused') {
            this.tell({ held: answer.kind, kind: 'held' });
            return;
        }
        this.tell(
            held === undefined
                ? answer
                : { code: answer.code, held, kind: 'failed', message: answer.message },
        );
    }
}

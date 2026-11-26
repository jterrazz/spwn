// value: where New machine was opened from: a Mac's page names that Mac, a machine file's page that file.
export type Opening = { on?: string | undefined; from?: string | undefined };

export type Sheet =
    | { kind: 'add-mac' }
    | { kind: 'add-password' }
    | { kind: 'find' }
    | { kind: 'new-machine'; opening: Opening }
    | { kind: 'none' };

const NONE: Sheet = { kind: 'none' };

// invariant: the sheet the window shows, held apart from the frame: opening or closing one tells only who reads it,
// invariant: so the page, the path bar and the sidebar beneath it are not drawn again. Every opener is one function
// invariant: for the window's life.
export class Opened {
    private sheet: Sheet = NONE;
    private readonly listeners = new Set<() => void>();

    readonly subscribe = (listener: () => void): (() => void) => {
        this.listeners.add(listener);
        return () => {
            this.listeners.delete(listener);
        };
    };

    readonly shown = (): Sheet => this.sheet;

    readonly addMac = (): void => {
        this.open({ kind: 'add-mac' });
    };

    readonly addPassword = (): void => {
        this.open({ kind: 'add-password' });
    };

    readonly close = (): void => {
        this.open(NONE);
    };

    readonly find = (): void => {
        this.open({ kind: 'find' });
    };

    readonly newBlank = (): void => {
        this.open({ kind: 'new-machine', opening: {} });
    };

    readonly newMachine = (opening: Opening = {}): void => {
        this.open({ kind: 'new-machine', opening });
    };

    private open(sheet: Sheet): void {
        if (sheet === this.sheet) {
            return;
        }
        this.sheet = sheet;
        for (const listener of this.listeners) {
            listener();
        }
    }
}

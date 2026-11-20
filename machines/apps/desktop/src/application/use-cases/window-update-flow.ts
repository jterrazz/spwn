import type {
    WindowChecked,
    WindowInstalled,
    WindowProgress,
    WindowRefused,
    WindowUpdateCode,
} from '../../domain/window-update';
import type { WindowUpdatePort } from '../ports/window-update.port';

export type WindowRelease = { current: string; version: string; date?: string };

export type WindowUpdateState =
    | (WindowRelease & { kind: 'failed'; code: WindowUpdateCode; message: string })
    | (WindowRelease & { kind: 'installing'; downloaded: number; total?: number })
    | (WindowRelease & { kind: 'out' })
    | (WindowRelease & { kind: 'relaunching' })
    | { kind: 'checking'; first: boolean }
    | { kind: 'current'; current: string }
    | { kind: 'refused'; code: WindowUpdateCode; message: string }
    | { kind: 'unknown' }
    | { kind: 'unsupported' };

export const NO_WINDOW_UPDATE: WindowUpdatePort = {
    check: async () => await Promise.resolve({ kind: 'unsupported' }),
    install: async () =>
        await Promise.resolve({
            code: 'window.update-gone',
            kind: 'refused',
            message: 'The window updates itself from its Tauri shell alone',
        }),
};

function refusedOf(code: WindowUpdateCode, error: unknown): WindowRefused {
    return { code, kind: 'refused', message: String(error) };
}

function releaseOf(state: WindowUpdateState): undefined | WindowRelease {
    if (!('version' in state)) {
        return undefined;
    }
    const { current, date, version } = state;
    return date === undefined ? { current, version } : { current, date, version };
}

// invariant: the window checks its own release once, when it starts, and again only on Check now: no timer. An
// invariant: install is pressed, never begun alone, and the press is the consent to relaunch.
export class WindowUpdateFlow {
    private current: WindowUpdateState = { kind: 'unknown' };
    private readonly listeners = new Set<() => void>();
    private readonly port: WindowUpdatePort;

    constructor(port: WindowUpdatePort) {
        this.port = port;
    }

    readonly state = (): WindowUpdateState => this.current;

    readonly subscribe = (listener: () => void): (() => void) => {
        this.listeners.add(listener);
        return () => {
            this.listeners.delete(listener);
        };
    };

    start(): void {
        if (this.current.kind === 'unknown') {
            this.checked(true);
        }
    }

    checkNow(): void {
        const { kind } = this.current;
        if (kind === 'current' || kind === 'out' || kind === 'refused' || kind === 'failed') {
            this.checked(false);
        }
    }

    install(): void {
        const release = releaseOf(this.current);
        if (
            release === undefined ||
            (this.current.kind !== 'out' && this.current.kind !== 'failed')
        ) {
            return;
        }
        this.show({ ...release, downloaded: 0, kind: 'installing' });
        void this.installed(release);
    }

    retry(): void {
        if (this.current.kind === 'failed') {
            this.install();
        }
    }

    private show(next: WindowUpdateState): void {
        this.current = next;
        for (const listener of this.listeners) {
            listener();
        }
    }

    private checked(first: boolean): void {
        this.show({ first, kind: 'checking' });
        void this.answered();
    }

    private async answered(): Promise<void> {
        const answer: WindowChecked | WindowRefused = await this.port
            .check()
            .catch((error: unknown) => refusedOf('window.update-unreachable', error));
        this.show(answer);
    }

    private readonly heard = (progress: WindowProgress): void => {
        const release = releaseOf(this.current);
        if (this.current.kind !== 'installing' || release === undefined) {
            return;
        }
        if (progress.kind === 'installed') {
            this.show({ ...release, kind: 'relaunching' });
            return;
        }
        const { downloaded, total } = progress;
        this.show(
            total === undefined
                ? { ...release, downloaded, kind: 'installing' }
                : { ...release, downloaded, kind: 'installing', total },
        );
    };

    private async installed(release: WindowRelease): Promise<void> {
        const end: WindowInstalled | WindowRefused = await this.port
            .install(this.heard)
            .catch((error: unknown) => refusedOf('window.update-install', error));
        if (end.kind === 'refused') {
            this.show({ ...release, code: end.code, kind: 'failed', message: end.message });
            return;
        }
        this.show({ ...release, kind: 'relaunching' });
    }
}

import type { CoreEnd, CoreLine, CorePort, CoreRun } from '../ports/core.port';

export const CORE_LOOK_EVERY = 3000;

const KEPT_LINES = 400;

export type CoreAct = 'install' | 'start';

export type CoreState =
    | { kind: 'failed'; of: CoreAct; lines: readonly CoreLine[]; said: string; code?: number }
    | { kind: 'installed'; later: boolean }
    | { kind: 'installing'; lines: readonly CoreLine[] }
    | { kind: 'looking' }
    | { kind: 'missing' }
    | { kind: 'started'; lines: readonly CoreLine[] }
    | { kind: 'starting'; lines: readonly CoreLine[] }
    | { kind: 'waiting' };

export type CoreKind = CoreState['kind'];

function refusedRun(message: string): CoreRun {
    return {
        cancel: () => undefined,
        ended: Promise.resolve({ kind: 'refused', message }),
    };
}

export const NO_CORE: CorePort = {
    install: () => refusedRun('The window installs Spwn from its Tauri shell alone'),
    present: async () => await Promise.resolve(false),
    start: () => refusedRun('The window starts the engine from its Tauri shell alone'),
};

function linesOf(state: CoreState): readonly CoreLine[] {
    return 'lines' in state ? state.lines : [];
}

export function saidOf(lines: readonly CoreLine[]): string {
    const spoken = lines.filter((line) => line.text.trim() !== '');
    const errors = spoken.filter((line) => line.stream === 'err');
    return (errors.at(-1) ?? spoken.at(-1))?.text ?? '';
}

const LOOKS_AGAIN: ReadonlySet<CoreKind> = new Set(['installed', 'looking', 'missing']);

export class CoreFlow {
    private current: CoreState = { kind: 'looking' };
    private readonly listeners = new Set<() => void>();
    private readonly core: CorePort;
    private readonly lookEvery: number;
    private run: CoreRun | undefined;
    private looker: ReturnType<typeof setInterval> | undefined;

    constructor(core: CorePort, lookEvery = CORE_LOOK_EVERY) {
        this.core = core;
        this.lookEvery = lookEvery;
    }

    readonly state = (): CoreState => this.current;

    readonly subscribe = (listener: () => void): (() => void) => {
        this.listeners.add(listener);
        return () => {
            this.listeners.delete(listener);
        };
    };

    enter(): void {
        if (LOOKS_AGAIN.has(this.current.kind)) {
            void this.look();
        }
    }

    answered(): void {
        if (this.run !== undefined) {
            return;
        }
        this.stopLooking();
        this.show({ kind: 'looking' });
    }

    close(): void {
        this.stopLooking();
    }

    install(): void {
        this.stopLooking();
        this.running('install', { kind: 'installing', lines: [] });
    }

    start(): void {
        this.running('start', { kind: 'starting', lines: [] });
    }

    retry(): void {
        if (this.current.kind === 'failed' && this.current.of === 'start') {
            this.start();
        } else {
            this.install();
        }
    }

    cancel(): void {
        this.run?.cancel();
    }

    later(): void {
        if (this.current.kind === 'installed') {
            this.show({ kind: 'installed', later: true });
        }
    }

    waitForTerminal(): void {
        if (this.run !== undefined) {
            return;
        }
        this.show({ kind: 'waiting' });
        this.stopLooking();
        this.looker = setInterval(() => {
            void this.lookWhileWaiting();
        }, this.lookEvery);
    }

    installHere(): void {
        this.stopLooking();
        this.show({ kind: 'missing' });
    }

    private show(next: CoreState): void {
        this.current = next;
        for (const listener of this.listeners) {
            listener();
        }
    }

    private stopLooking(): void {
        if (this.looker !== undefined) {
            clearInterval(this.looker);
            this.looker = undefined;
        }
    }

    private async present(): Promise<boolean> {
        return await this.core.present().catch(() => false);
    }

    private async look(): Promise<void> {
        const asked = this.current;
        const here = await this.present();
        if (this.current === asked) {
            this.show(here ? { kind: 'installed', later: false } : { kind: 'missing' });
        }
    }

    private async lookWhileWaiting(): Promise<void> {
        const here = await this.present();
        if (here && this.current.kind === 'waiting') {
            this.stopLooking();
            this.show({ kind: 'installed', later: false });
        }
    }

    private heard(line: CoreLine): void {
        const { current } = this;
        if (current.kind === 'installing' || current.kind === 'starting') {
            this.show({ ...current, lines: [...current.lines, line].slice(-KEPT_LINES) });
        }
    }

    private running(act: CoreAct, first: CoreState): void {
        if (this.run !== undefined) {
            return;
        }
        this.show(first);
        const heard = (line: CoreLine) => {
            this.heard(line);
        };
        const run = act === 'install' ? this.core.install(heard) : this.core.start(heard);
        this.run = run;
        void this.awaited(act, run);
    }

    private async awaited(act: CoreAct, run: CoreRun): Promise<void> {
        let end: CoreEnd;
        try {
            end = await run.ended;
        } catch (error: unknown) {
            end = { kind: 'refused', message: String(error) };
        }
        this.run = undefined;
        this.ended(act, end);
    }

    private ended(act: CoreAct, end: CoreEnd): void {
        const lines = linesOf(this.current);
        if (end.kind === 'refused') {
            this.show({ kind: 'failed', lines, of: act, said: end.message });
            return;
        }
        if (end.cancelled) {
            this.show(
                act === 'install' ? { kind: 'missing' } : { kind: 'installed', later: false },
            );
            return;
        }
        if (end.code !== 0) {
            this.show({ code: end.code, kind: 'failed', lines, of: act, said: saidOf(lines) });
            return;
        }
        this.show(
            act === 'install' ? { kind: 'installed', later: false } : { kind: 'started', lines },
        );
    }
}

import type { CoreEnd, CoreLine, CorePort, CoreRun } from '../ports/core.port';
import { saidOf } from './core-flow';

const KEPT_LINES = 400;

export type UpdateStep = 'install' | 'start';

export type UpdateLines = {
    readonly install: readonly CoreLine[];
    readonly start: readonly CoreLine[];
};

export type UpdateState =
    | {
          kind: 'failed';
          step: UpdateStep;
          to: string;
          lines: UpdateLines;
          said: string;
          code?: number;
      }
    | { kind: 'idle' }
    | { kind: 'restarting'; to: string; lines: UpdateLines }
    | { kind: 'updated'; to: string; lines: UpdateLines }
    | { kind: 'updating'; to: string; lines: UpdateLines };

export type UpdateKind = UpdateState['kind'];

export type UpdateAsked = { to: string; executable: string | undefined };

const NO_LINES: UpdateLines = { install: [], start: [] };

function linesOf(state: UpdateState): UpdateLines {
    return 'lines' in state ? state.lines : NO_LINES;
}

function stepOf(state: UpdateState): UpdateStep {
    return state.kind === 'restarting' ? 'start' : 'install';
}

async function endOf(run: CoreRun): Promise<CoreEnd> {
    try {
        return await run.ended;
    } catch (error: unknown) {
        return { kind: 'refused', message: String(error) };
    }
}

// invariant: one press runs the public installer, then `station engine start` on the station the engine named: the
// invariant: owner's Update is the consent to restart the engine, and nothing runs again until the next press.
export class UpdateFlow {
    private current: UpdateState = { kind: 'idle' };
    private readonly listeners = new Set<() => void>();
    private readonly core: CorePort;
    private run: CoreRun | undefined;
    private asked: undefined | UpdateAsked;

    constructor(core: CorePort) {
        this.core = core;
    }

    readonly state = (): UpdateState => this.current;

    readonly subscribe = (listener: () => void): (() => void) => {
        this.listeners.add(listener);
        return () => {
            this.listeners.delete(listener);
        };
    };

    update(asked: UpdateAsked): void {
        if (this.run !== undefined) {
            return;
        }
        this.asked = asked;
        this.show({ kind: 'updating', lines: NO_LINES, to: asked.to });
        this.running(this.core.install(this.heard));
    }

    retry(): void {
        if (this.asked !== undefined && this.current.kind === 'failed') {
            this.update(this.asked);
        }
    }

    cancel(): void {
        if (this.current.kind === 'updating') {
            this.run?.cancel();
        }
    }

    done(): void {
        if (this.run === undefined) {
            this.show({ kind: 'idle' });
        }
    }

    private show(next: UpdateState): void {
        this.current = next;
        for (const listener of this.listeners) {
            listener();
        }
    }

    private readonly heard = (line: CoreLine): void => {
        const { current } = this;
        if (current.kind !== 'updating' && current.kind !== 'restarting') {
            return;
        }
        const step = stepOf(current);
        const lines = {
            ...current.lines,
            [step]: [...current.lines[step], line].slice(-KEPT_LINES),
        };
        this.show({ ...current, lines });
    };

    private running(run: CoreRun): void {
        this.run = run;
        void this.awaited(run);
    }

    private async awaited(run: CoreRun): Promise<void> {
        const end = await endOf(run);
        this.run = undefined;
        this.ended(end);
    }

    private ended(end: CoreEnd): void {
        const { current } = this;
        if (current.kind !== 'updating' && current.kind !== 'restarting') {
            return;
        }
        const step = stepOf(current);
        const lines = linesOf(current);
        const { to } = current;
        if (end.kind === 'refused') {
            this.show({ kind: 'failed', lines, said: end.message, step, to });
        } else if (end.cancelled) {
            this.show({ kind: 'idle' });
        } else if (end.code !== 0) {
            this.show({
                code: end.code,
                kind: 'failed',
                lines,
                said: saidOf(lines[step]),
                step,
                to,
            });
        } else if (step === 'install') {
            this.show({ kind: 'restarting', lines, to });
            this.running(this.core.start(this.heard, this.asked?.executable));
        } else {
            this.show({ kind: 'updated', lines, to });
        }
    }
}

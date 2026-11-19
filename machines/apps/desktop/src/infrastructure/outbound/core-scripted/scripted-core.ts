import type {
    CoreEnd,
    CoreLine,
    CorePort,
    CoreRun,
    Heard,
} from '../../../application/ports/core.port';

export type ScriptedRun = {
    lines: readonly CoreLine[];
    code: number;
    holds?: boolean;
};

// value: `answering` scripts a core whose engine answers from the start: an update, not a first install.
export type CoreScript = {
    present: boolean;
    answering?: boolean;
    appearsAt?: number;
    install: ScriptedRun;
    start: ScriptedRun;
};

export type Scripted = {
    gap?: number;
    onStarted?: () => void;
};

const CANCELLED = 143;

const UNSETTLED = (_end: CoreEnd): void => undefined;

function played(
    run: ScriptedRun,
    heard: Heard,
    gap: number,
    onEnd: (end: CoreEnd) => void,
): CoreRun {
    const timers: ReturnType<typeof setTimeout>[] = [];
    let settle: (end: CoreEnd) => void = UNSETTLED;
    const ended = new Promise<CoreEnd>((resolve) => {
        settle = (end) => {
            for (const timer of timers) {
                clearTimeout(timer);
            }
            onEnd(end);
            resolve(end);
        };
    });
    for (const [index, line] of run.lines.entries()) {
        timers.push(
            setTimeout(() => {
                heard(line);
            }, index * gap),
        );
    }
    if (run.holds !== true) {
        timers.push(
            setTimeout(() => {
                settle({ cancelled: false, code: run.code, kind: 'ended' });
            }, run.lines.length * gap),
        );
    }
    return {
        cancel: () => {
            settle({ cancelled: true, code: CANCELLED, kind: 'ended' });
        },
        ended,
    };
}

export function scriptedCore(script: CoreScript, { gap = 0, onStarted }: Scripted = {}): CorePort {
    let installed = script.present;
    let looks = 0;
    return {
        install: (heard) =>
            played(script.install, heard, gap, (end) => {
                installed ||= end.kind === 'ended' && !end.cancelled && end.code === 0;
            }),
        present: async () => {
            looks += 1;
            installed ||= script.appearsAt !== undefined && looks >= script.appearsAt;
            return await Promise.resolve(installed);
        },
        start: (heard) =>
            played(script.start, heard, gap, (end) => {
                if (end.kind === 'ended' && !end.cancelled && end.code === 0) {
                    onStarted?.();
                }
            }),
    };
}

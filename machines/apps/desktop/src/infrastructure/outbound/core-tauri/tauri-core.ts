import type {
    CoreEnd,
    CoreLine,
    CorePort,
    CoreRun,
    Heard,
} from '../../../application/ports/core.port';
import { isRecord } from '../../../domain/request-key';
import type { Ipc } from '../engine-tauri/tauri';

type CoreIpc = Pick<Ipc, 'channel' | 'invoke'>;

const LAST_LINES_WITHIN = 1000;

function lineOf(message: unknown): CoreLine | undefined {
    if (!isRecord(message) || typeof message.text !== 'string') {
        return undefined;
    }
    if (message.stream === 'out' || message.stream === 'err') {
        return { stream: message.stream, text: message.text };
    }
    return undefined;
}

type Ended = { end: CoreEnd; lines: number };

function endOf(value: unknown): Ended {
    if (isRecord(value) && typeof value.code === 'number') {
        return {
            end: { cancelled: value.cancelled === true, code: value.code, kind: 'ended' },
            lines: typeof value.lines === 'number' ? value.lines : 0,
        };
    }
    return { end: { kind: 'refused', message: 'The window heard no end from its run' }, lines: 0 };
}

function refusalOf(error: unknown): Ended {
    const message =
        isRecord(error) && typeof error.message === 'string' ? error.message : String(error);
    return { end: { kind: 'refused', message }, lines: 0 };
}

class Counted {
    private count = 0;
    private waiting: undefined | { want: number; done: () => void };
    private readonly heard: Heard;

    constructor(heard: Heard) {
        this.heard = heard;
    }

    line(line: CoreLine): void {
        this.count += 1;
        this.heard(line);
        if (this.waiting !== undefined && this.count >= this.waiting.want) {
            this.waiting.done();
        }
    }

    async all(want: number): Promise<void> {
        if (this.count >= want) {
            return;
        }
        await new Promise<void>((resolve) => {
            this.waiting = { done: resolve, want };
            setTimeout(resolve, LAST_LINES_WITHIN);
        });
    }
}

async function cancelled(ipc: CoreIpc): Promise<void> {
    try {
        await ipc.invoke('core_cancel');
    } catch {
        return;
    }
}

type Args = { executable?: string };

function ran(ipc: CoreIpc, command: string, heard: Heard, args: Args = {}): CoreRun {
    const counted = new Counted(heard);
    const channel = ipc.channel((message: unknown) => {
        const line = lineOf(message);
        if (line) {
            counted.line(line);
        }
    });
    const ended = async (): Promise<CoreEnd> => {
        const { end, lines } = await ipc
            .invoke(command, { ...args, channel })
            .then(endOf, refusalOf);
        await counted.all(lines);
        return end;
    };
    return {
        cancel: () => {
            void cancelled(ipc);
        },
        ended: ended(),
    };
}

export function tauriCore(ipc: CoreIpc): CorePort {
    return {
        install: (heard) => ran(ipc, 'core_install', heard),
        present: async () => (await ipc.invoke('core_present')) === true,
        start: (heard, executable) =>
            ran(ipc, 'core_start', heard, executable === undefined ? {} : { executable }),
    };
}

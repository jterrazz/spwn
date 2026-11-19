export type CoreStream = 'err' | 'out';

export type CoreLine = { readonly stream: CoreStream; readonly text: string };

export type CoreEnd =
    | { kind: 'ended'; code: number; cancelled: boolean }
    | { kind: 'refused'; message: string };

export type CoreRun = { ended: Promise<CoreEnd>; cancel: () => void };

export type Heard = (line: CoreLine) => void;

export interface CorePort {
    present: () => Promise<boolean>;
    install: (heard: Heard) => CoreRun;
    start: (heard: Heard) => CoreRun;
}

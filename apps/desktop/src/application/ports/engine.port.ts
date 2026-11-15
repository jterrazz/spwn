import type {
    CallName,
    Calls,
    CoreCallWireError,
    EngineHello,
    Progress,
    Target,
} from '@jterrazz/station-api';

import type { Bridged } from '../../domain/bridged';
import type { EngineError } from '../../domain/engine-error';

export type Watched = {
    [N in CallName]: Calls[N]['watch'] extends true ? N : never;
}[CallName];

export type Called = {
    [N in CallName]: Calls[N]['watch'] extends false ? N : never;
}[CallName];

export type WatchName = Extract<Watched, Bridged>;

export type ReadName = Extract<Called, Bridged>;

export type Out<N extends CallName> = Calls[N]['out'];

export type In<N extends CallName> = Calls[N]['in'];

export type WatchHandlers<N extends Watched> = {
    out: (out: Out<N>) => void;
    error: (error: EngineError) => void;
    done?: () => void;
};

export type Watch = { close: () => void };

export type CallOptions = {
    target?: Target | undefined;
    progress?: (progress: Progress) => void;
    within?: number;
};

export type Link =
    | { kind: 'absent'; socket: string; reason: string; retryIn: number }
    | { kind: 'connecting'; socket: string }
    | {
          kind: 'open';
          socket: string;
          hello: EngineHello | undefined;
          refused: CoreCallWireError | undefined;
      };

export interface EnginePort {
    call: <N extends Called>(name: N, input: In<N>, options?: CallOptions) => Promise<Out<N>>;
    watch: <N extends Watched>(
        name: N,
        input: In<N>,
        handlers: WatchHandlers<N>,
        target?: Target,
    ) => Watch;
    link: (listener: (link: Link) => void) => () => void;
}

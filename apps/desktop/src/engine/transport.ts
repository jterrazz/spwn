import type { CancelFrame, CoreCallWireError, EngineHello, Request } from '@jterrazz/station-api';

export type Link =
    | { kind: 'absent'; socket: string; reason: string; retryIn: number }
    | { kind: 'connecting'; socket: string }
    | {
          kind: 'open';
          socket: string;
          hello: EngineHello | undefined;
          refused: CoreCallWireError | undefined;
      };

// invariant: a cancel is sent on the exchange whose request it names, before that exchange closes.
export type Exchange = { cancel: (frame: CancelFrame) => void; close: () => void };

export type Receiver = {
    frame: (frame: unknown) => void;
    // value: sent says the request reached the engine before the exchange ended.
    end: (reason: string, sent: boolean) => void;
};

export type Transport = {
    open: (request: Request, receiver: Receiver) => Exchange;
    link: (listener: (link: Link) => void) => () => void;
};

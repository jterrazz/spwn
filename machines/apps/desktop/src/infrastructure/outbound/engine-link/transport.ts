import type { CancelFrame, Request } from '@jterrazz/station-api';

import type { Link } from '../../../application/ports/engine.port';

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

import type { EngineHello } from '@jterrazz/station-api';

import { fitOf } from '../../domain/fit';
import type { Unfit } from '../../domain/fit';
import type { Link } from '../ports/engine.port';

// value: the link as the window holds it once judged: an engine that answers at a revision it does not draw is
// value: `incompatible`, running and never said stopped, and nothing of it is read but its hello.
export type Judged =
    | Link
    | { kind: 'incompatible'; socket: string; fit: Unfit; hello: EngineHello | undefined };

// invariant: the one place the window judges an engine, from its hello's `page`; ADR-070's `opened.fit` replaces the
// invariant: `fitOf` here once the engine sends it.
export function judged(link: Link): Judged {
    if (link.kind !== 'open') {
        return link;
    }
    const fit = fitOf(link.hello?.page);
    return fit === 'fits'
        ? link
        : { fit, hello: link.hello, kind: 'incompatible', socket: link.socket };
}

// value: whether an engine answers on the window's socket: the transport's state, never the engine's own (running,
// value: stopped), which `engine.read` alone says. `said` is the sentence for the socket nobody answers on.
export type Reach =
    | { kind: 'answering'; socket: string }
    | { kind: 'connecting'; socket: string | undefined }
    | { kind: 'incompatible'; socket: string; fit: Unfit; executable: string | undefined }
    | { kind: 'unanswered'; socket: string; said: string; reason: string; retryIn: number };

export function reachOf(link: Judged | undefined): Reach {
    if (link === undefined || link.kind === 'connecting') {
        return { kind: 'connecting', socket: link?.socket };
    }
    if (link.kind === 'open') {
        return { kind: 'answering', socket: link.socket };
    }
    if (link.kind === 'incompatible') {
        const { fit, hello, socket } = link;
        return { executable: hello?.executable, fit, kind: 'incompatible', socket };
    }
    const { reason, retryIn, socket } = link;
    return { kind: 'unanswered', reason, retryIn, said: `No engine answers at ${socket}`, socket };
}

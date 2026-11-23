import type { EngineHello } from '@jterrazz/station-api';

import type { Fit, Unfit } from '../../domain/fit';
import type { Link } from '../ports/engine.port';

// value: the link as the window holds it once judged: an engine that answers at a revision it does not draw is
// value: `incompatible`, running and never said stopped, and nothing of it is read but its hello.
export type Judged =
    | Link
    | { kind: 'incompatible'; socket: string; fit: Unfit; hello: EngineHello | undefined };

// invariant: an open link is incompatible only on the engine's own verdict, its `opened.fit`; an engine not judged
// invariant: yet counts as open.
export function judgedOf(link: Link, fit: Fit | undefined): Judged {
    if (link.kind !== 'open' || fit === undefined || fit === 'fits') {
        return link;
    }
    return { fit, hello: link.hello, kind: 'incompatible', socket: link.socket };
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

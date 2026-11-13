import type { Link } from './transport';

// value: whether an engine answers on the window's socket: the transport's state, never the engine's own (running,
// value: stopped), which `engine.read` alone says. `said` is the sentence for the socket nobody answers on.
export type Reach =
    | { kind: 'answering'; socket: string }
    | { kind: 'connecting'; socket: string | undefined }
    | { kind: 'unanswered'; socket: string; said: string; reason: string; retryIn: number };

export function reachOf(link: Link | undefined): Reach {
    if (link === undefined || link.kind === 'connecting') {
        return { kind: 'connecting', socket: link?.socket };
    }
    if (link.kind === 'open') {
        return { kind: 'answering', socket: link.socket };
    }
    const { reason, retryIn, socket } = link;
    return { kind: 'unanswered', reason, retryIn, said: `No engine answers at ${socket}`, socket };
}

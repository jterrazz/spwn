import type { EnginePort } from '../ports/engine.port';
import { answered, failed, failedOf, heardOf, stop } from './entry';
import type { Asked, Entry, Heard } from './entry';
import { heardDone, heardOut, markOf } from './marks';

async function read(
    client: EnginePort,
    entry: Entry,
    asked: Extract<Asked, { kind: 'read' }>,
    current: () => boolean,
): Promise<void> {
    try {
        const out: unknown = await client.call(
            asked.name,
            asked.input,
            asked.target ? { target: asked.target } : {},
        );
        if (current()) {
            heardDone(entry.mark);
            entry.asking = false;
            answered(entry, heardOf(entry, out), true);
        }
    } catch (error) {
        if (current()) {
            entry.asking = false;
            failed(entry, failedOf(error));
        }
    }
}

export function ask(client: EnginePort, entry: Entry): void {
    stop(entry);
    entry.asking = true;
    entry.mark = markOf(entry.key);
    const { asked, asks, mark, reading } = entry;
    const current = () => entry.asks === asks;
    if (asked.kind === 'read') {
        void read(client, entry, asked, current);
        return;
    }
    // invariant: a re-ask over a settled answer holds its outs; so does a watch the client reconnects over any answer
    // invariant: shown, since its new stream starts again from its first, pending out.
    let holds = reading.kind === 'read' && reading.done;
    let last: Heard | undefined;
    entry.watch = client.watch(
        asked.name,
        asked.input,
        {
            done: () => {
                if (current()) {
                    heardDone(mark);
                    entry.asking = false;
                    entry.held = undefined;
                    if (last) {
                        answered(entry, last, true);
                    }
                }
            },
            error: (error) => {
                if (current()) {
                    const reconnects = error.wire === undefined;
                    entry.asking = reconnects;
                    if (reconnects && entry.reading.kind === 'read') {
                        holds = true;
                    } else {
                        failed(entry, error);
                    }
                }
            },
            out: (out: unknown) => {
                if (current()) {
                    heardOut(mark);
                    last = heardOf(entry, out);
                    if (holds) {
                        entry.held = last;
                    } else {
                        answered(entry, last, false);
                    }
                }
            },
        },
        asked.target,
    );
}

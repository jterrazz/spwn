import type { EnginePort } from '../ports/engine.port';
import { failed, failedOf, show, stop } from './entry';
import type { Asked, Entry } from './entry';
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
            show(entry, { done: true, kind: 'read', out });
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
    const holds = reading.kind === 'read' && reading.done;
    let last: undefined | { out: unknown };
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
                        show(entry, { done: true, kind: 'read', out: last.out });
                    }
                }
            },
            error: (error) => {
                if (current()) {
                    entry.asking = error.wire === undefined;
                    failed(entry, error);
                }
            },
            out: (out: unknown) => {
                if (current()) {
                    heardOut(mark);
                    last = { out };
                    if (holds) {
                        entry.held = last;
                    } else {
                        show(entry, { done: false, kind: 'read', out });
                    }
                }
            },
        },
        asked.target,
    );
}

import type { CallName, Calls, Progress, Target } from '@jterrazz/station-api';

import type { Bridged } from './bridged';
import { EngineError } from './engine-error';
import { answerOf } from './frame';
import type { Link, Transport } from './transport';
import { Watching } from './watch';
import type { Timing } from './watch';

type Watched = {
    [N in CallName]: Calls[N]['watch'] extends true ? N : never;
}[CallName];

type Called = {
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

function unansweredIn(within: number): string {
    return `the engine did not answer in ${String(Math.round(within / 1000))} s`;
}

type Settler<T> = {
    settled: () => boolean;
    heard: () => void;
    resolve: (out: T) => void;
    reject: (error: EngineError) => void;
};

function settlerOf<T>(
    promised: { resolve: (out: T) => void; reject: (error: unknown) => void },
    within: number,
    close: () => void,
): Settler<T> {
    let settled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const finish = (then: () => void) => {
        if (!settled) {
            settled = true;
            clearTimeout(timer);
            close();
            then();
        }
    };
    return {
        heard: () => {
            clearTimeout(timer);
            timer = setTimeout(() => {
                finish(() => {
                    promised.reject(new EngineError(undefined, unansweredIn(within), 'lost'));
                });
            }, within);
        },
        reject: (error) => {
            finish(() => {
                promised.reject(error);
            });
        },
        resolve: (out) => {
            finish(() => {
                promised.resolve(out);
            });
        },
        settled: () => settled,
    };
}

const DEFAULTS: Timing = { call: 30_000, firstBackoff: 1000, maxBackoff: 30_000, silence: 45_000 };

const TICKING: ReadonlySet<string> = new Set(['page']);

function isOut<N extends CallName>(_name: N, value: unknown): value is Out<N> {
    return value !== undefined;
}

export class Client {
    private readonly transport: Transport;
    private readonly timing: Timing;
    private next = 0;

    constructor(transport: Transport, timing: Partial<Timing> = {}) {
        this.transport = transport;
        this.timing = { ...DEFAULTS, ...timing };
    }

    link(listener: (link: Link) => void): () => void {
        return this.transport.link(listener);
    }

    // invariant: a call ends on its out or its error; one that hears nothing of its id for `within` ms (its
    // invariant: progress counts) is closed and lost, like one whose connection drops after it was sent.
    async call<N extends Called>(
        name: N,
        input: In<N>,
        options: CallOptions = {},
    ): Promise<Out<N>> {
        const id = this.id();
        const target = options.target ? { target: options.target } : {};
        const within = options.within ?? this.timing.call;
        return await new Promise<Out<N>>((resolve, reject) => {
            const settler = settlerOf<Out<N>>({ reject, resolve }, within, () => {
                exchange.close();
            });
            const exchange = this.transport.open(
                { call: name, id, in: input, v: 2, ...target },
                {
                    end: (reason, sent) => {
                        settler.reject(
                            new EngineError(undefined, reason, sent ? 'lost' : 'unsent'),
                        );
                    },
                    frame: (raw) => {
                        const answer = answerOf(raw);
                        if (settler.settled() || answer?.id !== id) {
                            return;
                        }
                        if ('progress' in answer) {
                            settler.heard();
                            options.progress?.(answer.progress);
                        } else if ('error' in answer) {
                            settler.reject(new EngineError(answer.error));
                        } else if ('out' in answer && isOut(name, answer.out)) {
                            settler.resolve(answer.out);
                        }
                    },
                },
            );
            if (!settler.settled()) {
                settler.heard();
            }
        });
    }

    watch<N extends Watched>(
        name: N,
        input: In<N>,
        handlers: WatchHandlers<N>,
        target?: Target,
    ): Watch {
        const watching = new Watching(
            this.transport,
            {
                next: () => this.id(),
                request: (id) => ({
                    id,
                    in: input,
                    v: 2,
                    watch: name,
                    ...(target ? { target } : {}),
                }),
                ticking: TICKING.has(name),
                timing: this.timing,
            },
            {
                done: () => handlers.done?.(),
                error: handlers.error,
                out: (out) => {
                    if (isOut(name, out)) {
                        handlers.out(out);
                    }
                },
            },
        );
        return {
            close: () => {
                watching.close();
            },
        };
    }

    private id(): string {
        this.next += 1;
        return String(this.next);
    }
}

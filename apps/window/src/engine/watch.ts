import type { Request } from '@jterrazz/station-api';

import { EngineError } from './engine-error';
import { answerOf } from './frame';
import type { Exchange, Receiver, Transport } from './transport';

export type Timing = { call: number; silence: number; firstBackoff: number; maxBackoff: number };

export type Stream = {
    out: (out: unknown) => void;
    error: (error: EngineError) => void;
    done: () => void;
};

type Run = { ended: boolean; exchange: Exchange | undefined };

function receiverOf(id: string, run: Run, stream: Stream, heard: () => void): Receiver {
    return {
        end: (reason) => {
            if (!run.ended) {
                run.ended = true;
                stream.error(new EngineError(undefined, reason));
            }
        },
        frame: (raw) => {
            const answer = answerOf(raw);
            if (run.ended || answer?.id !== id) {
                return;
            }
            heard();
            if ('out' in answer) {
                stream.out(answer.out);
            } else if ('error' in answer) {
                run.ended = true;
                stream.error(new EngineError(answer.error));
            } else if ('done' in answer) {
                run.ended = true;
                stream.done();
            }
        },
    };
}

export type Plan = {
    timing: Timing;
    request: (id: string) => Request;
    next: () => string;
    ticking: boolean;
};

export class Watching {
    private run: Run = { ended: true, exchange: undefined };
    private timer: ReturnType<typeof setTimeout> | undefined;
    private silence: ReturnType<typeof setTimeout> | undefined;
    private backoff: number;
    private closed = false;
    private readonly transport: Transport;
    private readonly plan: Plan;
    private readonly stream: Stream;

    constructor(transport: Transport, plan: Plan, stream: Stream) {
        this.transport = transport;
        this.plan = plan;
        this.stream = stream;
        this.backoff = plan.timing.firstBackoff;
        this.open();
    }

    close(): void {
        this.closed = true;
        clearTimeout(this.timer);
        this.stop();
    }

    private open(): void {
        const id = this.plan.next();
        const current: Run = { ended: false, exchange: undefined };
        this.run = current;
        this.heard();
        const guarded: Stream = {
            done: () => {
                this.stop();
                this.stream.done();
            },
            error: (error) => {
                this.failed(error);
            },
            out: (out) => {
                this.backoff = this.plan.timing.firstBackoff;
                this.stream.out(out);
            },
        };
        current.exchange = this.transport.open(
            this.plan.request(id),
            receiverOf(id, current, guarded, () => {
                this.heard();
            }),
        );
        if (current.ended) {
            current.exchange.close();
        }
    }

    private failed(error: EngineError): void {
        if (error.wire) {
            this.stop();
            this.stream.error(error);
        } else if (!this.closed) {
            this.stream.error(error);
            this.again();
        }
    }

    private stop(): void {
        clearTimeout(this.silence);
        this.run.ended = true;
        this.run.exchange?.close();
    }

    private again(): void {
        this.stop();
        if (!this.closed) {
            this.timer = setTimeout(() => {
                this.open();
            }, this.backoff);
            this.backoff = Math.min(this.backoff * 2, this.plan.timing.maxBackoff);
        }
    }

    private heard(): void {
        clearTimeout(this.silence);
        if (this.plan.ticking) {
            this.silence = setTimeout(() => {
                this.again();
            }, this.plan.timing.silence);
        }
    }
}

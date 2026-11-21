import { keyOf } from '../../../domain/request-key';
import type { GestureExchange, Recording } from './replay';

// value: a world's recorded answers, each read's and each gesture's by the key of its request.
export type Answers = {
    reads: Map<string, readonly unknown[]>;
    gestures: Map<string, GestureExchange>;
};

export function answersOf(recording: Recording): Answers {
    return {
        gestures: new Map(
            (recording.gestures ?? []).map((gesture) => [keyOf(gesture.request), gesture]),
        ),
        reads: new Map(
            recording.exchanges.map((exchange) => [keyOf(exchange.request), exchange.answers]),
        ),
    };
}

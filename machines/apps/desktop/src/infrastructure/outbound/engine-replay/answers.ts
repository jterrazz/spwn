import { isRecord, keyOf } from '../../../domain/request-key';
import { viewNamed } from '../../../domain/views';
import type { View } from '../../../domain/views';
import type { Exchange, GestureExchange, Recording } from './replay';

// value: a world's recorded answers, each read's and each gesture's by the key of its request, and the view each read
// value: the engine keeps is, by the key of its request.
export type Answers = {
    reads: Map<string, readonly unknown[]>;
    gestures: Map<string, GestureExchange>;
    views: ReadonlyMap<string, View>;
};

function viewOfExchange(exchange: Exchange): undefined | View {
    const { request } = exchange;
    const name = typeof request.watch === 'string' ? request.watch : request.call;
    const target = isRecord(request.target) ? request.target : undefined;
    const aim = target && {
        name: typeof target.name === 'string' ? target.name : undefined,
        on: typeof target.on === 'string' ? target.on : undefined,
    };
    return typeof name === 'string' ? viewNamed(name, request.in ?? {}, aim) : undefined;
}

export function answersOf(recording: Recording): Answers {
    const views = new Map<string, View>();
    const gestured = (recording.gestures ?? []).flatMap((g) => [...(g.during ?? []), ...g.after]);
    for (const exchange of [...recording.exchanges, ...gestured]) {
        const view = viewOfExchange(exchange);
        if (view !== undefined) {
            views.set(keyOf(exchange.request), view);
        }
    }
    return {
        gestures: new Map(
            (recording.gestures ?? []).map((gesture) => [keyOf(gesture.request), gesture]),
        ),
        reads: new Map(
            recording.exchanges.map((exchange) => [keyOf(exchange.request), exchange.answers]),
        ),
        views,
    };
}

import type { Recording } from '../replay';

export function unansweredAt(recording: Recording, name: string): Recording {
    return {
        ...recording,
        exchanges: recording.exchanges.filter((exchange) => {
            const { target } = exchange.request;
            return !(
                typeof target === 'object' &&
                target !== null &&
                'name' in target &&
                target.name === name
            );
        }),
    };
}

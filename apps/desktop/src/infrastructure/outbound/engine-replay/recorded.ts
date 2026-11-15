import { recordingOf } from './replay';
import type { Recording } from './replay';

export function worldOf(path: string): string {
    const parts = path.split('/');
    return parts[parts.indexOf('worlds') + 1] ?? path;
}

export function joined(
    world: string,
    files: readonly unknown[],
    worlds: readonly string[],
): Recording {
    const calls = files.map((value) => recordingOf(value));
    const [first] = calls;
    if (!first) {
        throw new Error(`No world is recorded as ${world}: ${worlds.join(', ')}.`);
    }
    return {
        ...first,
        exchanges: calls.flatMap((call) => call.exchanges),
        gestures: calls.flatMap((call) => call.gestures ?? []),
    };
}

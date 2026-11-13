import { recordingOf } from '../engine/replay';
import type { Recording } from '../engine/replay';

export function worldOf(path: string): string {
    return path.split('/')[2] ?? path;
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

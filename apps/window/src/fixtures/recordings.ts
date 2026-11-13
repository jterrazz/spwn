import type { Recording } from '../engine/replay';
import { joined, worldOf } from './recorded';

const LOADERS = import.meta.glob<unknown>(['./worlds/*/*.json', './worlds/*/gestures/*.json'], {
    import: 'default',
});

const WORLDS: readonly string[] = [...new Set(Object.keys(LOADERS).map(worldOf))];

export async function recordingFor(world: string): Promise<Recording> {
    const files = await Promise.all(
        Object.entries(LOADERS)
            .filter(([path]) => worldOf(path) === world)
            .map(async ([, load]) => await load()),
    );
    return joined(world, files, WORLDS);
}

// hazard: a spec loads its worlds before its first render: once a component renders, @jterrazz/test's network
// hazard: stub refuses every request the page makes, a recording's module included.
export async function recordingsOf(
    ...worlds: readonly string[]
): Promise<(world: string) => Recording> {
    const loaded = new Map(
        await Promise.all(worlds.map(async (world) => [world, await recordingFor(world)] as const)),
    );
    return (world) => {
        const recording = loaded.get(world);
        if (!recording) {
            throw new Error(`${world} is not loaded: recordingsOf(${worlds.join(', ')}).`);
        }
        return recording;
    };
}

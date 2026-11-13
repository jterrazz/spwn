import type { CallName } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import type { Out } from '../../../engine/client';
import { keyOf } from '../../../engine/frame';
import { recordingsOf } from '../../../fixtures/recordings';
import { byAction, countOn, layersOf, onShelf, SHELVES } from '../model';

const recordingFor = await recordingsOf('machinefile-differs', 'stories-file', 'two-macs');

type Asked<N extends CallName> = { name: N; watch: boolean; in?: Record<string, string> };

function answered<N extends CallName>(world: string, request: Asked<N>): Out<N> {
    const asked = keyOf({
        [request.watch ? 'watch' : 'call']: request.name,
        in: request.in ?? {},
    });
    const exchange = recordingFor(world).exchanges.find((one) => keyOf(one.request) === asked);
    const outs = (exchange?.answers ?? []).filter(
        (answer): answer is { out: Out<N> } =>
            typeof answer === 'object' && answer !== null && 'out' in answer,
    );
    const last = outs.at(-1);
    if (!last) {
        throw new Error(`${world} records no answer to ${asked}.`);
    }
    return last.out;
}

describe('model', () => {
    test("a file's lines are grouped by the layer that sets each, in the engine's order, a colour per position", () => {
        // Given - This Mac's drift from its file over machinefile-differs, homebrew brought by jq
        const plan = answered('machinefile-differs', {
            name: 'machinefile.drift',
            watch: false,
        }).Plan;

        // When - its lines are grouped by layer
        const layers = layersOf(plan);

        // Then - each file once, as the engine first names it; homebrew sits in the layer of jq, its bringer
        expect(
            layers.map((layer) => [layer.name, layer.identity, layer.lines.length]),
        ).toStrictEqual([
            ['jterrazz-base', 'blue', 2],
            ['jterrazz-macbook-pro', 'purple', 4],
        ]);
    });

    test('each shelf holds the lines of its kind, a skill told by its category', () => {
        // Given - This Mac's drift over stories-file, its one setting, and gh filed as an agent skill
        const layers = layersOf(
            answered('stories-file', { name: 'machinefile.drift', watch: false }).Plan,
        );
        const categories = new Map([['gh', 'Agent skills' as const]]);

        // When - the lines are counted on each shelf, and the settings drawn
        const counts = SHELVES.map((shelf) => countOn(layers, shelf, categories));
        const settings = onShelf(layers, 'Settings', categories);

        // Then - five apps less the skill, one skill, one setting in the layer that sets it
        expect(counts).toStrictEqual([4, 1, 1]);
        expect(settings.map((layer) => [layer.name, layer.lines[0]?.Step.Package])).toStrictEqual([
            ['macbook-pro', 'git-shortcuts'],
        ]);
    });

    test('a plan groups its steps by the action the engine chose', () => {
        // Given - the plan of the built-in station-macos read over two-macs
        const plan = answered('two-macs', {
            in: { Machinefile: 'station-macos' },
            name: 'machinefile.plan',
            watch: false,
        });

        // When - its lines are grouped
        const groups = byAction(plan.Lines ?? []);

        // Then - each group holds the lines of its action, in the plan's order
        expect(groups.map(([action]) => action)).toStrictEqual(['keep', 'install', 'set']);
        expect(groups.flatMap(([, lines]) => lines)).toHaveLength(plan.Lines?.length ?? 0);
    });
});

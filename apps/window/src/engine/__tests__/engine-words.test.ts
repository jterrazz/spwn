import { describe, expect, test } from 'vitest';

import { recordingsOf } from '../../fixtures/recordings';
import type { Out } from '../client';
import { codesOf, wordOf } from '../engine-words';
import { keyOf } from '../frame';
import { vmLifetimeWord } from '../shown';

const recordingFor = await recordingsOf('two-macs');

function vmsOf(world: string) {
    const asked = keyOf({ in: {}, watch: 'machines.list' });
    const exchange = recordingFor(world).exchanges.find((one) => keyOf(one.request) === asked);
    const outs = (exchange?.answers ?? []).filter(
        (answer): answer is { out: Out<'machines.list'> } =>
            typeof answer === 'object' && answer !== null && 'out' in answer,
    );
    return (outs.at(-1)?.out.All.Macs ?? []).flatMap((mac) => mac.VMs ?? []);
}

describe('engine words', () => {
    test("every recorded VM's lifetime is shown by the engine's word, a server's always on", () => {
        // Given - the VMs of both Macs in two-macs
        const vms = vmsOf('two-macs');

        // When - each VM's lifetime is worded
        const shown = vms.map((vm) => [vm.Name, vmLifetimeWord(vm)]);

        // Then - each is the engine's word, infra's "always on"
        expect(shown.length).toBeGreaterThan(2);
        expect(shown).toContainEqual(['infra', 'always on']);
        expect(
            vms.every((vm) => vmLifetimeWord(vm) === wordOf('MachinesLifetime', vm.Lifetime).Word),
        ).toBe(true);
    });

    test("a lifetime carries the engine's detail, and the systems are the engine's three", () => {
        // Given - the engine's words for a lifetime and for the systems a package lives on
        const ephemeral = wordOf('MachinesLifetime', 'ephemeral');
        const systems = codesOf('CatalogMachineKind');

        // Then - its detail is the engine's sentence, and every system is listed
        expect(ephemeral.Detail).toBe('Deleted when it powers off');
        expect(systems.toSorted()).toStrictEqual(['linux', 'mac', 'macos']);
    });
});

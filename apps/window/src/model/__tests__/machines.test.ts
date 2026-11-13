import { describe, expect, test } from 'vitest';

import type { Out } from '../../engine/client';
import { keyOf } from '../../engine/frame';
import { recordingsOf } from '../../fixtures/recordings';
import { targetOf } from '../machines';

const recordingFor = await recordingsOf('two-macs');

function macs() {
    const asked = keyOf({ in: {}, watch: 'machines.list' });
    const exchange = recordingFor('two-macs').exchanges.find((one) => keyOf(one.request) === asked);
    const outs = (exchange?.answers ?? []).filter(
        (answer): answer is { out: Out<'machines.list'> } =>
            typeof answer === 'object' && answer !== null && 'out' in answer,
    );
    return outs.at(-1)?.out.All.Macs ?? [];
}

describe('machines', () => {
    test("a gesture goes to a VM's engine by its name, a far Mac's by its name, and this Mac's with no target", () => {
        // Given - the Macs of two-macs: this Mac with its VMs, and a far Mac
        const listed = macs();
        const here = listed.find((mac) => mac.ThisMac);
        const far = listed.find((mac) => !mac.ThisMac);
        const vm = here?.VMs?.[0];
        if (!here || !far || !vm) {
            throw new Error('two-macs records this Mac with a VM, and a far Mac.');
        }

        // When - the target of each is read
        const targets = [
            targetOf({ mac: here }),
            targetOf({ mac: far }),
            targetOf({ mac: here, vm }),
        ];

        // Then - none, the far Mac by its name, the VM by its name
        expect(targets).toStrictEqual([undefined, { name: far.Name }, { name: vm.Name }]);
    });
});

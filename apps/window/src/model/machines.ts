import type {
    MachinesListOutput,
    MachinesMacView,
    MachinesVM,
    Target,
} from '@jterrazz/station-api';

import { bytes, memory, sentence } from './format';

export type Found = { mac: MachinesMacView; vm?: MachinesVM | undefined };

export function macsOf(out: MachinesListOutput | undefined): readonly MachinesMacView[] {
    return out?.All.Macs ?? [];
}

export function findMachine(macs: readonly MachinesMacView[], name: string): Found | undefined {
    for (const mac of macs) {
        if (mac.Name === name) {
            return { mac };
        }
        const vm = (mac.VMs ?? []).find((candidate) => candidate.Name === name);
        if (vm) {
            return { mac, vm };
        }
    }
    return undefined;
}

// invariant: the engine a read or a gesture is sent to: a VM's own, else a Mac's by name; this Mac's is the one the
// invariant: socket reaches, so it takes no target, and a read of it is the same request whichever page asks it.
export function targetOf(found: Found): Target | undefined {
    if (found.vm) {
        return { name: found.vm.Name };
    }
    return found.mac.ThisMac ? undefined : { name: found.mac.Name };
}

export function resourcesOf(vm: MachinesVM): string {
    const { CPUs, MemoryBytes, DiskBytes } = vm.Resources;
    return sentence([
        CPUs > 0 ? `${CPUs} CPU` : undefined,
        MemoryBytes > 0 ? memory(MemoryBytes) : undefined,
        DiskBytes > 0 ? `${bytes(DiskBytes)} disk` : undefined,
    ]);
}

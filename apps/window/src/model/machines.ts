import type {
    MachinefileDriftInput,
    MachinesListOutput,
    MachinesMacView,
    MachinesVM,
    Target,
} from '@jterrazz/station-api';

export type Found = { mac: MachinesMacView; vm?: MachinesVM | undefined };

export function macsOf(out: MachinesListOutput | undefined): readonly MachinesMacView[] {
    return out?.All.Macs ?? [];
}

export function machineNamesOf(macs: readonly MachinesMacView[]): readonly string[] {
    return macs.flatMap((mac) => (mac.VMs ?? []).map((vm) => vm.Name));
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

// invariant: a drift is asked of this Mac's engine with the machine in its input and no target: a target would run it
// invariant: inside the machine, where a VM's Mac's shares do not read; this Mac is the input left empty.
export function driftAsked(found: Found): MachinefileDriftInput {
    if (found.vm) {
        return { Machine: found.vm.Name };
    }
    return found.mac.ThisMac ? {} : { Machine: found.mac.Name };
}

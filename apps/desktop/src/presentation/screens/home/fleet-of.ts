import type {
    ActivityReport,
    MachinesMacView,
    MachinesRollUp,
    MachinesRoom,
    MachinesShare,
    MachinesVM,
} from '@jterrazz/station-api';

import { cpuOf } from '../../../domain/activity/measures';
import { gigabytes, percent } from '../../../domain/format';
import type { IdentityOf } from '../../kit/identity';
import type { FleetBoardProps, FleetMac, FleetMachine, FleetShare } from '../../ui/index';
import { countOf } from '../../words/engine-words';
import { macCardState, rollUpState, vmState } from '../../words/shown';
import { THIS_MAC_WORD } from '../../words/words';

export type Fleet = Pick<FleetBoardProps, 'macs' | 'standing' | 'thisMac'>;

export type FleetReads = {
    macs: readonly MachinesMacView[];
    rooms: readonly MachinesRoom[];
    reports: ReadonlyMap<string, ActivityReport>;
    identityOf: IdentityOf;
    standing?: MachinesRollUp | undefined;
};

type Placed = { share: MachinesShare; at: number };

function placedOf(room: MachinesRoom): ReadonlyMap<string, Placed> {
    const { MacOS, VMs } = room.Capacity;
    let at = MacOS;
    return new Map(
        (VMs ?? []).map((share) => {
            const placed = { at, share };
            at += share.Bytes;
            return [share.Name, placed] as const;
        }),
    );
}

function shareOf({ share, at }: Placed, total: number): FleetShare {
    return {
        at: at / total,
        used: share.UsedBytes === undefined ? undefined : share.UsedBytes / share.Bytes,
        width: share.Bytes / total,
    };
}

function heldOf({ Bytes, UsedBytes }: MachinesShare): string {
    return UsedBytes === undefined
        ? `of ${gigabytes(Bytes)}`
        : `${gigabytes(UsedBytes)} / ${gigabytes(Bytes)}`;
}

function cpuOfVm(name: string, report: ActivityReport | undefined): string {
    const usage = report?.last.vms?.find((one) => one.vm === name);
    return usage === undefined ? '' : `${percent(usage.cpu)} %`;
}

type MacReads = {
    room: MachinesRoom | undefined;
    report: ActivityReport | undefined;
    identityOf: IdentityOf;
};

function machineOf(vm: MachinesVM, { room, report, identityOf }: MacReads): FleetMachine {
    const placed = room === undefined ? undefined : placedOf(room).get(vm.Name);
    return {
        cpu: cpuOfVm(vm.Name, report),
        identity: identityOf(vm.Name),
        line: vm.Head?.Line,
        memory: placed === undefined ? '' : heldOf(placed.share),
        name: vm.Name,
        purpose: vm.Purpose,
        sentence: vm.Head?.Sentence ?? '',
        share:
            placed === undefined || room === undefined
                ? undefined
                : shareOf(placed, room.Capacity.Total),
        state: vmState(vm),
    };
}

function cpuOfMac(report: ActivityReport | undefined): string {
    const cpu = report === undefined ? undefined : cpuOf(report);
    return cpu === undefined ? '' : `${cpu} %`;
}

function macOf(mac: MachinesMacView, reads: MacReads): FleetMac {
    const vms = mac.VMs ?? [];
    const { room } = reads;
    return {
        cpu: cpuOfMac(reads.report),
        free: room === undefined ? '' : `${gigabytes(room.Capacity.Free)} free`,
        kind: vms.length === 0 ? undefined : countOf('machine', vms.length),
        machines: vms.map((vm) => machineOf(vm, reads)),
        name: mac.Name,
        state: macCardState(mac),
        system:
            room === undefined
                ? undefined
                : { at: 0, used: 1, width: room.Capacity.MacOS / room.Capacity.Total },
        thisMac: mac.ThisMac,
    };
}

export function fleetOf({ macs, rooms, reports, identityOf, standing }: FleetReads): Fleet {
    return {
        macs: macs.map((mac) =>
            macOf(mac, {
                identityOf,
                report: reports.get(mac.Name),
                room: rooms.find((one) => one.Mac === mac.Name),
            }),
        ),
        standing: standing === undefined ? undefined : rollUpState(standing),
        thisMac: THIS_MAC_WORD,
    };
}

import type {
    AppsCell,
    AppsUpdateInput,
    BackgroundRemoveInput,
    DiskMoveInput,
    DiskReading,
    DiskRow,
    EngineStartInput,
    ImagesReport,
    MachinefileDiffering,
    MachinefileExtraLine,
    MachinefileLine,
    MachinesDeleteInput,
    MachinesFit,
    MachinesMacView,
    MachinesRestartInput,
    MachinesVM,
    OverviewNeed,
    PortsListing,
    PortsMacPorts,
    PortsPort,
    PortsSharedPort,
} from '@jterrazz/station-api';

import { duration } from '../model/format';
import type { EngineError } from './engine-error';

// outside: machines.list sends a Mac's reach as two booleans and no state code; this file alone makes it a code, until
// outside: the wire carries one.

export type MacState = 'asleep' | 'reachable' | 'unreachable';

// outside: a preset's Fit is the engine's verdict on whether it can be asked, and machines.new names no gesture.
export function fitAllows(fit: MachinesFit): boolean {
    return fit === 'fits';
}

export type RefusalPlace = 'file' | 'foot' | 'memory' | 'on' | 'purpose';

// outside: a create's refusal names no field of the sheet it is about, so its code places it (purpose, memory, file);
// outside: one of another code whose subject is a Mac the sheet offers is that Mac's, any other the sheet's own.
const CREATE_REFUSAL_PLACES: Readonly<Partial<Record<string, RefusalPlace>>> = {
    'machinefile.not-a-template': 'file',
    'machinefile.not-found': 'file',
    'machines.bad-purpose': 'purpose',
    'machines.server-template': 'file',
    'machines.too-little-memory': 'memory',
};

export function createRefusalPlaceOf(error: EngineError, macs: readonly string[]): RefusalPlace {
    const onAMac = macs.includes(error.wire?.subject ?? '');
    return CREATE_REFUSAL_PLACES[error.wire?.code ?? ''] ?? (onAMac ? 'on' : 'foot');
}

export type MachineKind = 'server' | 'workspace';

// outside: machines.list sends no kind; Go reads it off the Lifetime (apps/engine/machines/run.go, Lifetime.Kind).
export function machineKindOf(vm: Pick<MachinesVM, 'Lifetime'>): MachineKind {
    return vm.Lifetime === 'always-on' ? 'server' : 'workspace';
}

// outside: apps.matrix sends a cell's state and no flag that its machine carries the app; its Carry counts the states
// outside: the engine calls installed (apps/engine/apps/state.go, Installed), which this table repeats.
const CARRYING: Readonly<Record<AppsCell['State'], boolean>> = {
    '': false,
    failed: true,
    'not-installed': false,
    'not-known': false,
    'not-managed': true,
    ok: true,
    pending: false,
    'update-available': true,
};

export function carries(cell: Pick<AppsCell, 'State'>): boolean {
    return CARRYING[cell.State];
}

export function macStateOf(mac: MachinesMacView): MacState {
    if (mac.Asleep) {
        return 'asleep';
    }
    return mac.Reachable ? 'reachable' : 'unreachable';
}

// outside: a need's `Args` are its call's input fields, a list field holding the one value given; the call's input
// outside: type is not on the wire, so each call's reads its fields here, a list field wrapped.
export function updateInputOf(need: OverviewNeed): AppsUpdateInput | undefined {
    if (need.Call !== 'apps.update') {
        return undefined;
    }
    const manager = need.Args?.Managers ?? '';
    if (manager !== '') {
        return { Managers: [manager] };
    }
    const name = need.Args?.Names ?? '';
    return name === '' ? undefined : { Names: [name] };
}

export function moveInputOf(need: OverviewNeed): DiskMoveInput | undefined {
    const folder = need.Args?.Folder ?? '';
    return need.Call !== 'disk.move' || folder === '' ? undefined : { Folder: folder };
}

export function removeInputOf(need: OverviewNeed): BackgroundRemoveInput | undefined {
    const label = need.Args?.Label ?? '';
    return need.Call !== 'background.remove' || label === '' ? undefined : { Label: label };
}

export function deleteInputOf(need: OverviewNeed): MachinesDeleteInput | undefined {
    const vm = need.Args?.VM ?? '';
    return need.Call !== 'machines.delete' || vm === '' ? undefined : { VM: vm };
}

export function restartInputOf(need: OverviewNeed): MachinesRestartInput | undefined {
    const name = need.Args?.Name ?? '';
    return need.Call !== 'machines.restart' || name === '' ? undefined : { Name: name };
}

export function startInputOf(need: OverviewNeed): EngineStartInput | undefined {
    return need.Call === 'engine.start' ? {} : undefined;
}

export type NeedCall =
    | 'apps.update'
    | 'background.remove'
    | 'disk.move'
    | 'engine.start'
    | 'machines.delete'
    | 'machines.restart';

// outside: a listening port carries no field saying it is shared (apps/engine/ports/exposure.go, Port).
export function isPortShared(port: PortsPort, shares: readonly PortsSharedPort[]): boolean {
    return shares.some((share) => share.guest_port === port.Port);
}

export type DiskCall = 'disk.erase' | 'disk.free' | 'disk.move';
export type DiskGesture = { call: DiskCall; danger: boolean };

// outside: a disk.read row carries no gesture: what each group allows, and whether it destroys, is this table's
// outside: until DiskRow names its gesture (apps/engine/disk/group.go).
const DISK_GESTURES: Readonly<Record<DiskRow['Group'], DiskGesture | undefined>> = {
    'can-rebuild': { call: 'disk.free', danger: false },
    misplaced: { call: 'disk.move', danger: false },
    'never-touched': undefined,
    'your-data': { call: 'disk.erase', danger: true },
};

export function diskGestureOf(group: DiskRow['Group']): DiskGesture | undefined {
    return DISK_GESTURES[group];
}

// outside: disk.read gives no gesture of a whole group: freeing every rebuildable row at once is the one there is.
export function freesWhole(group: DiskRow['Group']): boolean {
    return group === 'can-rebuild';
}

// outside: images.list carries no "can be deleted": not the newest, and none a machine uses, until each image does.
export function isDeletable(image: Pick<ImagesReport, 'Newest' | 'VMs'>): boolean {
    return !image.Newest && (image.VMs ?? []).length === 0;
}

// outside: disk.read flags no volume as holding what can be rebuilt, only each row (apps/engine/disk/group.go).
export function holdsRebuildable(reading: DiskReading): boolean {
    return (reading.Rows ?? []).some((row) => freesWhole(row.Group));
}

export type ShareEnd =
    | { kind: 'ended' }
    | { kind: 'ends-in'; nanoseconds: number }
    | { kind: 'unread' };

const NS_PER_MS = 1_000_000;

// outside: accounts.read gives a share its end and the report its hour, and no state of the share: whether it still
// outside: holds, and how long, is read from the two until the share carries both.
export function shareEndOf(until: string, now: string): ShareEnd {
    const left = Date.parse(until) - Date.parse(now);
    if (Number.isNaN(left)) {
        return { kind: 'unread' };
    }
    return left > 0 ? { kind: 'ends-in', nanoseconds: left * NS_PER_MS } : { kind: 'ended' };
}

export function shareEndWordOf(end: ShareEnd, until: string): string {
    switch (end.kind) {
        case 'ends-in': {
            return `ends in ${duration(end.nanoseconds)}`;
        }
        case 'ended': {
            return `ended at ${until}`;
        }
        case 'unread': {
            return `until ${until}`;
        }
    }
}

export type PlanLine = MachinefileExtraLine | MachinefileLine;

export type AccountSide = 'not-in-the-file' | 'not-shared' | 'shared';

// outside: machinefile.drift says an account's difference by its Condition alone, extra where the Mac holds a copy the
// outside: file does not share, absent where the file shares one the Mac does not hold: each side is read from it,
// outside: until a difference carries the word of each side.
export function accountSideOf(
    side: 'asked' | 'seen',
    differing: MachinefileDiffering,
): AccountSide {
    const extra = differing.Condition === 'extra';
    if (side === 'asked') {
        return extra ? 'not-in-the-file' : 'shared';
    }
    return extra ? 'shared' : 'not-shared';
}

// outside: ports.list may name this Mac otherwise than machines.list does (the stories world names it ''), and
// outside: flags it this_mac: this Mac's ports are the ones it flags, until both calls name a Mac alike.
export function portsMacOf(
    listing: PortsListing,
    name: string,
    thisMac: boolean,
): PortsMacPorts | undefined {
    const macs = listing.all.macs ?? [];
    return (
        macs.find((candidate) => candidate.name === name) ??
        (thisMac ? macs.find((candidate) => candidate.this_mac) : undefined)
    );
}

const OUTSIDE_TAILSCALE = 2;

// outside: ports.list names no gesture per port: a port reached outside Tailscale that no declaration names public is
// outside: the one ports.declare takes, by the LAN it answers on, until each port carries the gesture it allows.
export const EXPOSED_ROUTE = 'lan';
export function isDeclarable(port: PortsPort): boolean {
    return port.Reach === OUTSIDE_TAILSCALE && port.Declared === null;
}

// outside: ports.list gives a share a public and a local address and no address to open from the window's Mac: a
// outside: local one opens only on the Mac its row is flagged this_mac, until each share carries that address.
export function shareOpensFromHere(share: PortsSharedPort, onThisMac: boolean): string {
    if (share.urls.public !== '') {
        return share.urls.public;
    }
    return onThisMac ? share.urls.local : '';
}

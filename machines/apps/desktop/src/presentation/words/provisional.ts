import type {
    AccountsProgram,
    AppsUpdateInput,
    BackgroundRemoveInput,
    DiskMoveInput,
    DiskRow,
    EngineStartInput,
    MachinefileExtraLine,
    MachinefileLine,
    MachinesFit,
    MachinesRestartInput,
    MachinesStartInput,
    MachinesVM,
    OverviewNeed,
    PortsPort,
    PortsSharedPort,
} from '@jterrazz/station-api';

import type { EngineError } from '../../domain/engine-error';
import { codesOf } from './engine-words';

// outside: a preset's Fit is the engine's verdict on whether it can be asked, and machines.new names no gesture.
export function fitAllows(fit: MachinesFit): boolean {
    return fit === 'fits';
}

export type RefusalPlace = 'file' | 'foot' | 'memory' | 'name' | 'on' | 'purpose';

// outside: a create's refusal names no field of the sheet it is about, so its code places it (name, purpose, memory,
// outside: file); one of another code whose subject is a Mac the sheet offers is that Mac's, any other the sheet's own.
const CREATE_REFUSAL_PLACES: Readonly<Partial<Record<string, RefusalPlace>>> = {
    'machinefile.not-a-template': 'file',
    'machinefile.not-found': 'file',
    'machines.bad-name': 'name',
    'machines.bad-purpose': 'purpose',
    'machines.name-taken': 'name',
    'machines.server-template': 'file',
    'machines.template-name': 'name',
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

// outside: a need's or a journal line's Retry `Args` are its call's input fields, a list field holding the one value
// outside: given; the call's input type is not on the wire, so each call's reads its fields here, a list field wrapped.
export function updateInputOf(
    need: Pick<OverviewNeed, 'Args' | 'Call'>,
): AppsUpdateInput | undefined {
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

export function loginProgramOf(need: OverviewNeed): AccountsProgram | undefined {
    return need.Call === 'accounts.login'
        ? codesOf('AccountsProgram').find((program) => program === need.Args?.Program)
        : undefined;
}

export function moveInputOf(need: OverviewNeed): DiskMoveInput | undefined {
    const folder = need.Args?.Folder ?? '';
    return need.Call !== 'disk.move' || folder === '' ? undefined : { Folder: folder };
}

export function removeInputOf(need: OverviewNeed): BackgroundRemoveInput | undefined {
    const label = need.Args?.Label ?? '';
    return need.Call !== 'background.remove' || label === '' ? undefined : { Label: label };
}

export function restartInputOf(need: OverviewNeed): MachinesRestartInput | undefined {
    const name = need.Args?.Name ?? '';
    return need.Call !== 'machines.restart' || name === '' ? undefined : { Name: name };
}

export function startVmInputOf(need: OverviewNeed): MachinesStartInput | undefined {
    const name = need.Args?.VM ?? '';
    return need.Call !== 'machines.start' || name === '' ? undefined : { VM: name };
}

export function startInputOf(need: OverviewNeed): EngineStartInput | undefined {
    return need.Call === 'engine.start' ? {} : undefined;
}

export type NeedCall =
    | 'apps.update'
    | 'background.remove'
    | 'disk.move'
    | 'engine.start'
    | 'machines.restart'
    | 'machines.start';

// outside: a listening port carries no field saying it is shared (apps/engine/ports/exposure.go, Port).
export function isPortShared(port: PortsPort, shares: readonly PortsSharedPort[]): boolean {
    return shares.some((share) => share.guest_port === port.Port);
}

export type DiskCall = 'disk.erase' | 'disk.free' | 'disk.free-unused' | 'disk.move';
export type DiskGesture = { call: Exclude<DiskCall, 'disk.free-unused'>; danger: boolean };

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

export type PlanLine = MachinefileExtraLine | MachinefileLine;

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

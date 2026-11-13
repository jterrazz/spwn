import type {
    AccountsAccount,
    AppsUpdateInput,
    BackgroundRemoveInput,
    CatalogApp,
    DiskMoveInput,
    DiskReading,
    DiskRow,
    EngineReading,
    EngineStartInput,
    MachinefileExtraLine,
    MachinefileLine,
    MachinefilePlan,
    MachinesEngineView,
    MachinesMacView,
    MachinesVM,
    OverviewNeed,
    PortsListing,
    PortsMacPorts,
    PortsPort,
    PortsSharedPort,
} from '@jterrazz/station-api';

import { duration } from '../model/format';
import type { ConfirmName } from './confirms';
import type { AskForm } from './gesture';

// outside: machines.list sends a Mac's reach as two booleans and no state code, and packages/api types a VM's
// outside: State as a plain string though station's Go gives it seven codes (apps/engine/machines/allmachines.go,
// outside: birth.go). This file is the one place either becomes a code; it goes when the wire carries both.

export type MacState = 'asleep' | 'reachable' | 'unreachable';

export const VM_STATES = [
    'broken',
    'creating',
    'running',
    'stale',
    'stopped',
    'suspended',
    'unknown',
] as const;

export type VmState = (typeof VM_STATES)[number];

// outside: packages/api types a create's Lifetime as a plain string though station's Go asks one of two codes of
// outside: a workspace (apps/engine/machines/run.go; a server's is always on, set by the engine): this is that list.
export const LIFETIMES = ['ephemeral', 'persistent'] as const;

export type Lifetime = (typeof LIFETIMES)[number];

// outside: packages/api types a VM's Lifetime as a plain string though station's Go names three codes, a server's
// outside: always on among them (apps/engine/machines/run.go); the recorded worlds send its older words too (throwaway,
// outside: permanent), which are shown as they came.
export const VM_LIFETIMES = [...LIFETIMES, 'always-on'] as const;

export type VmLifetime = (typeof VM_LIFETIMES)[number];

export function vmLifetimeOf(vm: MachinesVM): undefined | VmLifetime {
    return VM_LIFETIMES.find((lifetime) => lifetime === vm.Lifetime);
}

// outside: packages/api types engine.read's State as a plain string though station's Go gives it three codes
// outside: (apps/engine/engine/reading.go).
export const ENGINE_STATES = ['not-installed', 'running', 'stopped'] as const;

export type EngineState = (typeof ENGINE_STATES)[number];

export function engineStateOf(reading: EngineReading): EngineState | undefined {
    return ENGINE_STATES.find((state) => state === reading.State);
}

export function engineViewStateOf(engine: MachinesEngineView): EngineState | undefined {
    if (!engine.Known) {
        return undefined;
    }
    if (engine.Running) {
        return 'running';
    }
    return engine.Installed ? 'stopped' : 'not-installed';
}

export type MachineKind = 'server' | 'workspace';

// outside: machines.list sends no kind; Go reads it off the Lifetime (apps/engine/machines/run.go, Lifetime.Kind).
export function machineKindOf(vm: Pick<MachinesVM, 'Lifetime'>): MachineKind {
    return vm.Lifetime === 'always-on' ? 'server' : 'workspace';
}

export function macStateOf(mac: MachinesMacView): MacState {
    if (mac.Asleep) {
        return 'asleep';
    }
    return mac.Reachable ? 'reachable' : 'unreachable';
}

export function vmStateOf(vm: MachinesVM): undefined | VmState {
    return VM_STATES.find((state) => state === vm.State);
}

// outside: a need's fix names its call and a few words (`Args`), not the call's input: the menu bar's page
// outside: composes that input in Go (apps/engine/pages/menubar/menubar.go, Update all). Until a need carries its `in`,
// outside: this is the one place the window does it, one function per call, each word to its field by name;
// outside: disk.move's folder is the one path its finding names (apps/engine/disk/findings.go).
export function updateInputOf(need: OverviewNeed): AppsUpdateInput | undefined {
    if (need.Call !== 'apps.update') {
        return undefined;
    }
    const manager = need.Args?.manager ?? '';
    if (manager !== '') {
        return { Managers: [manager] };
    }
    const name = need.Args?.name ?? '';
    return name === '' ? undefined : { Names: [name] };
}

export function moveInputOf(need: OverviewNeed): DiskMoveInput | undefined {
    const folder = need.Args?.path ?? '';
    return need.Call !== 'disk.move' || folder === '' ? undefined : { Folder: folder };
}

export function removeInputOf(need: OverviewNeed): BackgroundRemoveInput | undefined {
    const label = need.Args?.name ?? '';
    return need.Call !== 'background.remove' || label === '' ? undefined : { Label: label };
}

export function startInputOf(need: OverviewNeed): EngineStartInput | undefined {
    return need.Call === 'engine.start' ? {} : undefined;
}

export type NeedCall = 'apps.update' | 'background.remove' | 'disk.move' | 'engine.start';

// outside: calls.json declares that a call asks a confirmation, not how: which gestures are confirmed by their
// outside: subject typed rather than a yes is this table's, one entry per confirmed call, until the call says it.
const CONFIRM_FORMS: Readonly<Record<ConfirmName, AskForm>> = {
    'accounts.take-back': 'yes',
    'agents.stop': 'yes',
    'apps.remove': 'yes',
    'background.remove': 'yes',
    'disk.erase': 'name',
    'engine.stop': 'yes',
    'images.delete': 'yes',
    'machinefile.apply': 'yes',
    'machines.cleanup': 'yes',
    'machines.create-server': 'yes',
    'machines.create-workspace': 'yes',
    'machines.delete': 'name',
    'machines.forget': 'yes',
    'machines.restart': 'yes',
    'machines.restart-mac': 'yes',
    'machines.stop': 'yes',
    'ports.stop-sharing': 'yes',
    'ports.undeclare': 'yes',
    'settings.apply': 'yes',
    'settings.off': 'yes',
    'settings.run-containers': 'yes',
};

export function confirmFormOf(call: ConfirmName): AskForm {
    return CONFIRM_FORMS[call];
}

// outside: a listening port carries no field saying it is shared (apps/engine/ports/exposure.go, Port).
export function isPortShared(port: PortsPort, shares: readonly PortsSharedPort[]): boolean {
    return shares.some((share) => share.guest_port === port.Port);
}

export type DiskCall = 'disk.erase' | 'disk.free' | 'disk.move';

export type DiskGesture = { call: DiskCall; danger: boolean };

// outside: a disk.read row carries no gesture: what each group allows, and whether it destroys, is this table's
// outside: until DiskRow names its gesture (apps/engine/disk/group.go).
const DISK_GESTURES: Readonly<Record<DiskRow['Group'], DiskGesture | undefined>> = {
    '': undefined,
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

// outside: disk.read flags no volume as holding what can be rebuilt, only each row (apps/engine/disk/group.go).
export function holdsRebuildable(reading: DiskReading): boolean {
    return (reading.Rows ?? []).some((row) => freesWhole(row.Group));
}

// outside: overview.read gives no urgency of the Inbox, only each need's Broken: one broken need makes it urgent.
export function inboxIsUrgent(needs: readonly OverviewNeed[]): boolean {
    return needs.some((need) => need.Broken);
}

export type ShareEnd =
    | { kind: 'ended' }
    | { kind: 'ends-in'; nanoseconds: number }
    | { kind: 'unread' };

const NS_PER_MS = 1_000_000;

// outside: accounts.list gives a share its end and the report its hour, and no state of the share: whether it still
// outside: holds, and how long, is read from the two until the share carries both.
export function shareEndOf(until: string, now: string): ShareEnd {
    const left = Date.parse(until) - Date.parse(now);
    if (Number.isNaN(left)) {
        return { kind: 'unread' };
    }
    return left > 0 ? { kind: 'ends-in', nanoseconds: left * NS_PER_MS } : { kind: 'ended' };
}

// outside: accounts.list sends an identity it could not read as "?", not as no identity, until it leaves it out.
export function identitiesOf(account: AccountsAccount): readonly string[] {
    return (account.identity ?? []).filter((one) => one !== '?');
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

// outside: machinefile.drift gives every line of a plan with its condition, and does not say which differ from the
// outside: file: a line differs when the machine does not hold it as the file asks, until the plan lists those.
export function differingLines(plan: MachinefilePlan): readonly PlanLine[] {
    return [...(plan.Lines ?? []), ...(plan.Extra ?? [])].filter(
        (line) => line.Seen.Condition !== 'present',
    );
}

// outside: apps.list marks no entry a skill; the catalog files skills under one category, until each carries a kind.
export function isSkill(category: CatalogApp['Category']): boolean {
    return category === 'Agent skills';
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

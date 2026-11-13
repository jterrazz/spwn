import type { MachinesMacView, MachinesRoom } from '@jterrazz/station-api';

import { MemoryGrid, Panel, ShareBar, Tile } from '../../design/index';
import type { MemoryHolder } from '../../design/index';
import { useWatch } from '../../engine/react';
import { GIB, memory } from '../../model/format';
import { NotGivenTile, split, Unknown } from '../activity/tiles';
import { useIdentityOf } from '../kit/identity';
import type { IdentityOf } from '../kit/identity';
import { NotGiven } from '../kit/readings';

const MEMORY_FREE = 'Memory free';

function squares(bytes: number): number {
    return bytes / GIB;
}

function useRooms(): 'no-answer' | 'reading' | readonly MachinesRoom[] {
    const listing = useWatch('machines.list', {});
    if (listing.kind === 'reading') {
        return 'reading';
    }
    if (listing.kind === 'failed') {
        return 'no-answer';
    }
    return listing.out.Rooms ?? [];
}

export function MemoryFreeTile() {
    const rooms = useRooms();
    const identityOf = useIdentityOf();
    if (typeof rooms === 'string') {
        return <Unknown label={MEMORY_FREE} state={rooms} />;
    }
    if (rooms.length === 0) {
        return <NotGivenTile label={MEMORY_FREE} />;
    }
    const free = rooms.reduce((sum, room) => sum + room.Capacity.Free, 0);
    const total = rooms.reduce((sum, room) => sum + room.Capacity.Total, 0);
    const { value, unit } = split(memory(free));
    return (
        <Tile
            foot={`of ${split(memory(total)).value}`}
            label={MEMORY_FREE}
            unit={unit}
            value={value}
            viz={
                <ShareBar
                    label="Memory each machine holds, and what is free"
                    parts={[
                        ...rooms.flatMap((room) =>
                            (room.Capacity.VMs ?? []).map((vm) => ({
                                identity: identityOf(vm.Name),
                                key: `${room.Mac}/${vm.Name}`,
                                size: vm.Bytes,
                            })),
                        ),
                        { key: 'free', rest: true, size: free },
                    ]}
                />
            }
        />
    );
}

function beingCreated(macs: readonly MachinesMacView[], mac: string, vm: string): boolean {
    return macs.some(
        (view) =>
            view.Name === mac &&
            (view.VMs ?? []).some((held) => held.Name === vm && held.Birth !== null),
    );
}

function holdersOf(
    room: MachinesRoom,
    macs: readonly MachinesMacView[],
    identityOf: IdentityOf,
): MemoryHolder[] {
    return [
        {
            identity: 'system',
            key: 'macos',
            label: 'macOS',
            squares: squares(room.Capacity.MacOS),
        },
        ...(room.Capacity.VMs ?? []).map((vm) => ({
            identity: identityOf(vm.Name),
            key: vm.Name,
            label: vm.Name,
            pending: beingCreated(macs, room.Mac, vm.Name),
            squares: squares(vm.Bytes),
        })),
    ];
}

export function MemoryPanel({ macs, only }: { macs: readonly MachinesMacView[]; only?: string }) {
    const rooms = useRooms();
    const identityOf = useIdentityOf();
    if (rooms === 'reading') {
        return null;
    }
    const shown =
        rooms === 'no-answer'
            ? []
            : rooms.filter((room) => only === undefined || room.Mac === only);
    if (shown.length === 0) {
        return (
            <NotGiven
                title="Memory"
                what={
                    only === undefined
                        ? "What each machine holds, against each Mac's total"
                        : "What each machine holds of this Mac's memory."
                }
            />
        );
    }
    return (
        <Panel note="one square, one GB" padded title="Memory">
            {shown.map((room) => (
                <MemoryGrid
                    free={squares(room.Capacity.Free)}
                    holders={holdersOf(room, macs, identityOf)}
                    key={room.Mac}
                    name={room.Mac}
                    total={squares(room.Capacity.Total)}
                />
            ))}
        </Panel>
    );
}

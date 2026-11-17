import type { ActivityReport, MachinesMacView, MachinesRoom } from '@jterrazz/station-api';

import { useWatch } from '../../dependencies/dependencies';
import { useActivities } from '../../features/activity/reading';
import type { Heard } from '../../features/activity/reading';
import { useIdentityOf } from '../../kit/identity';
import { machineRoute } from '../../navigation/route';
import type { Route } from '../../navigation/route';
import { Button, FleetBoard } from '../../ui/index';
import { fleetOf } from './fleet-of';

type Go = (to: Route) => void;

function useRooms(): readonly MachinesRoom[] {
    const listing = useWatch('machines.list', {});
    return listing.kind === 'read' ? (listing.out.Rooms ?? []) : [];
}

function reportsOf(heard: ReadonlyMap<string, Heard>): ReadonlyMap<string, ActivityReport> {
    return new Map(
        [...heard].flatMap(([mac, one]) =>
            one.kind === 'read' && one.out.silent !== true ? [[mac, one.out] as const] : [],
        ),
    );
}

type HomeFleetProps = {
    macs: readonly MachinesMacView[];
    go: Go;
    onAddMac?: (() => void) | undefined;
};

export function HomeFleet({ macs, go, onAddMac }: HomeFleetProps) {
    const rooms = useRooms();
    const identityOf = useIdentityOf();
    const { readers, shown } = useActivities(macs, '1h');
    const fleet = fleetOf({ identityOf, macs, reports: reportsOf(shown), rooms });
    const open = (name: string) => {
        go(machineRoute(name));
    };
    return (
        <>
            {readers}
            <FleetBoard
                {...fleet}
                foot={
                    onAddMac === undefined ? undefined : (
                        <Button onClick={onAddMac} variant="link">
                            Add a Mac
                        </Button>
                    )
                }
                onOpen={open}
                onOpenMac={open}
            />
        </>
    );
}

import type {
    OverviewNeed,
    PortsListing,
    PortsMacPorts,
    PortsNotShared,
    PortsPort,
    PortsQuiet,
    PortsSharedPort,
} from '@jterrazz/station-api';

import { portsMacOf } from '../../engine/provisional';
import { sentence } from '../../model/format';
import { tabOf } from '../needs/needs';

export type PortsOf = {
    mac: PortsMacPorts | undefined;
    ports: readonly PortsPort[];
    shared: readonly PortsSharedPort[];
    notShared: readonly PortsNotShared[];
    quiet: readonly PortsQuiet[];
};

export type Asked = { mac: string; thisMac: boolean; vm?: string | undefined };

export function portsOf(listing: PortsListing, { mac: macName, thisMac, vm }: Asked): PortsOf {
    const mac = portsMacOf(listing, macName, thisMac);
    const mine = <T extends { vm: string }>(rows: null | readonly T[] | undefined): T[] =>
        vm === undefined ? [] : (rows ?? []).filter((row) => row.vm === vm);
    return {
        mac,
        notShared: mine(mac?.not_shared),
        ports: (mac?.ports ?? []).filter((port) =>
            vm === undefined ? port.Machine.On === '' : port.Machine.Name === vm,
        ),
        quiet: mine(mac?.quiet),
        shared: mine(mac?.shared),
    };
}

export function aboutPort(port: PortsPort): string {
    const declared = port.Declared;
    return sentence([
        (port.Binds ?? []).join(', '),
        declared ? sentence([`declared by ${declared.via}`, declared.by, declared.url]) : undefined,
        port.Verified?.Missing,
        port.PID > 0 ? `pid ${port.PID}` : undefined,
    ]);
}

export function needsOf(needs: readonly OverviewNeed[], name: string): OverviewNeed[] {
    return needs.filter((need) => tabOf(need) === 'Ports' && need.Machine === name);
}

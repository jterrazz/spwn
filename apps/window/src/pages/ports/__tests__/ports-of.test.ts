import type { PortsListing, PortsPort } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { portReach } from '../../../engine/shown';
import { aboutPort, lineOf, portsOf } from '../ports-of';

function port(name: string, number: number, on: string): PortsPort {
    return {
        Answer: '',
        Binds: ['0.0.0.0'],
        Declared: null,
        Device: false,
        Machine: { Name: name, On: on },
        PID: 0,
        Port: number,
        Process: 'node',
        Reach: 2,
        Verified: null,
    };
}

const LISTING: PortsListing = {
    all: {
        macs: [
            {
                engine: { running: true, version: 'v3' },
                https: '',
                name: 'mbp',
                not_shared: [{ count: 2, detail: 'the range is full', vm: 'workspace' }],
                ports: [port('', 5000, ''), port('workspace', 3000, 'mbp')],
                public_name: 'mbp',
                reachable: true,
                serve: '',
                shared: [
                    {
                        bind: '0.0.0.0',
                        detail: '',
                        guest_port: 3000,
                        http: true,
                        last_seen_at: null,
                        pinned: false,
                        process: 'node',
                        profile: '',
                        reserved: 40_012,
                        state: 'shared',
                        urls: { local: 'http://127.0.0.1:40012', public: '' },
                        vm: 'workspace',
                    },
                ],
                this_mac: true,
            },
        ],
        schema: 'station.ports.all/v1',
    },
    tailnet: {
        BackendState: '',
        CLI: '',
        Connected: false,
        DNSName: '',
        IP: '',
        Name: '',
        Peers: null,
        Serve: null,
        ServeOn: false,
        State: '',
        Variant: '',
    },
};

describe('ports of', () => {
    test("a Mac's tab holds its own ports, and none of its machines' shares", () => {
        // Given - the listing of a Mac with one workspace
        const of = portsOf(LISTING, { mac: 'mbp', thisMac: true });

        // Then - only port 5000 is its own; the share and the count belong to the workspace
        expect(of.ports.map((row) => row.Port)).toStrictEqual([5000]);
        expect(of.shared).toStrictEqual([]);
        expect(of.notShared).toStrictEqual([]);
    });

    test("a VM's tab holds its ports, its shares and its counts", () => {
        // Given - the same listing, read for the workspace
        const of = portsOf(LISTING, { mac: 'mbp', thisMac: true, vm: 'workspace' });

        // Then - the workspace's port, share and not-shared count
        expect(of.ports.map((row) => row.Port)).toStrictEqual([3000]);
        expect(of.shared.map((row) => row.guest_port)).toStrictEqual([3000]);
        expect(of.notShared.map((row) => row.count)).toStrictEqual([2]);
    });

    test("a VM's short-lived listeners are counted on its head line, never listed", () => {
        // Given - the workspace's ports, 214 of its listeners on ports the system keeps
        const of = portsOf(LISTING, { mac: 'mbp', thisMac: true, vm: 'workspace' });
        const quiet = { ...of, quiet: [{ count: 214, vm: 'workspace' }] };

        // Then - the line says the count after what it may share, and says nothing of it when there is none
        expect(lineOf(quiet, 'workspace')).toBe(
            'Its ports are shared from its Mac · 214 short-lived listeners not listed',
        );
        expect(lineOf(of, 'workspace')).toBe('Its ports are shared from its Mac');
    });

    test('a far Mac the listing does not name holds nothing', () => {
        // Given - the listing of mbp alone
        // Then - no Mac, no row
        expect(portsOf(LISTING, { mac: 'mini', thisMac: false }).mac).toBeUndefined();
    });

    test('this Mac, listed with no name, is found by its flag', () => {
        // Given - a listing whose This Mac came with an empty name
        const macs = (LISTING.all.macs ?? []).map((mac) => ({ ...mac, name: '' }));
        const nameless = { ...LISTING, all: { ...LISTING.all, macs } };

        // Then - This Mac is still read; a far Mac of that name is not
        expect(portsOf(nameless, { mac: 'mbp', thisMac: true }).ports).toHaveLength(1);
        expect(portsOf(nameless, { mac: 'mbp', thisMac: false }).mac).toBeUndefined();
    });

    test('a reach the words table lacks is said as the number the engine sent', () => {
        // Given - a port reached outside Tailscale, and one of a reach no table has
        // Then - 2 is worded, 7 is shown as the engine's own value, both plain
        expect(portReach(port('', 1, ''))).toStrictEqual({
            tone: 'plain',
            word: 'Outside Tailscale',
        });
        expect(portReach({ ...port('', 1, ''), Reach: 7 })).toStrictEqual({
            tone: 'plain',
            word: 'reach 7, not worded',
        });
    });

    test("a declared port says by what, and the process's pid", () => {
        // Given - a port declared by its setting, heard from pid 812
        const declared: PortsPort = {
            ...port('', 22, ''),
            Declared: { by: 'the sshd setting', machine: 'mini', port: 22, via: 'setting' },
            PID: 812,
        };

        // Then - its binds, its declaration and its pid
        expect(aboutPort(declared)).toBe(
            '0.0.0.0 · declared by setting · the sshd setting · pid 812',
        );
    });
});

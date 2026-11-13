import type { OverviewNeed } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { gestureOf, needsOn, tabOf } from '../needs';
import type { Held } from '../needs';

function need(Machine: string, change: Partial<OverviewNeed> = {}): OverviewNeed {
    return {
        Args: null,
        Broken: false,
        Call: '',
        Command: '',
        Machine,
        Path: '',
        Section: 'apps',
        Severity: 'needs-a-look',
        Subject: Machine,
        Text: `${Machine} needs a look`,
        ...change,
    };
}

function mac(Name: string, vms: readonly string[]): Held {
    return { Name, VMs: vms.map((vm) => ({ Name: vm })) };
}

const MACS = [mac('book', ['linux-1']), mac('mini', ['infra'])];
const NEEDS = [need('book'), need('linux-1'), need('infra')];

describe('needs', () => {
    test('all machines keeps every need in the engine order', () => {
        // Given - three needs on two Macs

        // When - all machines are chosen
        const kept = needsOn(NEEDS, MACS, null);

        // Then - nothing is dropped or moved
        expect(kept).toStrictEqual(NEEDS);
    });

    test('a Mac keeps its own needs and its machines', () => {
        // Given - three needs on two Macs

        // When - book is chosen
        const kept = needsOn(NEEDS, MACS, 'book');

        // Then - book and linux-1 stay, infra goes
        expect(kept.map((one) => one.Machine)).toStrictEqual(['book', 'linux-1']);
    });

    test('a Mac no longer listed keeps nothing', () => {
        // Given - three needs on two Macs

        // When - a Mac machines.list no longer names is chosen
        const kept = needsOn(NEEDS, MACS, 'gone');

        // Then - no need is shown under its name
        expect(kept).toStrictEqual([]);
    });

    test("a need opens the tab its section names, else the machine's Overview", () => {
        // Given - a need of ports, one of disk and one of apps

        // When - their tabs are read
        const tabs = [
            tabOf(need('book', { Section: 'ports' })),
            tabOf(need('book', { Section: 'disk' })),
            tabOf(need('book', { Section: 'apps' })),
        ];

        // Then - Ports, Storage, then Overview
        expect(tabs).toStrictEqual(['Ports', 'Storage', 'Overview']);
    });

    test('a call is the gesture, then a command, then a path', () => {
        // Given - a need with a call, one with a command, one with a path, one with none

        // When - their gestures are read
        const gestures = [
            gestureOf(need('book', { Call: 'apps.update', Command: 'brew upgrade' })),
            gestureOf(need('book', { Command: 'npm login' })),
            gestureOf(need('book', { Path: 'System Settings > Network' })),
            gestureOf(need('book')),
        ];

        // Then - each says what the engine gave, in that order of preference
        expect(gestures).toStrictEqual([
            { call: 'apps.update' },
            { text: 'npm login' },
            { text: 'System Settings > Network' },
            undefined,
        ]);
    });
});

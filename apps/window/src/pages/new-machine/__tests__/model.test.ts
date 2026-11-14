import { describe, expect, test } from 'vitest';

import type { MachinesCallsNewMachine } from '../../../engine/machines-new';
import { ANSWERS } from '../../../engine/machines-new-answers';
import {
    askedOf,
    chosenOnOf,
    EMPTY_DRAFT,
    GB,
    leavesFreeOf,
    memoryOf,
    serverInputOf,
    targetOfOn,
    workspaceInputOf,
} from '../model';

const PRO = 'jterrazz-macbook-pro';
const MINI = 'jterrazz-mac-mini';

function answerOn(mac: string): MachinesCallsNewMachine {
    const answer = ANSWERS.find((one) => one.fleet === `${PRO} ${MINI}`);
    if (answer === undefined) {
        throw new Error('two-macs has an answer');
    }
    return { ...answer.out, Mac: mac };
}

function onOf(answer: MachinesCallsNewMachine) {
    const on = chosenOnOf(answer);
    if (on === undefined) {
        throw new Error('the answer lists its Mac');
    }
    return on;
}

describe('model', () => {
    test('the Mac asked is the one picked, else the one the sheet was opened on, and a file opened from is kept', () => {
        // Given - a sheet opened on the mini from the station-linux file

        // When - it asks before and after This Mac is picked
        const opened = askedOf({ from: 'station-linux', on: MINI }, EMPTY_DRAFT);
        const picked = askedOf({ from: 'station-linux', on: MINI }, { ...EMPTY_DRAFT, on: PRO });

        // Then - the pick wins, and an opening with neither asks nothing
        expect(opened).toStrictEqual({ From: 'station-linux', On: MINI });
        expect(picked).toStrictEqual({ From: 'station-linux', On: PRO });
        expect(askedOf({}, EMPTY_DRAFT)).toStrictEqual({});
    });

    test('an untouched draft asks the default of each field, its machine file and lifetime given', () => {
        // Given - the answer on the mini, and the empty draft
        const answer = answerOn(MINI);
        const memory = memoryOf(answer, onOf(answer), EMPTY_DRAFT);

        // When - both inputs are made
        const workspace = workspaceInputOf({ answer, draft: EMPTY_DRAFT, memory });
        const server = serverInputOf({
            answer,
            draft: { ...EMPTY_DRAFT, kind: 'server' },
            memory: 8 * GB,
        });

        // Then - the name and file the engine names, no size, and a server carries its image and no lifetime
        expect(workspace).toStrictEqual({
            Request: { From: 'station-linux', Lifetime: 'persistent', Name: 'linux-2' },
        });
        expect(server).toStrictEqual({ Image: 'station-linux', Name: 'linux-2' });
    });

    test('a purpose and a memory off the default are sent, trimmed and in bytes', () => {
        // Given - a draft for the mini with a purpose and 8 GB
        const answer = answerOn(MINI);
        const draft = { ...EMPTY_DRAFT, memory: 8 * GB, name: ' gate5 ', purpose: ' Lot L6 ' };

        // When - its input is made
        const workspace = workspaceInputOf({
            answer,
            draft,
            memory: memoryOf(answer, onOf(answer), draft),
        });

        // Then - every field the owner gave is carried
        expect(workspace).toStrictEqual({
            Request: {
                From: 'station-linux',
                Lifetime: 'persistent',
                Name: 'gate5',
                Purpose: 'Lot L6',
                Size: { MemoryBytes: 8 * GB },
            },
        });
    });

    test('the memory asked stays within what the engine offers on the Mac', () => {
        // Given - This Mac, where at most 4 GB is offered
        const answer = answerOn(PRO);
        const on = onOf(answer);

        // When - 8 GB is wanted, and a server's default of 8 GB
        const wanted = memoryOf(answer, on, { ...EMPTY_DRAFT, memory: 8 * GB });
        const server = memoryOf(answer, on, { ...EMPTY_DRAFT, kind: 'server' });

        // Then - both are held to the 4 GB offered
        expect(wanted).toBe(4 * GB);
        expect(server).toBe(4 * GB);
    });

    test('the room left is the Mac free less the memory chosen, drawn only when it is not below zero', () => {
        // Given - a Mac with 10 GB free, this Mac or another
        const mini = onOf(answerOn(MINI));
        const room = mini.Room;
        if (room === null) {
            throw new Error('the mini measures its room');
        }
        const withFree = (free: number, ThisMac: boolean) => ({
            ...mini,
            Room: { ...room, Capacity: { ...room.Capacity, Free: free * GB } },
            ThisMac,
        });

        // When - 8 GB and 10 GB are chosen
        const here = leavesFreeOf(withFree(10, true), 8 * GB);
        const there = leavesFreeOf(withFree(10, false), 8 * GB);
        const exact = leavesFreeOf(withFree(10, true), 10 * GB);

        // Then - the rest is told by Mac
        expect(here).toBe('leaves 2 GB free on this Mac');
        expect(there).toBe(`leaves 2 GB free on ${mini.Mac}`);
        expect(exact).toBe('leaves 0 GB free on this Mac');
    });

    test('no room left is drawn when the size is over the free, unchosen, or the Mac measures no room', () => {
        // Given - a Mac with 10 GB free
        const mini = onOf(answerOn(MINI));
        const room = mini.Room;
        if (room === null) {
            throw new Error('the mini measures its room');
        }
        const tenFree = {
            ...mini,
            Room: { ...room, Capacity: { ...room.Capacity, Free: 10 * GB } },
        };

        // When - 12 GB is chosen, then nothing, then on a Mac with no room measured
        const over = leavesFreeOf(tenFree, 12 * GB);
        const unchosen = leavesFreeOf(tenFree, undefined);
        const unmeasured = leavesFreeOf({ ...mini, Room: null }, 8 * GB);

        // Then - nothing is drawn, never a negative
        expect([over, unchosen, unmeasured]).toStrictEqual(['', '', '']);
    });

    test('this Mac is asked with no target, another Mac by its name', () => {
        // Given - the answers on This Mac and on the mini

        // When - each is targeted
        const here = targetOfOn(onOf(answerOn(PRO)));
        const there = targetOfOn(onOf(answerOn(MINI)));

        // Then - only the mini is named
        expect(here).toBeUndefined();
        expect(there).toStrictEqual({ name: MINI });
    });
});

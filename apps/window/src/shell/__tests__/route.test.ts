import { describe, expect, test } from 'vitest';

import {
    goBack,
    goForward,
    historyAt,
    machineRoute,
    navigate,
    replace,
    startRoute,
} from '../route';
import type { History } from '../route';

const start: History = historyAt({ page: 'home' });

function startOf(query: string) {
    return startRoute(new URLSearchParams(query));
}

describe('route', () => {
    test('a navigation pushes the page it leaves', () => {
        // Given - the window on Home, opening a machine
        const history = navigate(start, machineRoute('workspace'));

        // Then - the machine is shown and Home is one step back
        expect(history).toStrictEqual({
            back: [{ page: 'home' }],
            current: machineRoute('workspace'),
            forward: [],
        });
    });

    test('a navigation to the page already shown pushes nothing', () => {
        // Given - the window on Home, asked for Home again
        const history = navigate(start, { page: 'home' });

        // Then - the history is the one it was
        expect(history).toBe(start);
    });

    test('a tab change is a navigation of its own', () => {
        // Given - a machine opened, then its Ports tab
        const history = navigate(
            navigate(start, machineRoute('gate')),
            machineRoute('gate', 'Ports'),
        );

        // Then - back returns to the machine's Overview
        expect(goBack(history).current).toStrictEqual(machineRoute('gate'));
    });

    test('back walks the stack in reverse and stops at its start', () => {
        // Given - Inbox then Storage opened from Home
        const history = navigate(navigate(start, { page: 'inbox' }), { page: 'storage' });

        // When - back is taken three times
        const once = goBack(history);
        const twice = goBack(once);

        // Then - Inbox, then Home, then Home again with nothing left behind it
        expect(once.current).toStrictEqual({ page: 'inbox' });
        expect(twice.current).toStrictEqual({ page: 'home' });
        expect(twice.back).toStrictEqual([]);
        expect(goBack(twice)).toBe(twice);
    });

    test('forward walks again what back left, until a navigation clears it', () => {
        // Given - Inbox then Storage opened from Home, then back taken twice
        const history = navigate(navigate(start, { page: 'inbox' }), { page: 'storage' });
        const home = goBack(goBack(history));

        // When - forward is taken once, then once more, then a new page is opened
        const once = goForward(home);
        const twice = goForward(once);
        const elsewhere = navigate(once, { page: 'catalog' });

        // Then - Inbox, then Storage, and a navigation leaves nothing ahead
        expect(once.current).toStrictEqual({ page: 'inbox' });
        expect(twice.current).toStrictEqual({ page: 'storage' });
        expect(goForward(twice)).toBe(twice);
        expect(elsewhere.forward).toStrictEqual([]);
    });

    test('a replace changes the place shown and pushes nothing', () => {
        // Given - a machine opened from Home
        const history = navigate(start, machineRoute('gone'));

        // When - the machine is replaced by Home
        const replaced = replace(history, { page: 'home' });

        // Then - Home is shown, with the same place behind it as before
        expect(replaced.current).toStrictEqual({ page: 'home' });
        expect(replaced.back).toStrictEqual(history.back);
    });

    test('the stack keeps the last forty places', () => {
        // Given - forty-five machines opened one after the other
        let history = start;
        for (let index = 0; index < 45; index += 1) {
            history = navigate(history, machineRoute(`vm${index}`));
        }

        // Then - the forty most recent places are kept, the oldest gone
        expect(history.back).toHaveLength(40);
        expect(history.back[0]).toStrictEqual(machineRoute('vm4'));
    });

    test('a start route names a page, or a machine and its tab, and falls back to Home', () => {
        // Given - the queries a capture opens the window with
        // Then - each opens where it names, and a name the window does not know opens Home or Overview
        expect(startOf('?page=inbox')).toStrictEqual({ page: 'inbox' });
        expect(startOf('?machine=linux-1&tab=Ports')).toStrictEqual(
            machineRoute('linux-1', 'Ports'),
        );
        expect(startOf('?machine=linux-1&tab=Nope')).toStrictEqual(machineRoute('linux-1'));
        expect(startOf('?page=nope')).toStrictEqual({ page: 'home' });
        expect(startOf('?page=machine-files&file=api')).toStrictEqual({
            file: 'api',
            page: 'machine-files',
        });
    });

    test('machine files opened on another file is a navigation of its own', () => {
        // Given - Machine files opened on This Mac's file
        const history = navigate(start, { page: 'machine-files' });

        // When - it is opened again on the file of api
        const other = navigate(history, { file: 'api', page: 'machine-files' });

        // Then - the file of api is shown, This Mac's one step back
        expect(other.current).toStrictEqual({ file: 'api', page: 'machine-files' });
        expect(goBack(other).current).toStrictEqual({ page: 'machine-files' });
    });
});

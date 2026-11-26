import { describe, expect, test } from 'vitest';

import { Opened } from '../opened';

describe('Opened', () => {
    test('opening a sheet shows it and tells its reader, and closing twice tells it once', () => {
        // Given - the window's sheets, nothing open, and a reader
        const opened = new Opened();
        let told = 0;
        opened.subscribe(() => {
            told += 1;
        });

        // When - New machine opens from a Mac's page, then the sheet is closed twice
        opened.newMachine({ on: 'mac-mini' });
        const shown = opened.shown();
        opened.close();
        opened.close();

        // Then - the sheet named the Mac, nothing is open now, and the reader heard the open and one close
        expect(shown).toStrictEqual({ kind: 'new-machine', opening: { on: 'mac-mini' } });
        expect(opened.shown()).toStrictEqual({ kind: 'none' });
        expect(told).toBe(2);
    });
});

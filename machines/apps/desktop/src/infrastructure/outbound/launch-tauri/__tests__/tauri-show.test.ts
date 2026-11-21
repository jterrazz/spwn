import { describe, expect, test } from 'vitest';

import { NO_SHOW, tauriShow } from '../tauri-show';

describe('tauri show', () => {
    test('the shell is asked to show its window by its one command, with no argument', async () => {
        // Given - the shell's invoke
        const invoked: { command: string; args: Record<string, unknown> | undefined }[] = [];
        const ipc = {
            invoke: async (command: string, args?: Record<string, unknown>) => {
                invoked.push({ args, command });
                return await Promise.resolve(null);
            },
        };

        // When - the window is shown
        await tauriShow(ipc)();

        // Then - window_ready, alone
        expect(invoked).toStrictEqual([{ args: undefined, command: 'window_ready' }]);
    });

    test('outside the Tauri shell showing asks nothing and settles', async () => {
        // Given - a proof build's show

        // When - the window is shown
        const shown = NO_SHOW();

        // Then - it settles
        await expect(shown).resolves.toBeUndefined();
    });
});

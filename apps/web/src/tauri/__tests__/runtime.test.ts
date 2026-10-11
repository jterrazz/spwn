import { afterEach, describe, expect, test, vi } from 'vitest';

import type { TauriBridge } from '../runtime';

async function freshRuntime(bridge: TauriBridge | undefined) {
    vi.resetModules();
    globalThis.__TAURI__ = bridge;
    return await import('../runtime');
}

function answering(...ports: unknown[]) {
    const invoke = vi.fn<(command: string) => Promise<unknown>>();
    for (const port of ports) {
        invoke.mockResolvedValueOnce(port);
    }
    return { core: { invoke } };
}

afterEach(() => {
    globalThis.__TAURI__ = undefined;
});

describe('runtime', () => {
    test('a browser has no bridge, no port and no API base', async () => {
        // Given - a page with no Tauri bridge
        const runtime = await freshRuntime(undefined);

        // Then - it is no Tauri, and the port and the base are none
        expect([
            runtime.isTauri(),
            runtime.tauriBridge(),
            await runtime.initTauriApiPort(),
            runtime.getTauriApiBase(),
        ]).toStrictEqual([false, undefined, null, null]);
    });

    test('the desktop shell is asked for its port once, then the answer is cached', async () => {
        // Given - a shell answering port 4321
        const bridge = answering(4321, 9999);
        const runtime = await freshRuntime(bridge);

        // When - the port is asked twice
        const base = runtime.getTauriApiBase();
        const first = await runtime.initTauriApiPort();
        const second = await runtime.initTauriApiPort();

        // Then - no base before the answer, the port after it, and the shell was asked once for it
        expect([runtime.isTauri(), base, first, second, runtime.getTauriApiBase()]).toStrictEqual([
            true,
            null,
            4321,
            4321,
            'http://localhost:4321',
        ]);
        expect(bridge.core.invoke.mock.calls).toStrictEqual([['get_api_port']]);
    });

    test('a shell that answers no port, port zero, or fails leaves no base, and is not asked again', async () => {
        // Given - three shells: a text answer, a zero, and a refusal
        const results = [];
        for (const answer of ['4321', 0, new Error('no command')]) {
            const invoke = vi.fn<(command: string) => Promise<unknown>>(async () => {
                if (answer instanceof Error) {
                    throw answer;
                }
                return answer;
            });
            const runtime = await freshRuntime({ core: { invoke } });

            // When - the port is asked twice
            const first = await runtime.initTauriApiPort();
            const second = await runtime.initTauriApiPort();
            results.push([first, second, runtime.getTauriApiBase(), invoke.mock.calls.length]);
        }

        // Then - none, or the zero itself, and no base; one ask each
        expect(results).toStrictEqual([
            [null, null, null, 1],
            [0, 0, null, 1],
            [null, null, null, 1],
        ]);
    });

    test('a shell without the core bridge answers no port', async () => {
        // Given - a bridge that enables no core commands
        const runtime = await freshRuntime({});

        // Then - the port is none
        await expect(runtime.initTauriApiPort()).resolves.toBeNull();
    });
});

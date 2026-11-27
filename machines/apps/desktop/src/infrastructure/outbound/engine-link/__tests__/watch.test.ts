import { clock } from '@jterrazz/test';
import { describe, expect, test, vi } from 'vitest';

import type { EngineError } from '../../../../domain/engine-error';
import { Client } from '../client';
import { fakeTransport, last } from './fake-transport';

const BROKE = { code: 'broke', message: 'the engine broke', type: 'internal' } as const;

describe('watch', () => {
    test('a watch the engine broke on is opened again after 1 s, and resumes', async () => {
        // Given - a watch of machines.list whose handler broke
        using _ = clock.run('2026-11-12T10:00:00Z');
        const { opened, transport } = fakeTransport();
        const errors: EngineError[] = [];
        const outs: unknown[] = [];
        new Client(transport).watch(
            'machines.list',
            {},
            {
                error: (error) => {
                    errors.push(error);
                },
                out: (out) => {
                    outs.push(out);
                },
            },
        );
        last(opened).receiver.frame({ error: BROKE, id: last(opened).request.id, v: 2 });

        // When - 999 ms pass, then 1 ms, and the new stream answers
        await clock.advance(999);
        const before = opened.length;
        await clock.advance(1);
        const after = opened.length;
        last(opened).receiver.frame({ id: last(opened).request.id, out: { Rows: [] }, v: 2 });

        // Then - it was opened again at 1 s, the break was told, and the new stream's out reached the page
        expect([before, after]).toStrictEqual([1, 2]);
        expect(errors.map((error) => error.wire?.code)).toStrictEqual(['broke']);
        expect(outs).toStrictEqual([{ Rows: [] }]);
    });

    test('a watch the engine broke on waits 2 s on a second break with no out, and 1 s again after an out', async () => {
        // Given - a watch that broke twice with no out between, then spoke
        using _ = clock.run('2026-11-12T10:00:00Z');
        const { opened, transport } = fakeTransport();
        new Client(transport).watch(
            'machines.list',
            {},
            { error: vi.fn<(error: EngineError) => void>(), out: vi.fn<(out: unknown) => void>() },
        );
        const breakIt = () => {
            last(opened).receiver.frame({ error: BROKE, id: last(opened).request.id, v: 2 });
        };
        breakIt();
        await clock.advance(1000);
        breakIt();
        await clock.advance(1999);
        const beforeThird = opened.length;
        await clock.advance(1);
        last(opened).receiver.frame({ id: last(opened).request.id, out: {}, v: 2 });

        // When - it breaks after an out
        breakIt();
        await clock.advance(999);
        const beforeFourth = opened.length;
        await clock.advance(1);

        // Then - the second wait doubled, the one after an out started over at 1 s
        expect([beforeThird, beforeFourth, opened.length]).toStrictEqual([2, 3, 4]);
    });
});

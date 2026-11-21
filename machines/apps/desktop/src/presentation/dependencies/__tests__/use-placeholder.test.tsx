import { component, content, waitUntil } from '@jterrazz/test';
import { useState } from 'react';
import { describe, expect, test } from 'vitest';

import { EngineError } from '../../../domain/engine-error';
import type { Reading } from '../dependencies';
import { usePlaceholder } from '../use-placeholder';

const AT = new Date('2026-11-13T10:00:00Z');
const ONE_HOUR: Reading<string> = { at: AT, done: true, kind: 'read', out: 'one hour' };
const ONE_DAY: Reading<string> = { at: AT, done: true, kind: 'read', out: 'one day' };
const READING: Reading<string> = { kind: 'reading' };
const REFUSED: Reading<string> = {
    error: new EngineError(undefined),
    kind: 'failed',
};

function Span({ readings, placeholder }: { readings: Reading<string>[]; placeholder: boolean }) {
    const [at, setAt] = useState(0);
    const reading = usePlaceholder(readings[at] ?? READING, { placeholder });
    const said =
        reading.kind === 'read'
            ? `${reading.out}${reading.placeholder ? ', held' : ''}`
            : reading.kind;
    return (
        <button
            onClick={() => {
                setAt(at + 1);
            }}
            type="button"
        >
            {said}
        </button>
    );
}

function shown(): string {
    return document.querySelector('button')?.textContent ?? '';
}

async function steps(readings: Reading<string>[], placeholder: boolean): Promise<string[]> {
    const said: string[] = [];
    await component.render(
        <Span placeholder={placeholder} readings={readings} />,
        async (visitor) => {
            await waitUntil(() => shown() !== '', { why: 'the span drawn' });
            said.push(shown());
            while (said.length < readings.length) {
                const before = shown();
                await visitor.click(content(before));
                await waitUntil(() => shown() !== before, { why: 'the next span drawn' });
                said.push(shown());
            }
        },
    );
    return said;
}

describe('use-placeholder', () => {
    test('an input asked anew shows the last answer marked as a placeholder until its own answer comes', async () => {
        // Given - one hour answered, a new span read, then answered, then a span read and refused

        // When - the hook asks with a placeholder through each
        const said = await steps([ONE_HOUR, READING, ONE_DAY, READING, REFUSED], true);

        // Then - the last answer stood in while each was read, and the refusal is shown as it is
        expect(said).toStrictEqual([
            'one hour',
            'one hour, held',
            'one day',
            'one day, held',
            'failed',
        ]);
    });

    test('a hook that asks for no placeholder shows the reading as it is', async () => {
        // Given - one hour answered, then a new span read

        // When - the hook asks with no placeholder
        const said = await steps([ONE_HOUR, READING], false);

        // Then - the new span reads as reading
        expect(said).toStrictEqual(['one hour', 'reading']);
    });
});

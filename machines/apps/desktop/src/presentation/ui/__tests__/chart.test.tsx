import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Chart } from '../chart';

const PEAKING = [40, 70, 99];

describe('Chart', () => {
    test('a chart the caller gives no tone stays neutral, however high it reads', async () => {
        // Given - a CPU chart whose last mark is 99
        const result = await component.render(<Chart label="CPU" values={PEAKING} />);

        // Then - its height alone warns of nothing
        expect(result.html.text).not.toContain('_warn_');
        expect(result.errors.text).toBe('');
    });

    test('a chart takes the tone its caller reads from the engine', async () => {
        // Given - the same chart, toned warn by its caller
        const result = await component.render(<Chart label="CPU" tone="warn" values={PEAKING} />);

        // Then - it wears that tone
        expect(result.html.text).toContain('_warn_');
        expect(result.errors.text).toBe('');
    });
});

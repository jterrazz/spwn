import { component, disabled, enabled, field, focused } from '@jterrazz/test';
import { describe, expect, test, vi } from 'vitest';

import { RadioRows } from '../radio-rows';

const ROWS = [
    { id: 'linux-1', line: 'Signed out here', title: 'linux-1' },
    { held: true, id: 'api', line: 'Start the machine first', title: 'api' },
    { id: 'linux-2', line: 'Signed out here', title: 'linux-2' },
];

function ignored(): void {
    return undefined;
}

describe('RadioRows', () => {
    test('a row that can be chosen is a radio, and a held one a disabled radio that still says why', async () => {
        // Given - two rows to choose and one held
        const result = await component.render(
            <RadioRows label="Machines" onChange={ignored} rows={ROWS} value={null} />,
            async (visitor) => {
                await visitor.see(enabled(field('linux-1')));
            },
        );

        // Then - one radio group, the held row's radio disabled and described by its reason
        expect(result.html.text).toMatch(/role="radiogroup"/u);
        expect(result.html.text.match(/type="radio"/gu)).toHaveLength(3);
        expect(result.html.text).toMatch(/aria-describedby="[^"]*api-line"[^>]*disabled=""/u);
        expect(result.html.text).toContain('Start the machine first');
    });

    test('the arrows move the choice between the rows that can be chosen, past a held one', async () => {
        // Given - nothing chosen, the focus on the first row
        const onChange = vi.fn<(value: string) => void>();
        await component.render(
            <RadioRows label="Machines" onChange={onChange} rows={ROWS} value={null} />,
            async (visitor) => {
                await visitor.see(disabled(field('api')));
                await visitor.press('Tab');
                await visitor.see(focused(field('linux-1')));

                // When - the down arrow is pressed
                await visitor.press('ArrowDown');

                // Then - the focus is on linux-2, past the held api
                await visitor.see(focused(field('linux-2')));
            },
        );

        // Then - linux-2 is the one chosen
        expect(onChange).toHaveBeenLastCalledWith('linux-2');
    });
});

import { button, component, dialog, field, focused } from '@jterrazz/test';
import { useState } from 'react';
import { describe, expect, test, vi } from 'vitest';

import { Popover, usePopover } from '../popover';
import type { PopoverRow } from '../popover';
import { MACHINES } from './popover.fixtures';

const SECTIONS: readonly PopoverRow[] = [
    { id: 'home', label: 'Home' },
    { current: true, id: 'storage', label: 'Storage' },
    { id: 'engine', label: 'Engine' },
];

const FILTER = field('Go to');

// value: the card's form: two sections side by side in one listbox, under one filter for both.
function Card({ onChoose }: { onChoose: (id: string) => void }) {
    const { anchor, close, id, open, opener } = usePopover();
    const [query, setQuery] = useState('');
    const kept = (rows: readonly PopoverRow[]) => rows.filter((row) => row.label.includes(query));
    return (
        <>
            <button type="button" {...opener}>
                Storage
            </button>
            {open ? (
                <Popover
                    anchor={anchor}
                    columns
                    empty="Nothing matches"
                    filter={{
                        label: 'Go to',
                        onChange: setQuery,
                        placeholder: 'Go to',
                        value: query,
                    }}
                    id={id}
                    label="Storage"
                    onChoose={onChoose}
                    onClose={close}
                    sections={[
                        { empty: 'No section matches', label: 'Sections', rows: kept(SECTIONS) },
                        {
                            empty: 'No machine matches',
                            label: 'Fleet',
                            meta: '6 machines',
                            rows: kept(MACHINES),
                        },
                    ]}
                />
            ) : null}
        </>
    );
}

// value: what each column says: its rows' names, or the sentence in their place; the list's one sentence when
// value: no column is drawn.
function columnsSaid(): string[] {
    const list = document.querySelector('[role="listbox"]');
    const groups = [...(list?.querySelectorAll('[role="group"]') ?? [])];
    if (groups.length === 0) {
        return [list?.textContent ?? ''];
    }
    return groups.map((group) => {
        const rows = [...group.querySelectorAll('[role="option"] [class*="label"]')].map(
            (label) => label.textContent,
        );
        const said =
            rows.length > 0 ? rows.join(', ') : (group.lastElementChild?.textContent ?? '');
        return `${String(group.getAttribute('aria-label'))}: ${said}`;
    });
}

describe('PopoverList', () => {
    test('in columns, ↓ crosses from the first section into the next, and typing starts again at the first row', async () => {
        // Given - the card's form, opened on Storage
        const onChoose = vi.fn<(id: string) => void>();
        await component.render(<Card onChoose={onChoose} />, async (visitor) => {
            await visitor.click(button('Storage'));
            await visitor.see(focused(FILTER));

            // When - ↓ twice from Storage, past Engine into the next section, and Enter
            await visitor.press('ArrowDown');
            await visitor.press('ArrowDown');
            await visitor.press('Enter');
            await visitor.gone(dialog('Storage'));

            // When - it opens again, o is typed while Storage still shows, and Enter
            await visitor.click(button('Storage'));
            await visitor.fill(FILTER, 'o');
            await visitor.press('Enter');
        });

        // Then - the first Mac, then the first row o keeps, Home, not the current Storage
        expect(onChoose.mock.calls).toStrictEqual([['macbook-pro'], ['home']]);
    });

    test('in columns, a filter that empties one section says so in its place, and both emptied say it once', async () => {
        // Given - the card's form, open
        let said: string[] = [];
        await component.render(
            <Card onChoose={vi.fn<(id: string) => void>()} />,
            async (visitor) => {
                await visitor.click(button('Storage'));

                // When - lin keeps machines alone, then zz keeps nothing
                await visitor.fill(FILTER, 'lin');
                const machines = columnsSaid();
                await visitor.fill(FILTER, 'zz');
                said = [...machines, ...columnsSaid()];
            },
        );

        // Then - Sections says no section matches beside the machines kept; then one sentence
        expect(said).toStrictEqual([
            'Sections: No section matches',
            'Fleet: linux-1, linux-2, linux-3',
            'Nothing matches',
        ]);
    });
});

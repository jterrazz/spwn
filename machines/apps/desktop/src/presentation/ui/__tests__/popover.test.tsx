import { button, component, dialog, field, file, focused, option } from '@jterrazz/test';
import { useState } from 'react';
import { describe, expect, test, vi } from 'vitest';

import { Screen } from '../button';
import { Popover, usePopover } from '../popover';
import type { PopoverRow } from '../popover';
import { POPOVER_STORIES, PopoverSample } from '../popover.stories';
import { MACHINES } from './popover.fixtures';

const SPACES: readonly PopoverRow[] = [
    { current: true, id: 'machine', label: 'Machine' },
    { description: 'Soon', disabled: true, id: 'agent', label: 'Agent' },
    { description: 'Soon', disabled: true, id: 'os', label: 'OS' },
];

type Heard = { onChoose: (id: string) => void; onClose: () => void };

function Machines({ onChoose, onClose }: Heard) {
    const { anchor, close, id, open, opener } = usePopover();
    const [query, setQuery] = useState('');
    return (
        <>
            <button type="button" {...opener}>
                linux-1
            </button>
            <button type="button">Elsewhere</button>
            {open ? (
                <Popover
                    anchor={anchor}
                    empty="No machine matches"
                    filter={{
                        label: 'Filter machines',
                        onChange: setQuery,
                        placeholder: 'Filter machines',
                        value: query,
                    }}
                    id={id}
                    label="Machines"
                    onChoose={onChoose}
                    onClose={() => {
                        onClose();
                        close();
                    }}
                    sections={[
                        {
                            label: 'Machines',
                            rows: MACHINES.filter((row) => row.label.includes(query)),
                        },
                    ]}
                />
            ) : null}
        </>
    );
}

function Spaces({ onChoose }: Pick<Heard, 'onChoose'>) {
    const { anchor, close, id, open, opener } = usePopover();
    return (
        <>
            <button type="button" {...opener}>
                Machine
            </button>
            {open ? (
                <Popover
                    anchor={anchor}
                    empty="No space"
                    id={id}
                    label="Spaces"
                    onChoose={onChoose}
                    onClose={close}
                    sections={[{ label: 'Spaces', rows: SPACES }]}
                />
            ) : null}
        </>
    );
}

function found(selector: string): HTMLElement {
    const element = document.querySelector<HTMLElement>(selector);
    if (element === null) {
        throw new Error(`nothing drawn matches ${selector}`);
    }
    return element;
}

const GOLDENS = [
    './__expected__/popover-machines.aria.yaml',
    './__expected__/popover-filtered.aria.yaml',
    './__expected__/popover-no-match.aria.yaml',
    './__expected__/popover-long-list.aria.yaml',
    './__expected__/popover-spaces.aria.yaml',
    './__expected__/popover-window-edge.aria.yaml',
    './__expected__/popover-card.aria.yaml',
] as const;

const DRAWN = GOLDENS.map((golden, at) => {
    const story = POPOVER_STORIES[at];
    if (story === undefined) {
        throw new Error(`no story for ${golden}`);
    }
    return { golden, name: story.name, props: story.props };
});

describe('Popover', () => {
    test.each(DRAWN)(
        '$name: a dialog over a listbox, its rows grouped by section',
        async (story) => {
            // Given - the story's popover, open under its opener
            const result = await component.render(
                <Screen>
                    <PopoverSample {...story.props} />
                </Screen>,
                async (visitor) => {
                    // When - it has drawn
                    await visitor.see(dialog(story.props.popover.label));
                },
            );

            // Then - its accessibility tree is the story's
            await expect(result.tree).toMatch(file(story.golden));
        },
    );

    test('a press opens it on its filter and its current row, the arrows move along it, Enter chooses and returns the keyboard', async () => {
        // Given - the machine list's opener
        const onChoose = vi.fn<(id: string) => void>();
        await component.render(
            <Machines onChoose={onChoose} onClose={vi.fn<() => void>()} />,
            async (visitor) => {
                // When - it is pressed, the arrows move down once then up twice from linux-1, and Enter is pressed
                await visitor.click(button('linux-1'));
                await visitor.see(focused(field('Filter machines')));
                await visitor.press('ArrowDown');
                await visitor.press('ArrowUp');
                await visitor.press('ArrowUp');
                await visitor.press('Enter');

                // Then - it is gone and the keyboard is on its opener again
                await visitor.gone(dialog('Machines'));
                await visitor.see(focused(button('linux-1')));
            },
        );

        // Then - the row above the current one was chosen, once
        expect(onChoose.mock.calls).toStrictEqual([['api']]);
    });

    test('a filter that hides the active row starts it again at its first match, End stays the caret, ArrowUp wraps', async () => {
        // Given - the machine list, open, its active row moved to api
        const onChoose = vi.fn<(id: string) => void>();
        await component.render(
            <Machines onChoose={onChoose} onClose={vi.fn<() => void>()} />,
            async (visitor) => {
                await visitor.click(button('linux-1'));
                await visitor.see(focused(field('Filter machines')));
                await visitor.press('ArrowUp');

                // When - a filter hides api, then End, ArrowUp and Enter are pressed
                await visitor.fill(field('Filter machines'), 'lin');
                await visitor.press('End');
                await visitor.press('ArrowUp');
                await visitor.press('Enter');

                // Then - it is gone
                await visitor.gone(dialog('Machines'));
            },
        );

        // Then - from linux-1, the first match, ArrowUp wrapped to the last one
        expect(onChoose.mock.calls).toStrictEqual([['linux-3']]);
    });

    test('pressing Escape or Tab closes it and returns the keyboard to its opener, and ArrowDown opens it again', async () => {
        // Given - the machine list, open
        const onClose = vi.fn<() => void>();
        await component.render(
            <Machines onChoose={vi.fn<(id: string) => void>()} onClose={onClose} />,
            async (visitor) => {
                await visitor.click(button('linux-1'));
                await visitor.see(focused(field('Filter machines')));

                // When - Escape is pressed
                await visitor.press('Escape');

                // Then - it is gone, and its opener holds the keyboard
                await visitor.gone(dialog('Machines'));
                await visitor.see(focused(button('linux-1')));

                // When - ArrowDown is pressed on the opener, then Tab
                await visitor.press('ArrowDown');
                await visitor.see(focused(field('Filter machines')));
                await visitor.press('Tab');

                // Then - Tab closed it the same way
                await visitor.gone(dialog('Machines'));
                await visitor.see(focused(button('linux-1')));
            },
        );

        // Then - each dismissal was heard once
        expect(onClose).toHaveBeenCalledTimes(2);
    });

    test('a press outside closes it and leaves the keyboard where the press put it; a press on its opener toggles it', async () => {
        // Given - the machine list, open
        const onClose = vi.fn<() => void>();
        await component.render(
            <Machines onChoose={vi.fn<(id: string) => void>()} onClose={onClose} />,
            async (visitor) => {
                await visitor.click(button('linux-1'));
                await visitor.see(dialog('Machines'));

                // When - another button is pressed
                await visitor.click(button('Elsewhere'));

                // Then - it is gone and the pressed button keeps the keyboard
                await visitor.gone(dialog('Machines'));
                await visitor.see(focused(button('Elsewhere')));

                // When - its opener is pressed twice
                await visitor.click(button('linux-1'));
                await visitor.see(dialog('Machines'));
                await visitor.click(button('linux-1'));

                // Then - the second press closed it
                await visitor.gone(dialog('Machines'));
            },
        );

        // Then - the press outside was heard as a dismissal; the opener closed it itself
        expect(onClose).toHaveBeenCalledOnce();
    });

    test('the focus leaving it closes it', async () => {
        // Given - the machine list, open
        const onClose = vi.fn<() => void>();
        await component.render(
            <Machines onChoose={vi.fn<(id: string) => void>()} onClose={onClose} />,
            async (visitor) => {
                await visitor.click(button('linux-1'));
                await visitor.see(focused(field('Filter machines')));

                // When - the keyboard is moved elsewhere by the window, as a shortcut opening a page does
                found('button:not([aria-haspopup])').focus();

                // Then - it is gone, the keyboard left where it was put
                await visitor.gone(dialog('Machines'));
                await visitor.see(focused(button('Elsewhere')));
            },
        );

        // Then - its owner heard it close
        expect(onClose).toHaveBeenCalledOnce();
    });

    test('a press on a row chooses it', async () => {
        // Given - the machine list, open
        const onChoose = vi.fn<(id: string) => void>();
        await component.render(
            <Machines onChoose={onChoose} onClose={vi.fn<() => void>()} />,
            async (visitor) => {
                await visitor.click(button('linux-1'));
                await visitor.see(dialog('Machines'));

                // When - a row is pressed
                await visitor.click(option('linux-2', { exact: false }));

                // Then - it is gone, its row chosen
                await visitor.gone(dialog('Machines'));
            },
        );

        // Then - the pressed row was the one chosen
        expect(onChoose.mock.calls).toStrictEqual([['linux-2']]);
    });

    test('with no filter the list holds the keyboard, and a greyed row is reached but never chosen', async () => {
        // Given - the spaces, opened from their segment
        const onChoose = vi.fn<(id: string) => void>();
        await component.render(<Spaces onChoose={onChoose} />, async (visitor) => {
            await visitor.click(button('Machine'));
            await visitor.see(dialog('Spaces'));

            // When - End moves to OS, greyed, and Enter is pressed on it
            await visitor.press('End');
            await visitor.press('Enter');

            // Then - it stays open, nothing chosen
            await visitor.see(dialog('Spaces'));

            // When - Home returns to Machine and Enter is pressed
            await visitor.press('Home');
            await visitor.press('Enter');

            // Then - Machine was chosen and the popover closed
            await visitor.gone(dialog('Spaces'));
        });

        // Then - only the open space was ever chosen
        expect(onChoose.mock.calls).toStrictEqual([['machine']]);
    });
});

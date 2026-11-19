import { component, content, file } from '@jterrazz/test';
import { useState } from 'react';
import { describe, expect, test, vi } from 'vitest';

import type { MemoryHolder } from '../memory-grid';
import { MemoryPick } from '../memory-pick';

const ROOM = 10;

const HOLDERS: readonly MemoryHolder[] = [
    { identity: 'system', key: 'macos', label: 'macOS', squares: 4 },
    { identity: 'blue', key: 'infra', label: 'infra', squares: 12 },
];

type Taken = (squares: number) => void;

function Host({ max, onTake }: { max: number; onTake: Taken }) {
    const [chosen, setChosen] = useState(4);
    return (
        <MemoryPick
            chosen={chosen}
            free={ROOM - chosen}
            holders={HOLDERS}
            identity="purple"
            line={`leaves ${String(ROOM - chosen)} GB free on this Mac`}
            name="mac-mini"
            take={{
                max,
                min: 2,
                onTake: (squares) => {
                    onTake(squares);
                    setChosen(squares);
                },
            }}
        />
    );
}

function found(selector: string): HTMLElement {
    const element = document.querySelector<HTMLElement>(selector);
    if (element === null) {
        throw new Error(`nothing drawn matches ${selector}`);
    }
    return element;
}

function grid(): HTMLElement {
    return found('[role="img"]');
}

function square(at: number): HTMLElement {
    return found(`[data-take="${String(at)}"]`);
}

type Point = { x: number; y: number };

function centreOf(element: HTMLElement): Point {
    const box = element.getBoundingClientRect();
    return { x: box.left + box.width / 2, y: box.top + box.height / 2 };
}

function pointer(type: string, on: HTMLElement, at: Point): void {
    on.dispatchEvent(
        new PointerEvent(type, {
            bubbles: true,
            buttons: type === 'pointerup' ? 0 : 1,
            cancelable: true,
            clientX: at.x,
            clientY: at.y,
            isPrimary: true,
            pointerId: 1,
            pointerType: 'mouse',
        }),
    );
}

function drag(from: HTMLElement, through: readonly Point[]): void {
    pointer('pointerdown', from, centreOf(from));
    for (const at of through) {
        pointer('pointermove', grid(), at);
    }
    pointer('pointerup', grid(), through.at(-1) ?? centreOf(from));
}

describe('MemoryPick', () => {
    test('a drag across the free squares takes the square under the pointer, live, and the release keeps it', async () => {
        // Given - a Mac with 10 GB free, 4 of them chosen
        const onTake = vi.fn<Taken>();
        const result = await component.render(
            <Host max={ROOM} onTake={onTake} />,
            async (visitor) => {
                await visitor.see(content('leaves 6 GB free on this Mac'));

                // When - the pointer goes down on the sixth free square, moves over the eighth, then the seventh, and is released
                drag(square(6), [centreOf(square(8)), centreOf(square(7))]);

                // Then - the figure and what it leaves follow the square last under the pointer
                await visitor.see(content('leaves 3 GB free on this Mac'));
            },
        );

        // Then - each square under the pointer was taken in turn, and seven squares are lit among those held
        expect(onTake.mock.calls).toStrictEqual([[6], [8], [7]]);
        expect(result.html.text.match(/data-identity="purple"/gu)).toHaveLength(7);
        await expect(result.tree).toMatch(file('./__expected__/memory-pick-dragged.aria.yaml'));
    });

    test('a drag past what can be chosen stops at it, and a pointer that leaves the grid keeps the last square', async () => {
        // Given - a Mac with 10 GB free, of which 8 can be chosen
        const onTake = vi.fn<Taken>();
        await component.render(<Host max={8} onTake={onTake} />, async (visitor) => {
            await visitor.see(content('leaves 6 GB free on this Mac'));

            // When - the pointer goes down on the fifth square, moves over the tenth, then far outside the grid
            const outside = { x: grid().getBoundingClientRect().right + 200, y: 1 };
            drag(square(5), [centreOf(square(10)), outside]);

            // Then - the size stops at 8 GB
            await visitor.see(content('leaves 2 GB free on this Mac'));
        });

        // Then - the tenth square was taken as eight, and the move outside took nothing
        expect(onTake.mock.calls).toStrictEqual([[5], [8]]);
    });

    test('a drag that starts on a held square takes nothing', async () => {
        // Given - a Mac with 10 GB free, 4 of them chosen
        const onTake = vi.fn<Taken>();
        await component.render(<Host max={ROOM} onTake={onTake} />, async (visitor) => {
            await visitor.see(content('leaves 6 GB free on this Mac'));

            // When - the pointer goes down on a square infra holds and moves over the ninth free one
            drag(found('[data-identity="blue"]'), [centreOf(square(9))]);

            // Then - the size is still 4 GB
            await visitor.see(content('leaves 6 GB free on this Mac'));
        });

        // Then - nothing was taken
        expect(onTake.mock.calls).toStrictEqual([]);
    });

    test('a click on a free square takes it, and one below the minimum takes the minimum', async () => {
        // Given - a Mac with 10 GB free, 4 of them chosen, at least 2 to choose
        const onTake = vi.fn<Taken>();
        await component.render(<Host max={ROOM} onTake={onTake} />, async (visitor) => {
            await visitor.see(content('leaves 6 GB free on this Mac'));

            // When - the ninth square is clicked
            drag(square(9), []);

            // Then - the size is 9 GB
            await visitor.see(content('leaves 1 GB free on this Mac'));

            // When - the first square is clicked
            drag(square(1), []);

            // Then - the size is the minimum, 2 GB
            await visitor.see(content('leaves 8 GB free on this Mac'));
        });

        // Then - the two squares were taken, the first raised to the minimum
        expect(onTake.mock.calls).toStrictEqual([[9], [2]]);
    });

    test('with nothing to take, the squares are drawn and no square can be taken', async () => {
        // Given - a Mac's squares drawn with no take
        const result = await component.render(
            <MemoryPick chosen={4} free={6} holders={HOLDERS} line="" name="mac-mini" />,
        );

        // Then - no square is takeable
        expect(result.html.text).not.toContain('data-take');
    });
});

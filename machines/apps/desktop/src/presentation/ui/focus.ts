export type Move = 'first' | 'last' | 'next' | 'previous';

export function moved(count: number, index: number, move: Move, wrap: boolean): number {
    if (count === 0) {
        return -1;
    }
    const last = count - 1;
    switch (move) {
        case 'first': {
            return 0;
        }
        case 'last': {
            return last;
        }
        case 'next': {
            if (index < 0 || index >= last) {
                return index < 0 || wrap ? 0 : last;
            }
            return index + 1;
        }
        case 'previous': {
            if (index <= 0) {
                return index < 0 || wrap ? last : 0;
            }
            return index - 1;
        }
    }
}

const ARROWS: Readonly<Record<string, Move>> = {
    ArrowDown: 'next',
    ArrowUp: 'previous',
    End: 'last',
    Home: 'first',
};

const SIDEWAYS: Readonly<Record<string, Move>> = {
    ArrowLeft: 'previous',
    ArrowRight: 'next',
    End: 'last',
    Home: 'first',
};

export function moveOfKey(key: string, axis: 'horizontal' | 'vertical'): Move | undefined {
    const moves = axis === 'vertical' ? ARROWS : SIDEWAYS;
    return Object.hasOwn(moves, key) ? moves[key] : undefined;
}

const FOCUSABLE = [
    'a[href]',
    'button:not(:disabled)',
    'input:not(:disabled)',
    'select:not(:disabled)',
    'textarea:not(:disabled)',
    '[tabindex]',
]
    .map((one) => `${one}:not([tabindex="-1"])`)
    .join(', ');

export function focusablesIn(root: Element, selector = FOCUSABLE): readonly HTMLElement[] {
    return [...root.querySelectorAll<HTMLElement>(selector)];
}

export function moveFocus(
    within: readonly HTMLElement[],
    move: Move,
    wrap: boolean,
): HTMLElement | undefined {
    const active = document.activeElement;
    const index = active instanceof HTMLElement ? within.indexOf(active) : -1;
    const next = within[moved(within.length, index, move, wrap)];
    next?.focus();
    return next;
}

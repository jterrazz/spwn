import { useEffect } from 'react';
import type { KeyboardEvent, MouseEvent, PointerEvent, RefObject } from 'react';

import type { Frame } from './camera';
import type { Direction } from './neighbour';
import type { CanvasRig } from './rig';

// value: what the canvas does with a node: open or close it, open it by Enter, hand it to the reader, go up.
export type Actions = {
    activate: (id: string, by: 'click' | 'key') => void;
    enter: (id: string) => void;
    raise: (id: string) => void;
    up: () => void;
};

const ARROWS: Readonly<Record<string, Direction>> = {
    ArrowDown: 'down',
    ArrowLeft: 'left',
    ArrowRight: 'right',
    ArrowUp: 'up',
};

function nodeAt(target: EventTarget): string | undefined {
    return target instanceof Element
        ? target.closest<HTMLElement>('button[data-n]')?.dataset.n
        : undefined;
}

function readAt(target: EventTarget | null): string | undefined {
    return target instanceof Element
        ? target.closest<HTMLElement>('button[data-read]')?.dataset.read
        : undefined;
}

// value: the prototype's click: an open node's Read hands it to the reader, a node's button opens it, a region's
// value: ground opens the region, an open node's ground does nothing, and the desk goes up.
function clicked(event: MouseEvent<HTMLElement>, rig: CanvasRig, actions: Actions): void {
    if (rig.swallows()) {
        event.preventDefault();
        event.stopPropagation();
        return;
    }
    const read = readAt(event.target);
    if (read !== undefined) {
        actions.raise(read);
        return;
    }
    const id = nodeAt(event.target);
    if (id !== undefined) {
        actions.activate(id, event.detail === 0 ? 'key' : 'click');
        return;
    }
    const node =
        event.target instanceof Element ? event.target.closest<HTMLElement>('[data-node]') : null;
    if (node?.dataset.face === 'region' && node.dataset.node !== undefined) {
        actions.activate(node.dataset.node, 'click');
    } else if (node?.dataset.face !== 'open') {
        actions.up();
    }
}

type Keyed = { rig: CanvasRig; frame: Frame; motion: boolean; actions: Actions };

function keyed(event: KeyboardEvent<HTMLElement>, { actions, frame, motion, rig }: Keyed): void {
    if (event.key === 'Escape') {
        event.preventDefault();
        actions.up();
        return;
    }
    const id = nodeAt(event.target);
    if (id === undefined) {
        return;
    }
    const direction = ARROWS[event.key];
    if (event.key === ' ' || event.key === 'Enter' || direction !== undefined) {
        event.preventDefault();
    }
    if (event.key === ' ') {
        actions.raise(id);
    } else if (event.key === 'Enter') {
        actions.enter(id);
    } else if (direction !== undefined) {
        rig.step(event.currentTarget, frame, id, { animate: motion, direction });
    }
}

export type WheelPast = (delta: number, at: number) => void;

// invariant: the wheel pans as a drag does, through the rig, and is listened to as not passive so it can hold the page.
// invariant: A wheel down that leaves the camera at its clamp's bottom is handed on, its delta and its time.
export function useWheel(
    root: RefObject<HTMLElement | null>,
    rig: CanvasRig,
    frame: Frame,
    past: undefined | WheelPast,
): void {
    useEffect(() => {
        const element = root.current;
        if (element === null) {
            return;
        }
        const wheel = (event: WheelEvent) => {
            event.preventDefault();
            const atBottom = rig.pan(frame, { x: -event.deltaX, y: -event.deltaY });
            if (atBottom && event.deltaY > 0) {
                past?.(event.deltaY, event.timeStamp);
            }
        };
        element.addEventListener('wheel', wheel, { passive: false });
        return () => {
            element.removeEventListener('wheel', wheel);
        };
    }, [frame, past, rig, root]);
}

export function handlersOf(keys: Keyed) {
    const { actions, frame, rig } = keys;
    return {
        onClick: (event: MouseEvent<HTMLElement>) => {
            clicked(event, rig, actions);
        },
        onKeyDown: (event: KeyboardEvent<HTMLElement>) => {
            keyed(event, keys);
        },
        onPointerCancel: rig.end,
        onPointerDown: rig.down,
        onPointerMove: (event: PointerEvent<HTMLElement>) => {
            rig.move(event, frame);
        },
        onPointerUp: rig.end,
    };
}

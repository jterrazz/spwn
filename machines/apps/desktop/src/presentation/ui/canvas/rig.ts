import type { PointerEvent, RefObject } from 'react';

import { cameraClamp, cameraShowing, cameraTarget } from './camera';
import type { Camera, Frame } from './camera';
import { carry, liveCamera, place } from './flip';
import type { Placed } from './graph';
import { neighbourOf } from './neighbour';
import type { Direction } from './neighbour';

// value: the prototype's pan: a press moves the world once the pointer has gone 4 px, and the click it ends in opens
// value: nothing.
const THRESHOLD = 4;

export type Toward = { direction: Direction; animate: boolean };

type Drag = { x: number; y: number; from: Camera; pointer: number; moved: boolean };

// invariant: a node is left out when made inert inside the canvas (a ghost, a node leaving); an inert page around the
// invariant: canvas (a raised sheet over it) leaves every node in, so its tab stop stands when the page lowers.
function buttonsIn(root: HTMLElement): HTMLButtonElement[] {
    return [...root.querySelectorAll<HTMLButtonElement>('button[data-n]')].filter((one) => {
        const held = one.closest('[inert]');
        return held === null || !root.contains(held);
    });
}

// value: one tab stop in the canvas, on `want` when it is drawn, else on the first node; the focus follows it when
// value: the canvas held it, so the node a click or a key opened keeps it through the move.
export function rove(
    root: HTMLElement,
    want: null | string,
    drawn: ReadonlyMap<string, Placed>,
): void {
    const live = buttonsIn(root).filter((one) => drawn.has(one.dataset.n ?? ''));
    const chosen = live.find((one) => one.dataset.n === want) ?? live[0];
    for (const one of live) {
        one.tabIndex = one === chosen ? 0 : -1;
    }
    if (
        chosen !== undefined &&
        root.contains(document.activeElement) &&
        document.activeElement !== chosen
    ) {
        chosen.focus({ preventScroll: true });
    }
}

// invariant: the camera lives here, outside React: a pan, a wheel and a carry write the world's custom properties and
// invariant: set no state, so they render no component. `anchor` is where the layout put the camera, the clamp's
// invariant: reference, as the prototype's `cam0`.
export class CanvasRig {
    private readonly shown: RefObject<HTMLElement | null>;

    constructor(shown: RefObject<HTMLElement | null>) {
        this.shown = shown;
    }

    get world(): HTMLElement | null {
        return this.shown.current;
    }

    readonly elements = new Map<string, HTMLElement>();
    private before: null | ReadonlyMap<string, Placed> = null;
    private at: Camera = { x: 0, y: 0 };
    private anchor: Camera = { x: 0, y: 0 };
    private running: Animation | null = null;
    private drag: Drag | null = null;
    private dragged = false;

    readonly hold = (id: string, element: HTMLElement | null): void => {
        if (element === null) {
            this.elements.delete(id);
        } else {
            this.elements.set(id, element);
        }
    };

    // value: the rects of the last layout drawn, null before the first, and `rects` kept as the next one's.
    remembered(rects: ReadonlyMap<string, Placed>): null | ReadonlyMap<string, Placed> {
        const { before } = this;
        this.before = rects;
        return before;
    }

    private stop(world: HTMLElement): Camera {
        if (this.running !== null) {
            this.at = liveCamera(world);
            this.running.cancel();
            this.running = null;
            place(world, this.at);
        }
        return this.at;
    }

    private moveTo(world: HTMLElement, to: Camera, animate: boolean): void {
        const from = this.stop(world);
        this.at = to;
        this.running = animate ? carry(world, from, to) : null;
        if (!animate) {
            place(world, to);
        }
    }

    settle(frame: Frame, path: readonly string[], animate: boolean): void {
        if (this.world === null) {
            return;
        }
        const target = cameraTarget(frame, path);
        this.anchor = target;
        this.moveTo(this.world, target, animate);
    }

    // value: the camera moved by `by`, held in its clamp; whether it now rests at the clamp's bottom is handed back.
    pan(frame: Frame, by: Camera): boolean {
        if (this.world === null) {
            return false;
        }
        const from = this.stop(this.world);
        const to = cameraClamp(frame, this.anchor, { x: from.x + by.x, y: from.y + by.y });
        this.at = { x: to.x, y: to.y };
        place(this.world, this.at);
        return to.atBottom;
    }

    readonly down = (event: PointerEvent<HTMLElement>): void => {
        if (event.button !== 0 || this.world === null) {
            return;
        }
        const from = this.stop(this.world);
        this.drag = {
            from,
            moved: false,
            pointer: event.pointerId,
            x: event.clientX,
            y: event.clientY,
        };
        this.dragged = false;
    };

    move(event: PointerEvent<HTMLElement>, frame: Frame): void {
        const { drag } = this;
        if (drag === null || drag.pointer !== event.pointerId || this.world === null) {
            return;
        }
        const x = event.clientX - drag.x;
        const y = event.clientY - drag.y;
        if (!drag.moved && Math.hypot(x, y) < THRESHOLD) {
            return;
        }
        if (!drag.moved) {
            drag.moved = true;
            event.currentTarget.setPointerCapture(event.pointerId);
            event.currentTarget.dataset.grabbing = '';
        }
        const to = cameraClamp(frame, this.anchor, { x: drag.from.x + x, y: drag.from.y + y });
        this.at = { x: to.x, y: to.y };
        place(this.world, this.at);
    }

    readonly end = (event: PointerEvent<HTMLElement>): void => {
        if (this.drag === null || this.drag.pointer !== event.pointerId) {
            return;
        }
        this.dragged = this.drag.moved;
        this.drag = null;
        delete event.currentTarget.dataset.grabbing;
        setTimeout(() => {
            this.dragged = false;
        }, 0);
    };

    // value: whether the click a pan ended in is to be swallowed.
    swallows(): boolean {
        return this.dragged;
    }

    // value: the arrow keys: the focus moves to the neighbour the prototype's score names, and the camera follows it
    // value: just far enough to show it.
    step(root: HTMLElement, frame: Frame, from: string, toward: Toward): void {
        const { animate, direction } = toward;
        const drawn = buttonsIn(root);
        const reachable = new Map(
            drawn.flatMap((one) => {
                const id = one.dataset.n ?? '';
                const rect = frame.layout.rects.get(id);
                return rect === undefined ? [] : [[id, rect] as const];
            }),
        );
        const to = neighbourOf(reachable, from, direction, frame.geometry);
        const next = drawn.find((one) => one.dataset.n === to);
        if (to === null || next === undefined || this.world === null) {
            return;
        }
        for (const one of drawn) {
            one.tabIndex = one === next ? 0 : -1;
        }
        next.focus({ preventScroll: true });
        const at = this.stop(this.world);
        const showing = cameraShowing(frame, at, to);
        if (showing !== at) {
            this.moveTo(this.world, showing, animate);
        }
    }
}

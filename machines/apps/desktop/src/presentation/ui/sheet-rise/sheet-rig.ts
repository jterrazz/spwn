import type { PointerEvent, RefObject } from 'react';

import { dragVerdict, RISE } from './rise';
import type { Verdict } from './rise';

type Drag = {
    y: number;
    from: number;
    at: number;
    pointer: number;
    moved: boolean;
    raised: boolean;
};

function translate(y: number): string {
    return `translateY(${String(y)}px)`;
}

// value: the --ease-sheet token as WAAPI reads it; a page that loads no tokens reads it empty and runs linear.
function curveOf(element: Element): string {
    const value = getComputedStyle(element).getPropertyValue('--ease-sheet').trim();
    return value === '' ? 'linear' : value;
}

// invariant: the sheet's place lives here, outside React: a drag writes the sheet's own transform and a rise or a
// invariant: fall is one WAAPI animation of it, so neither renders a component. At rest the place is the stylesheet's
// invariant: (lowered by the strip's height, raised at zero), and the inline transform is cleared.
export class SheetRig {
    private readonly sheet: RefObject<HTMLElement | null>;
    private readonly strip: RefObject<HTMLElement | null>;
    private running: Animation | null = null;
    private drag: Drag | null = null;
    private dragged = false;
    private held: null | number = null;

    constructor(sheet: RefObject<HTMLElement | null>, strip: RefObject<HTMLElement | null>) {
        this.sheet = sheet;
        this.strip = strip;
    }

    // value: where the sheet rests: at zero raised, else its height less the strip's, so only the strip shows.
    resting(raised: boolean): number {
        const sheet = this.sheet.current;
        if (raised || sheet === null) {
            return 0;
        }
        return sheet.offsetHeight - (this.strip.current?.offsetHeight ?? RISE.peek.line);
    }

    // value: where the sheet shows now: mid-animation as drawn, mid-drag or just released where the hand left it.
    private live(sheet: HTMLElement, raised: boolean): number {
        if (this.held !== null) {
            return this.held;
        }
        if (this.running !== null) {
            return new DOMMatrixReadOnly(getComputedStyle(sheet).transform).f;
        }
        return this.resting(raised);
    }

    private stop(sheet: HTMLElement, raised: boolean): number {
        const at = this.live(sheet, raised);
        this.running?.cancel();
        this.running = null;
        this.held = null;
        return at;
    }

    // value: the sheet carried from where it shows to its rest for `raised`, on the prototype's curve; with no motion,
    // value: or less than a pixel to go, it jumps.
    settle(raised: boolean, from: boolean, motion: boolean): void {
        const sheet = this.sheet.current;
        if (sheet === null) {
            return;
        }
        const at = this.stop(sheet, from);
        const to = this.resting(raised);
        sheet.style.removeProperty('transform');
        if (!motion || Math.abs(at - to) <= 1) {
            delete sheet.dataset.moving;
            return;
        }
        sheet.dataset.moving = '';
        const running = sheet.animate(
            [{ transform: translate(at) }, { transform: translate(to) }],
            {
                duration: RISE.duration,
                easing: curveOf(sheet),
            },
        );
        this.running = running;
        const ended = () => {
            if (this.running === running) {
                this.running = null;
                delete sheet.dataset.moving;
            }
        };
        running.addEventListener('finish', ended);
        running.addEventListener('cancel', ended);
    }

    readonly down = (event: PointerEvent<HTMLElement>, raised: boolean): void => {
        const sheet = this.sheet.current;
        if (event.button !== 0 || sheet === null) {
            return;
        }
        this.drag = {
            at: event.timeStamp,
            from: 0,
            moved: false,
            pointer: event.pointerId,
            raised,
            y: event.clientY,
        };
        this.dragged = false;
    };

    move(event: PointerEvent<HTMLElement>): void {
        const { drag } = this;
        const sheet = this.sheet.current;
        if (drag === null || drag.pointer !== event.pointerId || sheet === null) {
            return;
        }
        const by = event.clientY - drag.y;
        if (!drag.moved && Math.abs(by) < RISE.drag.start) {
            return;
        }
        if (!drag.moved) {
            drag.moved = true;
            drag.from = this.stop(sheet, drag.raised);
            event.currentTarget.setPointerCapture(event.pointerId);
            sheet.dataset.moving = '';
        }
        const y = Math.max(0, Math.min(this.resting(false), drag.from + by));
        this.held = y;
        sheet.style.transform = translate(y);
    }

    // value: a drag's end: where it lands by the prototype's distances and speeds, null for a press that never moved.
    end(event: PointerEvent<HTMLElement>): null | Verdict {
        const { drag } = this;
        if (drag === null || drag.pointer !== event.pointerId) {
            return null;
        }
        this.drag = null;
        this.dragged = drag.moved;
        setTimeout(() => {
            this.dragged = false;
        }, 0);
        if (!drag.moved || this.held === null) {
            return null;
        }
        return dragVerdict(drag.raised, this.held - drag.from, event.timeStamp - drag.at);
    }

    // value: a pointer taken away mid-drag: the sheet goes back to where it rested.
    cancel(raised: boolean, motion: boolean): void {
        if (this.drag === null) {
            return;
        }
        this.drag = null;
        this.settle(raised, raised, motion);
    }

    // value: whether the click a drag ended in is to be swallowed.
    swallows(): boolean {
        return this.dragged;
    }
}

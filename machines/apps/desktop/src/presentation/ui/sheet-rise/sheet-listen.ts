import { useEffect } from 'react';
import type { RefObject } from 'react';

import { aDialogIsOpen } from '../dialog';
import { PullGate } from './pull-gate';
import { RISE, WheelGate } from './rise';

// value: what the listeners read at the time of an event: whether the sheet is raised, and what to ask of its holder.
export type Told = { current: { raised: boolean; onRaise: () => void; onLower: () => void } };

export type Parts = {
    root: RefObject<HTMLDivElement | null>;
    behind: RefObject<HTMLDivElement | null>;
    bar: RefObject<HTMLDivElement | null>;
    sheet: RefObject<HTMLElement | null>;
    grab: RefObject<HTMLButtonElement | null>;
    strip: RefObject<HTMLDivElement | null>;
    scroll: RefObject<HTMLDivElement | null>;
};

function held(event: WheelEvent): void {
    event.preventDefault();
}

// invariant: the wheels over the strip, the grabber and the bar are not passive, so a wheel over them never scrolls
// invariant: what holds the space; the page's wheel and its touch are passive, so its own scroll never waits on them.
function listenWheels(
    strip: HTMLElement,
    grab: HTMLElement,
    bar: HTMLElement,
    told: Told,
): () => void {
    const rising = new WheelGate(RISE.wheel.strip);
    const raising = (event: WheelEvent) => {
        event.preventDefault();
        if (
            !told.current.raised &&
            event.deltaY > 0 &&
            rising.push(event.deltaY, event.timeStamp)
        ) {
            told.current.onRaise();
        }
    };
    strip.addEventListener('wheel', raising, { passive: false });
    grab.addEventListener('wheel', raising, { passive: false });
    bar.addEventListener('wheel', held, { passive: false });
    return () => {
        strip.removeEventListener('wheel', raising);
        grab.removeEventListener('wheel', raising);
        bar.removeEventListener('wheel', held);
    };
}

// value: the page pulled back to the canvas at its very top: a wheel past the prototype's sum, a touch past 80 px.
function listenPull(scroll: HTMLElement, told: Told): () => void {
    const pulling = new PullGate();
    let touched: null | number = null;
    const pulled = (event: WheelEvent) => {
        if (
            told.current.raised &&
            pulling.push(event.deltaY, event.timeStamp, scroll.scrollTop > 0)
        ) {
            told.current.onLower();
        }
    };
    const touchStart = (event: TouchEvent) => {
        touched = scroll.scrollTop <= 0 ? (event.touches[0]?.clientY ?? null) : null;
    };
    const touchMove = (event: TouchEvent) => {
        const at = event.touches[0]?.clientY;
        if (
            touched !== null &&
            at !== undefined &&
            told.current.raised &&
            scroll.scrollTop <= 0 &&
            at - touched > RISE.pull
        ) {
            touched = null;
            told.current.onLower();
        }
    };
    scroll.addEventListener('wheel', pulled, { passive: true });
    scroll.addEventListener('touchstart', touchStart, { passive: true });
    scroll.addEventListener('touchmove', touchMove, { passive: true });
    return () => {
        scroll.removeEventListener('wheel', pulled);
        scroll.removeEventListener('touchstart', touchStart);
        scroll.removeEventListener('touchmove', touchMove);
    };
}

// invariant: Escape lowers a raised page before anything under it hears the key: pressed in the space or on nothing,
// invariant: never while a dialog is open over it, and taken, so the canvas's own Escape goes up only once lowered.
function listenEscape(root: HTMLElement, told: Told): () => void {
    const escape = (event: KeyboardEvent) => {
        const inside = event.target instanceof Node && root.contains(event.target);
        if (event.key !== 'Escape' || event.defaultPrevented || !told.current.raised) {
            return;
        }
        if ((inside || event.target === document.body) && !aDialogIsOpen()) {
            event.preventDefault();
            told.current.onLower();
        }
    };
    document.addEventListener('keydown', escape);
    return () => {
        document.removeEventListener('keydown', escape);
    };
}

// value: the sheet's native listeners, set once for its life: the wheels, the pull and Escape.
export function useListeners(parts: Parts, told: Told): void {
    useEffect(() => {
        const [root, bar, grab, scroll, strip] = [
            parts.root.current,
            parts.bar.current,
            parts.grab.current,
            parts.scroll.current,
            parts.strip.current,
        ];
        if (root === null || bar === null || grab === null || scroll === null || strip === null) {
            return;
        }
        const ends = [
            listenWheels(strip, grab, bar, told),
            listenPull(scroll, told),
            listenEscape(root, told),
        ];
        return () => {
            for (const end of ends) {
                end();
            }
        };
    }, [parts, told]);
}

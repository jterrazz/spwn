import { useLayoutEffect, useMemo, useRef } from 'react';
import type { MouseEvent, PointerEvent, ReactNode } from 'react';

import type { WheelPast } from '../canvas/gestures';
import { useMotion } from '../canvas/motion';
import { RISE, WheelGate } from './rise';
import { useListeners } from './sheet-listen';
import type { Parts, Told } from './sheet-listen';
import { SheetRig } from './sheet-rig';

import styles from './sheet.module.css';

// value: the one word the sheet says itself: the accessible name of its grabber.
export type SheetWords = { grab: string };

export type SheetProps = {
    raised: boolean;
    label: string;
    words: SheetWords;
    behind: ReactNode;
    bar: ReactNode;
    strip: ReactNode;
    children: ReactNode;
    onRaise: () => void;
    onLower: () => void;
};

// value: what the canvas hands on of a wheel past its bottom (`onWheelPast`), gated as the prototype's canvas raises
// value: the sheet; one function for the holder's life, so the canvas it is handed to never renders for it.
export function useWheelPast(onPast: () => void): WheelPast {
    const told = useRef(onPast);
    useLayoutEffect(() => {
        told.current = onPast;
    });
    return useMemo(() => {
        const gate = new WheelGate(RISE.wheel.canvas);
        return (delta: number, at: number) => {
            if (gate.push(delta, at)) {
                told.current();
            }
        };
    }, []);
}

// value: the focus a lowered sheet hands back: the canvas's one tab stop, once the canvas has drawn the path it
// value: lowers to; only when the focus was in the sheet, its bar or nowhere, never taken from elsewhere.
function handBack(root: HTMLElement, behind: HTMLElement): void {
    const now = document.activeElement;
    if (now !== null && now !== document.body && !root.contains(now)) {
        return;
    }
    behind.querySelector<HTMLElement>('[tabindex="0"]')?.focus({ preventScroll: true });
}

// value: the sheet carried to its rest when `raised` changes, then the focus moved: into the page 40 ms after a
// value: rise (the prototype's), back to the canvas once lowered.
function useSettle(rig: SheetRig, parts: Parts, raised: boolean, motion: boolean): void {
    const was = useRef(raised);
    useLayoutEffect(() => {
        const from = was.current;
        if (from === raised) {
            return;
        }
        was.current = raised;
        rig.settle(raised, from, motion);
        const timer = setTimeout(
            () => {
                const [root, behind] = [parts.root.current, parts.behind.current];
                if (raised) {
                    parts.scroll.current?.focus({ preventScroll: true });
                } else if (root !== null && behind !== null) {
                    handBack(root, behind);
                }
            },
            raised && motion ? RISE.focusAfter : 0,
        );
        return () => {
            clearTimeout(timer);
        };
    }, [motion, parts, raised, rig]);
}

// value: the pointer's part: a drag of the grabber or the strip lands by the rig's verdict, a click on the grabber
// value: toggles, and the click a drag ends in is swallowed.
function useDrag(rig: SheetRig, { onLower, onRaise, raised }: Told['current'], motion: boolean) {
    const dragging = {
        onPointerCancel: () => {
            rig.cancel(raised, motion);
        },
        onPointerDown: (event: PointerEvent<HTMLElement>) => {
            rig.down(event, raised);
        },
        onPointerMove: (event: PointerEvent<HTMLElement>) => {
            rig.move(event);
        },
        onPointerUp: (event: PointerEvent<HTMLElement>) => {
            const verdict = rig.end(event);
            if (verdict === 'raise' && !raised) {
                onRaise();
            } else if (verdict === 'lower' && raised) {
                onLower();
            } else if (verdict !== null) {
                rig.settle(raised, raised, motion);
            }
        },
    };
    const swallowed = (event: MouseEvent<HTMLElement>) => {
        if (rig.swallows()) {
            event.preventDefault();
            event.stopPropagation();
        }
    };
    const toggle = () => {
        if (!rig.swallows()) {
            (raised ? onLower : onRaise)();
        }
    };
    return { dragging, swallowed, toggle };
}

function useTold({ onLower, onRaise, raised }: SheetProps): Told {
    const told = useRef({ onLower, onRaise, raised });
    useLayoutEffect(() => {
        told.current = { onLower, onRaise, raised };
    });
    return told;
}

function hidden(when: boolean) {
    return { 'aria-hidden': when ? true : undefined, inert: when };
}

// value: the page as a sheet over what is `behind` it (the canvas): lowered, only its strip peeks at the foot of the
// value: space; raised, it covers the space under the collapsed `bar`, the canvas faded, lifted and inert behind it.
// value: It is controlled: a click on the strip, a wheel down over it, a drag up and the grabber ask `onRaise`;
// value: Escape, a pull at the page's top, a drag down and the grabber ask `onLower`.
export function Sheet(props: SheetProps) {
    const { raised } = props;
    const motion = useMotion();
    const root = useRef<HTMLDivElement | null>(null);
    const behind = useRef<HTMLDivElement | null>(null);
    const bar = useRef<HTMLDivElement | null>(null);
    const sheet = useRef<HTMLElement | null>(null);
    const grab = useRef<HTMLButtonElement | null>(null);
    const strip = useRef<HTMLDivElement | null>(null);
    const scroll = useRef<HTMLDivElement | null>(null);
    const parts = useMemo(() => ({ bar, behind, grab, root, scroll, sheet, strip }), []);
    const rig = useMemo(() => new SheetRig(sheet, strip), []);
    useListeners(parts, useTold(props));
    useSettle(rig, parts, raised, motion);
    const { dragging, swallowed, toggle } = useDrag(rig, props, motion);

    return (
        <div className={styles.space} data-raised={raised ? '' : undefined} ref={root}>
            <div className={styles.behind} ref={behind} {...hidden(raised)}>
                {props.behind}
            </div>
            <div className={styles.bar} ref={bar} {...hidden(!raised)}>
                {props.bar}
            </div>
            <section aria-label={props.label} className={styles.sheet} ref={sheet}>
                <button
                    aria-label={props.words.grab}
                    className={styles.grab}
                    onClick={toggle}
                    ref={grab}
                    tabIndex={-1}
                    type="button"
                    {...dragging}
                />
                <div
                    className={styles.strip}
                    onClickCapture={swallowed}
                    ref={strip}
                    {...hidden(raised)}
                    {...dragging}
                >
                    {props.strip}
                </div>
                <div
                    className={styles.scroll}
                    data-scroll=""
                    inert={!raised}
                    ref={scroll}
                    tabIndex={raised ? -1 : undefined}
                >
                    {props.children}
                </div>
            </section>
        </div>
    );
}

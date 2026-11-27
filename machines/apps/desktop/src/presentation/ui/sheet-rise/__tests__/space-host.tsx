import { waitUntil } from '@jterrazz/test';
import { Profiler } from 'react';

import { OsSpaceScene } from '../os-space';

import styles from './space-host.module.css';

export const PAGE = ['S:product', 'P:product/boundaries'];

export const DEEP = ['S:product', 'P:product/boundaries', 'C:product/boundaries:2'];

export const OPEN = 'Open Boundaries';

export const BAR = 'Where this page sits';

const nothing = (): void => undefined;

// value: the OS space the specs drive, in the prototype's space at 1100 wide, its renders told to `onRender`.
export function Space({
    onRender = nothing,
    raised = false,
    start = PAGE,
}: {
    onRender?: () => void;
    raised?: boolean;
    start?: readonly string[];
}) {
    return (
        <div className={styles.desk}>
            <Profiler id="space" onRender={onRender}>
                <OsSpaceScene start={start} startRaised={raised} />
            </Profiler>
        </div>
    );
}

export function scroller(): HTMLElement {
    const found = document.querySelector<HTMLElement>('[data-scroll]');
    if (found === null) {
        throw new Error('no page drawn');
    }
    return found;
}

export function isRaised(): boolean {
    return scroller().closest('section')?.parentElement?.dataset.raised !== undefined;
}

export function expanded(id: string): boolean {
    return (
        document.querySelector(`[data-node="${id}"] button`)?.getAttribute('aria-expanded') ===
        'true'
    );
}

export function focusedNode(): string | undefined {
    return document.activeElement?.closest<HTMLElement>('[data-node]')?.dataset.node;
}

export function strip(): Element {
    const found = document.querySelector('[data-peek]');
    if (found === null) {
        throw new Error('no strip drawn');
    }
    return found;
}

export function wheel(target: Element | null, deltaY: number): WheelEvent {
    const event = new WheelEvent('wheel', { bubbles: true, cancelable: true, deltaY });
    target?.dispatchEvent(event);
    return event;
}

export function pointer(type: string, target: Element, y: number): void {
    target.dispatchEvent(
        new PointerEvent(type, {
            bubbles: true,
            button: 0,
            cancelable: true,
            clientX: 300,
            clientY: y,
            isPrimary: true,
            pointerId: 1,
            pointerType: 'mouse',
        }),
    );
}

// value: a gesture's own time passing: the gates read each event's time stamp, so a pause in a wheel or a slow drag
// value: is the clock having moved on, waited on as that.
export async function elapsed(ms: number): Promise<void> {
    const from = performance.now();
    await waitUntil(() => performance.now() - from >= ms, {
        why: `${String(ms)} ms of the gesture passed`,
    });
}

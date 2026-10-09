'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

const PANEL_WIDTH = 380;
const PANEL_GAP = 24;
const DRAG_THRESHOLD = 3;

interface Drag {
    startX: number;
    moved: boolean;
}

function centerOf(element: HTMLDivElement | null | undefined): number {
    return element ? element.offsetLeft + element.offsetWidth / 2 : 0;
}

function nearestTo(centers: number[], target: number): number {
    let nearest = 0;
    for (const [index, center] of centers.entries()) {
        if (Math.abs(center - target) < Math.abs((centers[nearest] ?? 0) - target)) {
            nearest = index;
        }
    }
    return nearest;
}

function useDragListeners(dragging: boolean, onMove: (x: number) => void, onEnd: () => void): void {
    useEffect(() => {
        if (!dragging) {
            return;
        }
        const move = (event: MouseEvent) => {
            onMove(event.clientX);
        };
        const touch = (event: TouchEvent) => {
            const first = event.touches.item(0);
            if (first !== null) {
                onMove(first.clientX);
            }
        };
        globalThis.addEventListener('mousemove', move);
        globalThis.addEventListener('mouseup', onEnd);
        globalThis.addEventListener('touchmove', touch, { passive: true });
        globalThis.addEventListener('touchend', onEnd);
        return () => {
            globalThis.removeEventListener('mousemove', move);
            globalThis.removeEventListener('mouseup', onEnd);
            globalThis.removeEventListener('touchmove', touch);
            globalThis.removeEventListener('touchend', onEnd);
        };
    }, [dragging, onMove, onEnd]);
}

function useMeasures(count: number) {
    const trackRef = useRef<HTMLDivElement>(null);
    const panelRef = useRef<HTMLDivElement>(null);
    const planets = useRef<(HTMLDivElement | null)[]>([]);

    const visibleWidth = useCallback((withPanel: boolean) => {
        const parent = trackRef.current?.parentElement;
        if (!parent) {
            return 800;
        }
        return withPanel
            ? parent.clientWidth - (panelRef.current?.offsetWidth ?? PANEL_WIDTH) - PANEL_GAP
            : parent.clientWidth;
    }, []);

    const centers = useCallback(
        () => Array.from({ length: count }, (_, index) => centerOf(planets.current[index])),
        [count],
    );

    const centringOffset = useCallback(
        (focus: null | number, withPanel: boolean) => {
            const target = visibleWidth(withPanel) / 2;
            const all = centers();
            if (focus !== null) {
                return target - (all[focus] ?? 0);
            }
            return all.length === 0 ? 0 : target - ((all[0] ?? 0) + (all.at(-1) ?? 0)) / 2;
        },
        [visibleWidth, centers],
    );

    return { centers, centringOffset, panelRef, planets, trackRef, visibleWidth };
}

/** The draggable, self-centring row of planets: where it sits, how a drag moves it, and which planet a drag lands on. */
export function useCarousel(count: number, selected: null | number) {
    const measures = useMeasures(count);
    const { centers, centringOffset, panelRef, planets, trackRef, visibleWidth } = measures;
    const [offset, setOffset] = useState(0);
    const [delta, setDelta] = useState(0);
    const [dragging, setDragging] = useState(false);
    const drag = useRef<Drag>({ moved: false, startX: 0 });
    const lastFocus = useRef<null | number>(null);

    useEffect(() => {
        lastFocus.current = selected ?? lastFocus.current;
        const recentre = () => {
            setOffset(centringOffset(lastFocus.current, selected !== null));
            setDelta(0);
        };
        // hazard: two frames — the first lets React mount the selection panel, the second measures it; one frame measures a panel that is not there yet
        const frame = requestAnimationFrame(() => {
            requestAnimationFrame(recentre);
        });
        globalThis.addEventListener('resize', recentre);
        return () => {
            cancelAnimationFrame(frame);
            globalThis.removeEventListener('resize', recentre);
        };
    }, [selected, centringOffset]);

    const startDrag = useCallback((x: number) => {
        drag.current = { moved: false, startX: x };
        setDragging(true);
    }, []);

    const moveDrag = useCallback((x: number) => {
        const dx = x - drag.current.startX;
        drag.current.moved ||= Math.abs(dx) > DRAG_THRESHOLD;
        setDelta(dx);
    }, []);

    const endDrag = useCallback(() => {
        setDragging(false);
        setDelta(0);
        if (drag.current.moved) {
            const target = visibleWidth(selected !== null) / 2;
            const landed = centers().map((center) => center + offset + delta);
            setOffset(target - (centers()[nearestTo(landed, target)] ?? 0));
        }
    }, [centers, delta, offset, selected, visibleWidth]);

    useDragListeners(dragging, moveDrag, endDrag);

    // invariant: a click that ends a drag is the drag's, not a selection
    const wasDrag = useCallback(() => drag.current.moved, []);

    return { dragging, offset: offset + delta, panelRef, planets, startDrag, trackRef, wasDrag };
}

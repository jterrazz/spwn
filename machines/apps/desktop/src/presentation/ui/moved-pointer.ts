import { useRef } from 'react';
import type { MouseEvent } from 'react';

type Pointer = { x: number; y: number };

// hazard: Chromium sends a mousemove to whatever comes to lie under a resting pointer when the list redraws, so
// hazard: only a pointer that moved since its last event, and not the first event heard, chooses a row.
export function useMovedPointer(
    onChoose: (index: number) => void,
): (event: MouseEvent, index: number) => void {
    const last = useRef<Pointer | undefined>(undefined);
    return (event, index) => {
        const before = last.current;
        last.current = { x: event.clientX, y: event.clientY };
        if (before !== undefined && (before.x !== event.clientX || before.y !== event.clientY)) {
            onChoose(index);
        }
    };
}

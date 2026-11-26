import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';

import { foldOf } from './fold';
import type { Segment } from './segments';

export function keyOf(segment: Segment): string {
    return `${segment.key}:${segment.label}`;
}

// invariant: the last level shrinks into its ellipsis when the room is short, so its width is the one it would take
// invariant: whole: what overflows it, plus what its name hides.
function naturalWidth(item: HTMLElement): number {
    const name = item.querySelector<HTMLElement>('[data-fits]');
    const hidden = name === null ? 0 : name.scrollWidth - name.clientWidth;
    return Math.max(item.offsetWidth, item.scrollWidth) + hidden;
}

// invariant: each segment's width is kept from the last time it was drawn, so a folded one still counts; the room
// invariant: is the path's own, measured again whenever the bar changes width.
export function useFold(segments: readonly Segment[]) {
    const nav = useRef<HTMLElement>(null);
    const widths = useRef(new Map<string, number>());
    const [folded, setFolded] = useState<readonly string[]>([]);
    const drawn = useRef<readonly string[]>([]);
    const keys = segments.map(keyOf).join('/');
    const refold = useCallback(() => {
        const element = nav.current;
        if (!element) {
            return;
        }
        for (const item of element.querySelectorAll<HTMLElement>('li[data-segment]')) {
            widths.current.set(item.dataset.segment ?? '', naturalWidth(item));
        }
        const named = keys.split('/');
        const next = foldOf(
            named.map((key) => widths.current.get(key) ?? 0),
            element.clientWidth,
        ).map((at) => named[at] ?? '');
        if (next.join('/') !== drawn.current.join('/')) {
            drawn.current = next;
            setFolded(next);
        }
    }, [keys]);
    useLayoutEffect(refold);
    useEffect(() => {
        const sized = new ResizeObserver(refold);
        if (nav.current) {
            sized.observe(nav.current);
        }
        return () => {
            sized.disconnect();
        };
    }, [refold]);
    return { folded, nav };
}

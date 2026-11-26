import type { Geometry } from './geometry';
import type { Rect } from './graph';

export type Direction = 'down' | 'left' | 'right' | 'up';

const AXES: Readonly<Record<Direction, readonly [number, number]>> = {
    down: [0, 1],
    left: [-1, 0],
    right: [1, 0],
    up: [0, -1],
};

function centreOf(rect: Rect, geometry: Geometry): readonly [number, number] {
    const { clipHeight, clipWidth } = geometry.neighbour;
    return [rect.x + Math.min(rect.w, clipWidth) / 2, rect.y + Math.min(rect.h, clipHeight) / 2];
}

// value: how far `to` lies from `from` when walking `direction`: the distance along it plus the distance across it,
// value: weighted, measured between centres clipped to a pill's head; null when `to` is not ahead.
export function scoreToward(
    from: Rect,
    to: Rect,
    direction: Direction,
    geometry: Geometry,
): null | number {
    const [dx, dy] = AXES[direction];
    const [mx, my] = centreOf(from, geometry);
    const [ox, oy] = centreOf(to, geometry);
    const along = (ox - mx) * dx + (oy - my) * dy;
    if (along <= geometry.neighbour.ahead) {
        return null;
    }
    return along + Math.abs((ox - mx) * dy - (oy - my) * dx) * geometry.neighbour.across;
}

// invariant: two nodes at the same score are told apart by the order `rects` holds them, the first kept: the
// invariant: prototype kept the first in its DOM, whose order is the history of its renders.
export function neighbourOf(
    rects: ReadonlyMap<string, Rect>,
    from: string,
    direction: Direction,
    geometry: Geometry,
): null | string {
    const me = rects.get(from);
    if (me === undefined) {
        return null;
    }
    let best: null | string = null;
    let bestScore = Number.POSITIVE_INFINITY;
    for (const [id, rect] of rects) {
        const score = id === from ? null : scoreToward(me, rect, direction, geometry);
        if (score !== null && score < bestScore) {
            bestScore = score;
            best = id;
        }
    }
    return best;
}

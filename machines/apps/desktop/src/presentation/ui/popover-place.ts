export type Box = { left: number; top: number; width: number; height: number };

export type Align = 'end' | 'start';

export type Side = 'above' | 'below';

export type Placed = { x: number; y: number; height: number; side: Side };

// value: the gap between an opener and its popover, and the margin a popover keeps from the window's edges, in px.
const GAP = 6;
const MARGIN = 8;

function clamped(value: number, low: number, high: number): number {
    return Math.max(low, Math.min(value, high));
}

// invariant: a popover opens below its opener unless it would be cut there and more room lies above; it never
// invariant: leaves its bounds: shifted sideways to fit, its height capped at the room of the side it opened on.
export function placed(
    anchor: Box,
    size: { width: number; height: number },
    bounds: Box,
    align: Align,
): Placed {
    const below = bounds.top + bounds.height - (anchor.top + anchor.height) - GAP - MARGIN;
    const above = anchor.top - bounds.top - GAP - MARGIN;
    const side: Side = size.height <= below || below >= above ? 'below' : 'above';
    const height = Math.max(0, Math.min(size.height, side === 'below' ? below : above));
    const wanted = align === 'start' ? anchor.left : anchor.left + anchor.width - size.width;
    const x = clamped(
        wanted,
        bounds.left + MARGIN,
        Math.max(bounds.left + MARGIN, bounds.left + bounds.width - MARGIN - size.width),
    );
    const y = side === 'below' ? anchor.top + anchor.height + GAP : anchor.top - GAP - height;
    return { height, side, x: x - bounds.left, y: y - bounds.top };
}

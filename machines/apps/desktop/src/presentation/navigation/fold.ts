// value: the width of the `…` a fold draws in place of the segments it hides, its separator included, in px.
export const FOLD_WIDTH = 44;

// value: the bar's parts that never fold, in px as `path-bar.module.css` draws them on a Mac: the lights' inset,
// value: the sign, the column's pin, Back and Forward, the gaps between them, then at its right the Inbox, Find
// value: and the Settings gear, with the bar's end padding.
export const BAR_FIXED = {
    arrows: 2 * 28,
    end: 14,
    find: 120,
    gaps: 9 * 6,
    gear: 28,
    inbox: 44,
    lights: 84,
    sign: 32,
    toggle: 28,
} as const;

export function fixedWidth(): number {
    return Object.values(BAR_FIXED).reduce((sum, width) => sum + width, 0);
}

// invariant: the first and the last segment never fold: the space says where I am, the last what I see. The ones
// invariant: between fold from the first onward, so the path keeps its nearest levels while it fits.
export function foldOf(widths: readonly number[], room: number): readonly number[] {
    const total = widths.reduce((sum, width) => sum + width, 0);
    if (total <= room || widths.length < 3) {
        return [];
    }
    const folded: number[] = [];
    let shown = total + FOLD_WIDTH;
    for (let at = 1; at < widths.length - 1 && shown > room; at += 1) {
        folded.push(at);
        shown -= widths[at] ?? 0;
    }
    return folded;
}

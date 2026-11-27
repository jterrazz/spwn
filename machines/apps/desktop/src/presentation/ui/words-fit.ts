export const CUT = '…';

// value: the longest run of whole words, then `…`, that `room` px holds; the words whole when they fit, or when
// value: nothing is laid out yet (`room` 0). A first word too long alone keeps its `…` and the line's own ellipsis.
export function wordsThatFit(
    words: string,
    room: number,
    widthOf: (text: string) => number,
): string {
    if (room <= 0 || widthOf(words) <= room) {
        return words;
    }
    const each = words.split(' ');
    for (let count = each.length - 1; count > 0; count -= 1) {
        const shown = `${each.slice(0, count).join(' ')}${CUT}`;
        if (widthOf(shown) <= room) {
            return shown;
        }
    }
    return `${each[0] ?? ''}${CUT}`;
}

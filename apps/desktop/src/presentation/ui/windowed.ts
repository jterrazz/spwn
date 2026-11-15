export type Span = { start: number; end: number };

export type Viewport = {
    top: number;
    height: number;
    count: number;
    row: number;
    overscan: number;
};

export function spanOf({ top, height, count, row, overscan }: Viewport): Span {
    if (count === 0 || row <= 0) {
        return { end: 0, start: 0 };
    }
    const first = Math.floor(Math.max(0, top) / row);
    const last = Math.ceil((Math.max(0, top) + Math.max(0, height)) / row);
    return {
        end: Math.min(count, last + overscan),
        start: Math.min(Math.max(0, first - overscan), Math.max(0, count - 1)),
    };
}

export function heightOf(count: number, row: number, most: number): number {
    return Math.min(count * row, most);
}

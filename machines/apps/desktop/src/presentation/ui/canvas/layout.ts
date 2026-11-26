import { nodeOf } from './graph';
import type { CanvasGraph, Placed, Rect } from './graph';
import { place } from './place';
import type { Options } from './place';

export type Viewport = { width: number; height: number };

export type Guide = { band: number; x: number; y: number; w: number };

export type Layout = {
    rects: ReadonlyMap<string, Placed>;
    guides: readonly Guide[];
    box: Rect;
};

function shifted(part: ReadonlyMap<string, Placed>, dx: number, dy: number): [string, Placed][] {
    return [...part].map(([id, rect]) => [id, { ...rect, x: rect.x + dx, y: rect.y + dy }]);
}

function rowsOf(graph: CanvasGraph): Map<number, string[]> {
    const rows = new Map<number, string[]>();
    for (const root of graph.roots) {
        const { band } = nodeOf(graph, root);
        rows.set(band, [...(rows.get(band) ?? []), root]);
    }
    return rows;
}

export function layout(
    graph: CanvasGraph,
    sel: readonly string[],
    viewport: Viewport,
    options: Options,
): Layout {
    const { layout: shape } = options.geometry;
    const width = Math.min(shape.maxWidth, viewport.width - shape.sideMargin);
    const rects = new Map<string, Placed>();
    const guides: Guide[] = [];
    const rows = rowsOf(graph);
    let y = 0;
    for (const band of [...rows.keys()].toSorted((a, b) => a - b)) {
        let x = shape.gutter;
        let tallest = 0;
        for (const root of rows.get(band) ?? []) {
            const part = new Map<string, Placed>();
            const size = place({ graph, options, out: part, sel }, root, { width, x: 0, y: 0 });
            for (const [id, rect] of shifted(part, x, y)) {
                rects.set(id, rect);
            }
            x += size.w + shape.seatGap;
            tallest = Math.max(tallest, size.h);
        }
        guides.push({ band, w: 0, x: 0, y: y - shape.guideLift });
        y +=
            Math.max(tallest, shape.bandFloor) +
            (sel.length > 0 ? shape.bandGapOpen : shape.bandGap);
    }
    const all = [...rects.values()];
    if (all.length === 0) {
        return { box: { h: 0, w: 0, x: 0, y: 0 }, guides, rects };
    }
    const x0 = Math.min(0, ...all.map((rect) => rect.x));
    const y0 = Math.min(...all.map((rect) => rect.y));
    const x1 = Math.max(...all.map((rect) => rect.x + rect.w));
    const y1 = Math.max(...all.map((rect) => rect.y + rect.h));
    return {
        box: { h: y1 - y0, w: x1 - x0, x: x0, y: y0 },
        guides: guides.map((guide) => ({ ...guide, w: x1 - guide.x })),
        rects,
    };
}

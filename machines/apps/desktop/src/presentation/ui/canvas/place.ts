import { faceOf } from './faces';
import type { Geometry, Size } from './geometry';
import { nodeOf } from './graph';
import type { CanvasGraph, CanvasNode, Face, Placed } from './graph';

export type Options = {
    geometry: Geometry;
    measureText: (text: string) => number;
};

export type Extent = { w: number; h: number };

export type Scene = {
    graph: CanvasGraph;
    sel: readonly string[];
    options: Options;
    out: Map<string, Placed>;
};

export type At = { x: number; y: number; width: number };

type Kid = { face: Face; node: CanvasNode };

type Holder = Kid & { kids: Kid[] };

function cardOf(geometry: Geometry, depth: number): Size {
    return geometry.cards[depth] ?? geometry.otherCard;
}

export function measure(node: CanvasNode, face: Face, options: Options): Extent {
    const { dot, pill } = options.geometry;
    if (face === 'pill') {
        const count = node.marks.length;
        const marks =
            count > 0 ? pill.marksLead + Math.min(count, pill.marksCap) * pill.markPitch : 0;
        const label = Math.min(options.measureText(node.label), pill.labelCap);
        return { h: pill.height, w: Math.ceil(pill.lead + label + marks + pill.tail) };
    }
    if (face === 'dot') {
        if (node.depth === dot.chipDepth) {
            const count = Math.min(node.kids.length, dot.chipCap);
            return { h: dot.chipHeight, w: dot.chipLead + count * dot.chipPitch + dot.chipTail };
        }
        return { h: dot.size, w: dot.size };
    }
    const [w, h] = cardOf(options.geometry, node.depth);
    return { h, w };
}

function kidsOf(scene: Scene, node: CanvasNode): Kid[] {
    return node.kids
        .map((id) => nodeOf(scene.graph, id))
        .flatMap((kid) => {
            const face = faceOf(kid, scene.sel);
            return face === null ? [] : [{ face, node: kid }];
        });
}

function stackCards(scene: Scene, holder: Holder, at: At): Extent {
    const { face, kids, node } = holder;
    const { geometry } = scene.options;
    const { gap, pad } = geometry;
    const first = kids[0];
    const [cardWidth, cardHeight] =
        first === undefined ? geometry.emptyCard : cardOf(geometry, first.node.depth);
    const head = geometry.headByDepth[node.depth] ?? geometry.head;
    const fitting = Math.floor((at.width - 2 * pad + gap) / (cardWidth + gap));
    const cols = Math.max(
        1,
        Math.min(kids.length, face === 'region' ? geometry.regionColumns : Math.max(1, fitting)),
    );
    const rows = Math.ceil(kids.length / cols);
    for (const [index, kid] of kids.entries()) {
        scene.out.set(kid.node.id, {
            face: 'card',
            h: cardHeight,
            w: cardWidth,
            x: at.x + pad + (index % cols) * (cardWidth + gap),
            y: at.y + head + Math.floor(index / cols) * (cardHeight + gap),
        });
    }
    const w = pad * 2 + cols * cardWidth + (cols - 1) * gap;
    const h = head + rows * cardHeight + Math.max(0, rows - 1) * gap + pad;
    const floor =
        face === 'open'
            ? Math.min(at.width, geometry.minOpen[node.depth] ?? geometry.otherMinOpen)
            : 0;
    return { h, w: Math.max(w, floor) };
}

function flowKids(scene: Scene, holder: Holder, at: At, opened: Kid): Extent {
    const { kids, node } = holder;
    const { geometry } = scene.options;
    const { pad, row } = geometry;
    const head = geometry.headByDepth[node.depth] ?? geometry.head;
    let cx = at.x + pad;
    let cy = at.y + head;
    let any = false;
    for (const kid of kids.filter((each) => each !== opened)) {
        const size = measure(kid.node, kid.face, scene.options);
        if (cx + size.w > at.x + at.width - pad && cx > at.x + pad) {
            cx = at.x + pad;
            cy += row.pitch;
        }
        scene.out.set(kid.node.id, {
            face: kid.face,
            x: cx,
            y: cy + (row.height - size.h) / 2,
            ...size,
        });
        cx += size.w + (kid.face === 'dot' ? row.dotGap : row.pillGap);
        any = true;
    }
    if (any) {
        cy += row.height + row.after;
    }
    const inner = place(scene, opened.node.id, { width: at.width - 2 * pad, x: at.x + pad, y: cy });
    return { h: cy + inner.h + pad - at.y, w: at.width };
}

export function place(scene: Scene, id: string, at: At): Extent {
    const node = nodeOf(scene.graph, id);
    const face = faceOf(node, scene.sel);
    if (face === null) {
        throw new Error(`the node ${id} wears no face under this path`);
    }
    if (face !== 'open' && face !== 'region') {
        const size = measure(node, face, scene.options);
        scene.out.set(id, { face, x: at.x, y: at.y, ...size });
        return size;
    }
    const kids = kidsOf(scene, node);
    const opened = kids.find((kid) => kid.face === 'open');
    const size =
        opened === undefined
            ? stackCards(scene, { face, kids, node }, at)
            : flowKids(scene, { face, kids, node }, at, opened);
    scene.out.set(id, { face, x: at.x, y: at.y, ...size });
    return size;
}

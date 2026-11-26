import type { Geometry } from './geometry';
import type { Rect } from './graph';
import type { Layout, Viewport } from './layout';

export type Camera = { x: number; y: number };

export type Clamped = Camera & { atBottom: boolean; atTop: boolean };

export type Frame = { layout: Layout; viewport: Viewport; geometry: Geometry };

function rectOf(layout: Layout, id: string | undefined): Rect | undefined {
    return id === undefined ? undefined : layout.rects.get(id);
}

function focusOf(frame: Frame, sel: readonly string[]): Rect {
    const { geometry, layout, viewport } = frame;
    if (sel.length === 0) {
        return layout.box;
    }
    const deep = rectOf(layout, sel.at(-1));
    if (deep === undefined) {
        throw new Error('the open path names a node the layout does not place');
    }
    const parent = rectOf(layout, sel.at(-2));
    const fits =
        parent !== undefined &&
        parent.h <= viewport.height - geometry.camera.fitHeight &&
        parent.w <= viewport.width - geometry.camera.fitWidth;
    return fits ? parent : deep;
}

export function cameraTarget(frame: Frame, sel: readonly string[]): Camera {
    const { camera } = frame.geometry;
    const { height, width } = frame.viewport;
    const rect = focusOf(frame, sel);
    const top = sel.length > 0 ? camera.topOpen : camera.topRoot;
    const x = Math.round(width / 2 - (rect.x + rect.w / 2));
    const y =
        rect.h <= height - camera.fitHeight
            ? Math.round((height - rect.h) / 2 - rect.y)
            : Math.round(top - rect.y);
    const { box } = frame.layout;
    return sel.length === 0 && box.w > width - camera.fitWidth
        ? { x: Math.round(camera.fitWidth - box.x), y }
        : { x, y };
}

type Span = { start: number; size: number; room: number };

function pulledIn(at: number, { room, size, start }: Span, margin: number): number {
    const shown = Math.min(size, room - 2 * margin);
    if (start + at < margin) {
        return margin - start;
    }
    if (start + shown + at > room - margin) {
        return room - margin - start - shown;
    }
    return at;
}

// value: the camera moved the least that shows `id` whole, a margin from each edge; a node larger than the canvas
// value: is shown from its top-left corner. The camera is handed back as it was when nothing moves.
export function cameraShowing(frame: Frame, at: Camera, id: string): Camera {
    const rect = frame.layout.rects.get(id);
    if (rect === undefined) {
        return at;
    }
    const margin = frame.geometry.camera.visible;
    const { height, width } = frame.viewport;
    const x = pulledIn(at.x, { room: width, size: rect.w, start: rect.x }, margin);
    const y = pulledIn(at.y, { room: height, size: rect.h, start: rect.y }, margin);
    return x === at.x && y === at.y ? at : { x, y };
}

export function cameraClamp(frame: Frame, from: Camera, to: Camera): Clamped {
    const { box } = frame.layout;
    const margin = frame.geometry.camera.clamp;
    const minX = Math.min(frame.viewport.width - margin - (box.x + box.w), from.x);
    const maxX = Math.max(margin - box.x, from.x);
    const minY = Math.min(frame.viewport.height - margin - (box.y + box.h), from.y);
    const maxY = Math.max(margin - box.y, from.y);
    return {
        atBottom: to.y <= minY + 0.5,
        atTop: to.y >= maxY - 0.5,
        x: Math.max(minX, Math.min(maxX, to.x)),
        y: Math.max(minY, Math.min(maxY, to.y)),
    };
}

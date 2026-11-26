// value: one projection for every drawing, 30° isometric: x runs down-right, y down-left, z up.
const COS = 0.8660254;
const SIN = 0.5;

// value: every drawing is framed in the same box, centred on what it reaches.
const FRAME_WIDTH = 280;
const FRAME_HEIGHT = 190;

export type Point = readonly [number, number];
export type Spot = readonly [number, number, number];

export type Paint =
    | 'accent'
    | 'bad'
    | 'bright'
    | 'dim'
    | 'far'
    | 'ink'
    | 'left'
    | 'litLeft'
    | 'litRight'
    | 'litTop'
    | 'near'
    | 'none'
    | 'off'
    | 'onBad'
    | 'panel'
    | 'right'
    | 'shadow'
    | 'top'
    | 'well';

export type Ink = 'accent' | 'dim' | 'ink' | 'none' | 'onAccent' | 'onBad';

export type Look = {
    paint?: Paint;
    ink?: Ink;
    dash?: 'dot' | 'long' | 'short' | undefined;
    fade?: 'little' | 'more' | 'most';
    glyph?: boolean;
};

export type Geometry =
    | { tag: 'circle'; cx: number; cy: number; r: number }
    | { tag: 'ellipse'; cx: number; cy: number; rx: number; ry: number }
    | { tag: 'path'; d: string }
    | { tag: 'polygon'; points: string }
    | { tag: 'polyline'; points: string };

export type Shape = { id: number; look: Look; geometry: Geometry };

export type Drawing = {
    viewBox: string;
    reach: { width: number; height: number };
    shapes: readonly Shape[];
};

export type Box = {
    lit?: boolean;
    ghost?: boolean;
    faces?: Paint;
    ink?: Ink;
    dash?: Look['dash'];
};

export function projected([x, y, z]: Spot): Point {
    return [(x - y) * COS, (x + y) * SIN - z];
}

export const fixed = (value: number): number => Math.round(value * 10) / 10;

export const listed = (points: readonly Point[]): string =>
    points.map(([x, y]) => `${String(fixed(x))},${String(fixed(y))}`).join(' ');

export const radians = (degrees: number): number => (degrees * Math.PI) / 180;

function facesOf(box: Box): readonly [Paint, Paint, Paint] {
    if (box.ghost === true) {
        return ['none', 'none', 'none'];
    }
    if (box.lit === true) {
        return ['litTop', 'litLeft', 'litRight'];
    }
    return box.faces === undefined ? ['top', 'left', 'right'] : [box.faces, box.faces, box.faces];
}

export class Iso {
    private readonly drawn: Shape[] = [];
    private left = Infinity;
    private top = Infinity;
    private right = -Infinity;
    private bottom = -Infinity;

    reached([x, y]: Point): void {
        this.left = Math.min(this.left, x);
        this.top = Math.min(this.top, y);
        this.right = Math.max(this.right, x);
        this.bottom = Math.max(this.bottom, y);
    }

    path(d: string, look: Look): void {
        this.add({ d, tag: 'path' }, look);
    }

    dot([x, y]: Point, r: number, paint: Paint): void {
        this.add({ cx: fixed(x), cy: fixed(y), r, tag: 'circle' }, { ink: 'none', paint });
    }

    poly(spots: readonly Spot[], look: Look = {}): void {
        this.add({ points: this.projectedAll(spots), tag: 'polygon' }, look);
    }

    line(spots: readonly Spot[], look: Look = {}): void {
        this.add({ points: this.projectedAll(spots), tag: 'polyline' }, look);
    }

    box([x, y, z]: Spot, [w, d, h]: Spot, box: Box = {}): void {
        const top = z + h;
        const [faceTop, faceLeft, faceRight] = facesOf(box);
        const look: Look = {
            dash: box.dash,
            ink: box.ink ?? (box.lit === true ? 'accent' : 'ink'),
        };
        const faces: readonly (readonly [readonly Spot[], Paint])[] = [
            [
                [
                    [x + w, y, z],
                    [x + w, y + d, z],
                    [x + w, y + d, top],
                    [x + w, y, top],
                ],
                faceRight,
            ],
            [
                [
                    [x, y + d, z],
                    [x + w, y + d, z],
                    [x + w, y + d, top],
                    [x, y + d, top],
                ],
                faceLeft,
            ],
            [
                [
                    [x, y, top],
                    [x + w, y, top],
                    [x + w, y + d, top],
                    [x, y + d, top],
                ],
                faceTop,
            ],
        ];
        for (const [spots, paint] of faces) {
            this.poly(spots, { ...look, paint });
        }
    }

    shadow(x: number, y: number, [w, d]: Point, spread = 8): void {
        this.poly(
            [
                [x + 2, y + spread, 0],
                [x + w + 2, y + spread, 0],
                [x + w + 2, y + d + spread, 0],
                [x + 2, y + d + spread, 0],
            ],
            { ink: 'none', paint: 'shadow' },
        );
    }

    ellipse([cx, cy, z]: Spot, r: number, look: Look = {}): void {
        const [x, y] = projected([cx, cy, z]);
        const rx = r * 1.2247;
        const ry = r * Math.SQRT1_2;
        this.reached([x - rx, y - ry]);
        this.reached([x + rx, y + ry]);
        this.add(
            { cx: fixed(x), cy: fixed(y), rx: fixed(rx), ry: fixed(ry), tag: 'ellipse' },
            look,
        );
    }

    cylinder([cx, cy, z]: Spot, [r, h]: Point, look: Look & { side: Paint }): void {
        const [x, y] = projected([cx, cy, z]);
        const rx = fixed(r * 1.2247);
        const ry = fixed(r * Math.SQRT1_2);
        this.reached([x - rx, y - h - ry]);
        this.reached([x + rx, y + ry]);
        const left = String(fixed(x - rx));
        const right = String(fixed(x + rx));
        const base = String(fixed(y));
        const lid = String(fixed(y - h));
        const arc = `A${String(rx)},${String(ry)} 0 0 0`;
        this.path(`M${left},${base} ${arc} ${right},${base} L${right},${lid} L${left},${lid} Z`, {
            ...look,
            paint: look.side,
        });
        this.ellipse([cx, cy, z + h], r, look);
    }

    arc([cx, cy, z]: Spot, r: number, [from, to]: Point, look: Look): void {
        const spots: Spot[] = [];
        for (let step = 0; step <= 28; step += 1) {
            const turn = radians(from + ((to - from) * step) / 28);
            spots.push([cx + r * Math.cos(turn), cy + r * Math.sin(turn), z]);
        }
        this.line(spots, look);
    }

    upright(points: readonly Point[], y: number, look: Look): void {
        this.poly(
            points.map(([x, z]) => [x, y, z]),
            look,
        );
    }

    drawing(): Drawing {
        const x = (this.left + this.right) / 2 - FRAME_WIDTH / 2;
        const y = (this.top + this.bottom) / 2 - FRAME_HEIGHT / 2;
        return {
            reach: { height: this.bottom - this.top, width: this.right - this.left },
            shapes: this.drawn,
            viewBox: `${String(fixed(x))} ${String(fixed(y))} ${String(FRAME_WIDTH)} ${String(FRAME_HEIGHT)}`,
        };
    }

    private add(geometry: Geometry, look: Look): void {
        this.drawn.push({ geometry, id: this.drawn.length, look });
    }

    private projectedAll(spots: readonly Spot[]): string {
        const points = spots.map((spot) => projected(spot));
        for (const point of points) {
            this.reached(point);
        }
        return listed(points);
    }
}

import { listed, projected, radians } from './iso';
import type { Iso, Look, Point, Spot } from './iso';

// value: a shape standing in the plane of a pane's face: its centre (x, z), its radius, and a capsule's length.
type Upright = { at: Point; r: number; length?: number };

const segment = (points: readonly Point[]): string => `M${listed(points).replaceAll(' ', ' L')}`;

export function capsule(
    iso: Iso,
    { at: [x, z], r, length = 0 }: Upright,
    y: number,
    look: Look,
): void {
    const points: Point[] = [];
    for (let step = 0; step <= 12; step += 1) {
        const turn = -Math.PI / 2 + (Math.PI * step) / 12;
        points.push([x + length + r * Math.cos(turn), z + r * Math.sin(turn)]);
    }
    for (let step = 0; step <= 12; step += 1) {
        const turn = Math.PI / 2 + (Math.PI * step) / 12;
        points.push([x + r * Math.cos(turn), z + r * Math.sin(turn)]);
    }
    iso.upright(points, y, look);
}

export function disc(iso: Iso, { at: [x, z], r }: Upright, y: number, look: Look): void {
    const points: Point[] = [];
    for (let step = 0; step < 20; step += 1) {
        const turn = (2 * Math.PI * step) / 20;
        points.push([x + r * Math.cos(turn), z + r * Math.sin(turn)]);
    }
    iso.upright(points, y, look);
}

export function diamond(iso: Iso, spot: Spot, size = 4): void {
    const [cx, cy] = projected(spot);
    iso.reached([cx, cy]);
    iso.path(
        `${segment([
            [cx, cy - size * 0.6],
            [cx + size, cy],
            [cx, cy + size * 0.6],
            [cx - size, cy],
        ])} Z`,
        {},
    );
}

export function dash(iso: Iso, spot: Spot, angle: number, length = 8): void {
    const start = projected(spot);
    const turn = radians(angle);
    const end: Point = [start[0] + length * Math.cos(turn), start[1] + length * Math.sin(turn)];
    iso.reached(start);
    iso.reached(end);
    iso.path(segment([start, end]), {});
}

export function rays(iso: Iso, spot: Spot, [near, far]: Point, angles: readonly number[]): void {
    const [cx, cy] = projected(spot);
    for (const angle of angles) {
        const turn = radians(angle);
        const end: Point = [cx + far * Math.cos(turn), cy + far * Math.sin(turn)];
        iso.reached(end);
        iso.path(segment([[cx + near * Math.cos(turn), cy + near * Math.sin(turn)], end]), {});
    }
}

export function badge(iso: Iso, spot: Spot, r: number, kind: 'bad' | 'check'): void {
    const [cx, cy] = projected(spot);
    iso.reached([cx - r, cy - r]);
    iso.reached([cx + r, cy + r]);
    iso.dot([cx, cy], r, kind === 'bad' ? 'bad' : 'accent');
    if (kind === 'check') {
        const tick: Point[] = [
            [cx - r * 0.42, cy + r * 0.02],
            [cx - r * 0.1, cy + r * 0.34],
            [cx + r * 0.45, cy - r * 0.3],
        ];
        iso.path(segment(tick), { glyph: true, ink: 'onAccent' });
        return;
    }
    const bar: Point[] = [
        [cx, cy - r * 0.45],
        [cx, cy + r * 0.1],
    ];
    iso.path(segment(bar), { glyph: true, ink: 'onBad' });
    iso.dot([cx, cy + r * 0.42], 1.4, 'onBad');
}

// value: the sign, three dots lying on a top face, the front one in the accent.
export function sign(iso: Iso, [x, y, z]: Spot, k = 1): void {
    iso.ellipse([x + 4 * k, y - 15 * k, z], 4.5 * k, { ink: 'none', paint: 'far' });
    iso.ellipse([x + k, y - 6 * k, z], 6.5 * k, { ink: 'none', paint: 'near' });
    iso.ellipse([x - 2 * k, y + 5 * k, z], 9 * k, { ink: 'none', paint: 'accent' });
}

// value: the few spark marks, always at the sides, never on top of the object.
export function sparks(iso: Iso, left: number, right: number): void {
    diamond(iso, [-34, 66, left], 4.5);
    dash(iso, [62, -26, right], -30, 9);
    dash(iso, [66, -20, right - 9], -8, 6);
}

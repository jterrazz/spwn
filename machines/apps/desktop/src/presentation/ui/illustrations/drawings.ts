import { Iso, projected } from './iso';
import type { Drawing, Ink, Look, Point, Spot } from './iso';
import { badge, capsule, dash, diamond, disc, rays, sign, sparks } from './iso-marks';

const GUIDE: Look = { dash: 'long', ink: 'dim' };
const LANDING: Look = { dash: 'short', fade: 'little', ink: 'accent' };
const LAYER = 11;
const GAP = 8;
const SLAB: Spot = [80, 80, LAYER];

function engine(arrow: boolean): Drawing {
    const iso = new Iso();
    iso.shadow(-62, -46, [124, 92], 10);
    iso.box([-62, -46, 0], [124, 92, 10]);
    iso.poly(
        [
            [-23, -12, 10],
            [27, -12, 10],
            [27, 28, 10],
            [-23, 28, 10],
        ],
        { ink: 'none', paint: 'shadow' },
    );
    for (const [x, y] of [
        [25, -20],
        [25, 20],
        [-25, 20],
    ] as const) {
        iso.line(
            [
                [x, y, 12],
                [x, y, 28],
            ],
            GUIDE,
        );
    }
    iso.box([-25, -20, 32], [50, 40, 22]);
    if (arrow) {
        iso.line(
            [
                [0, 0, 96],
                [0, 0, 66],
            ],
            { ink: 'accent' },
        );
        iso.line(
            [
                [-8, 0, 75],
                [0, 0, 66],
                [8, 0, 75],
            ],
            { ink: 'accent' },
        );
    } else {
        sign(iso, [0, 0, 54]);
    }
    sparks(iso, 84, 70);
    return iso.drawing();
}

function landing(iso: Iso, base: number): void {
    for (const [x, y] of [
        [40, -40],
        [40, 40],
        [-40, 40],
    ] as const) {
        iso.line(
            [
                [x, y, base],
                [x, y, base + GAP],
            ],
            LANDING,
        );
    }
    iso.box([-40, -40, base + GAP], SLAB, { lit: true });
}

// value: the build is its own progress: the engine's four steps are four layers, the done ones stacked, the
// value: current one landing in the accent, the rest dashed above it; a slipped layer is a build that failed.
function build(layer: 1 | 2 | 3 | 4, slipped: boolean): Drawing {
    const iso = new Iso();
    iso.shadow(-40, -40, [80, 80], 10);
    for (let below = 0; below < layer - 1; below += 1) {
        iso.box([-40, -40, below * LAYER], SLAB);
    }
    const base = (layer - 1) * LAYER;
    if (slipped) {
        iso.box([-16, -40, base], SLAB, { lit: true });
        dash(iso, [72, -44, base + 30], -40, 9);
        dash(iso, [76, -36, base + 22], -12, 7);
    } else {
        landing(iso, base);
    }
    const above = slipped ? base : base + GAP + LAYER;
    for (let next = layer; next < 4; next += 1) {
        iso.box([-40, -40, above + GAP + (next - layer) * (LAYER + GAP)], SLAB, {
            dash: 'long',
            faces: 'panel',
            ink: 'dim',
        });
    }
    if (!slipped) {
        sparks(iso, above + 44, above + 26);
    }
    return iso.drawing();
}

function machine(iso: Iso, failed: boolean): void {
    iso.shadow(-52, -40, [104, 80], 10);
    iso.box([-52, -40, 0], [104, 80, 26]);
    for (const z of [8, 13, 18]) {
        iso.line(
            [
                [-40, 40, z],
                [-14, 40, z],
            ],
            { ink: 'dim' },
        );
    }
    iso.dot(projected([34, 40, 13]), 2.6, failed ? 'dim' : 'accent');
    iso.poly(
        [
            [-38, -28, 26],
            [30, -28, 26],
            [30, 24, 26],
            [-38, 24, 26],
        ],
        { ink: 'dim', paint: 'well' },
    );
    const lines: readonly (readonly [number, number, Ink])[] = [
        [-18, 20, failed ? 'dim' : 'ink'],
        [-6, 12, 'dim'],
        [6, 16, 'dim'],
    ];
    for (const [y, length, ink] of lines) {
        iso.line(
            [
                [-28, y, 26],
                [-28 + length, y, 26],
            ],
            { ink },
        );
    }
}

function boot(applying: boolean): Drawing {
    const iso = new Iso();
    machine(iso, false);
    iso.line(
        [
            [-28, 16, 26],
            [-22, 16, 26],
        ],
        { ink: 'accent' },
    );
    if (applying) {
        iso.box([-10, -18, 40], [14, 14, 14], { lit: true });
        iso.line(
            [
                [4, -18, 26],
                [4, -18, 38],
            ],
            LANDING,
        );
        iso.box([16, -26, 26], [12, 12, 9]);
        iso.box([-30, 4, 60], [12, 12, 12], { dash: 'long', ghost: true, ink: 'dim' });
    } else {
        rays(iso, [-4, -2, 60], [10, 18], [-150, -120, -90, -60, -30]);
        sparks(iso, 80, 56);
    }
    return iso.drawing();
}

function bootFailed(): Drawing {
    const iso = new Iso();
    machine(iso, true);
    badge(iso, [-4, -2, 92], 12, 'bad');
    return iso.drawing();
}

function touch(): Drawing {
    const iso = new Iso();
    iso.shadow(-56, -56, [112, 112], 10);
    iso.box([-56, -56, 0], [112, 112, 8]);
    iso.ellipse([0, 0, 8], 46, { dash: 'dot', fade: 'little', ink: 'accent' });
    iso.box([-30, -30, 8], [60, 60, 10]);
    const ridges: readonly (readonly [number, Point])[] = [
        [4, [0, 330]],
        [9, [30, 320]],
        [14, [-60, 220]],
        [19, [40, 300]],
        [24, [-40, 140]],
    ];
    for (const [r, turn] of ridges) {
        iso.arc([0, 0, 18], r, turn, { ink: 'accent' });
    }
    sparks(iso, 72, 50);
    return iso.drawing();
}

function settings(): Drawing {
    const iso = new Iso();
    iso.shadow(-64, -30, [128, 60], 10);
    iso.box([-64, -30, 0], [128, 60, 6]);
    iso.poly(
        [
            [-46, 6, 6],
            [50, 6, 6],
            [50, 14, 6],
            [-46, 14, 6],
        ],
        { ink: 'none', paint: 'shadow' },
    );
    iso.box([-48, -4, 6], [96, 8, 78]);
    const face = 4;
    iso.upright(
        [
            [-38, 72],
            [-6, 72],
            [-6, 68],
            [-38, 68],
        ],
        face,
        { ink: 'none', paint: 'ink' },
    );
    for (const [z, on] of [
        [54, true],
        [38, false],
        [22, false],
    ] as const) {
        iso.line(
            [
                [-38, face, z],
                [-2, face, z],
            ],
            { ink: 'dim' },
        );
        const track: Look = on ? { ink: 'accent', paint: 'accent' } : { ink: 'dim', paint: 'off' };
        capsule(iso, { at: [18, z], length: 14, r: 5.5 }, face, track);
        disc(iso, { at: [on ? 32 : 18, z], r: 4 }, face, {
            ink: 'none',
            paint: on ? 'bright' : 'ink',
        });
    }
    dash(iso, [62, -10, 76], -30, 9);
    dash(iso, [66, -4, 68], -8, 7);
    diamond(iso, [-34, 52, 84], 4.5);
    return iso.drawing();
}

export { boot, bootFailed, build, engine, settings, touch };

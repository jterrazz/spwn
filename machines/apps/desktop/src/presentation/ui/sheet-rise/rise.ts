// value: the prototype's sheet (wiki-proto/spwn-wiki-proto-r1.html: `stripH`, `closedY`, `T_SHEET`, `sheetUp`, the
// value: wheels on `#cv`, `#strip` and `#ss`, the touch on `#ss`), in px, ms and px per ms. The sheet rises over 520
// value: ms on the --ease-sheet curve; the strip peeks 112 px with its line, 78 with its title alone; the collapsed
// value: bar and the raised sheet's top sit 52 px down the space.
export const RISE = {
    bar: 52,
    duration: 520,
    drag: { lower: 90, lowerSpeed: 0.6, raise: -60, raiseSpeed: -0.5, start: 5 },
    focusAfter: 40,
    peek: { line: 112, title: 78 },
    pull: 80,
    wheel: {
        canvas: { gap: 220, past: 90 },
        lower: { gap: 240, past: 110 },
        strip: { gap: 220, past: 30 },
    },
} as const;

export type Gate = { past: number; gap: number };

// value: a wheel's deltas summed while they come less than `gap` ms apart; the gate opens once the sum passes
// value: `past`, and starts again from zero.
export class WheelGate {
    private sum = 0;
    private last = Number.NEGATIVE_INFINITY;
    private readonly gate: Gate;

    constructor(gate: Gate) {
        this.gate = gate;
    }

    push(delta: number, at: number): boolean {
        if (at - this.last > this.gate.gap) {
            this.sum = 0;
        }
        this.last = at;
        this.sum += delta;
        if (this.sum > this.gate.past) {
            this.sum = 0;
            return true;
        }
        return false;
    }
}

export type Verdict = 'lower' | 'raise' | 'stay';

// value: where a drag of the sheet lands: `moved` px down (negative up) over `took` ms, from raised or lowered.
export function dragVerdict(raised: boolean, moved: number, took: number): Verdict {
    const speed = moved / Math.max(1, took);
    const { drag } = RISE;
    if (raised) {
        return moved > drag.lower || speed > drag.lowerSpeed ? 'lower' : 'stay';
    }
    return moved < drag.raise || speed < drag.raiseSpeed ? 'raise' : 'stay';
}

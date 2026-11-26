import { RISE } from './rise';
import type { Gate } from './rise';

// value: the raised page's wheel back to the canvas: upward deltas summed at the page's very top, the gesture's
// value: own; a gesture that scrolled the page first never lowers it, until it pauses `gap` ms.
export class PullGate {
    private sum = 0;
    private away = false;
    private last = Number.NEGATIVE_INFINITY;
    private readonly gate: Gate = RISE.wheel.lower;

    push(delta: number, at: number, scrolled: boolean): boolean {
        if (at - this.last > this.gate.gap) {
            this.sum = 0;
            this.away = false;
        }
        this.last = at;
        if (scrolled) {
            this.away = true;
            this.sum = 0;
            return false;
        }
        if (this.away || delta >= 0) {
            this.sum = 0;
            return false;
        }
        this.sum -= delta;
        if (this.sum > this.gate.past) {
            this.sum = 0;
            this.away = true;
            return true;
        }
        return false;
    }
}

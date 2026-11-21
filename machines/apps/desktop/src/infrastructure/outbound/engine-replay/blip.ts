import type { Link } from '../../../application/ports/engine.port';
import type { Transport } from '../engine-link/transport';
import { helloOf } from './hello';
import { stagedReplay } from './replay';
import type { Pace, Recording } from './replay';

export type Blip = { after: number; lasting: number };

export const BLIP: Blip = { after: 1000, lasting: 3000 };

// value: `?link=blip`, a replay whose link the bridge declares absent once, `after` ms after it is composed, and
// value: hears open again `lasting` ms later, on the same engine: what a window sees of an engine that went away
// value: for three hellos in a row and came back.
export function blippedReplay(recording: Recording, pace?: Pace, blip: Blip = BLIP): Transport {
    const stage = stagedReplay(recording, undefined, pace);
    const socket = `replay of ${recording.world}`;
    const opened: Link = { hello: helloOf(recording), kind: 'open', refused: undefined, socket };
    const absent: Link = { kind: 'absent', reason: 'connection refused', retryIn: 1000, socket };
    setTimeout(() => {
        stage.show(absent);
        setTimeout(() => {
            stage.show(opened);
        }, blip.lasting);
    }, blip.after);
    return stage.transport;
}

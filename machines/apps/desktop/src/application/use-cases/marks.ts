// value: a read's own timeline, in the web inspector's User Timing: per request, from the ask to its first out and from
// value: the ask to its end, each kept under one name so a window that runs for days holds one measure per request.
export type Mark = { start: number; first: string; done: string; out: boolean };

export function markOf(key: string): Mark {
    return {
        done: `read done ${key}`,
        first: `read first ${key}`,
        out: false,
        start: performance.now(),
    };
}

function measured(name: string, start: number): void {
    performance.clearMeasures(name);
    performance.measure(name, { end: performance.now(), start });
}

export function heardOut(mark: Mark | undefined): void {
    if (mark && !mark.out) {
        mark.out = true;
        measured(mark.first, mark.start);
    }
}

export function heardDone(mark: Mark | undefined): void {
    heardOut(mark);
    if (mark) {
        measured(mark.done, mark.start);
    }
}

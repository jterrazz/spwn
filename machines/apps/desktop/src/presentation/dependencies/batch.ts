type Subscribe = (listener: () => void) => () => void;

type Schedule = (flush: () => void) => void;

const NEXT_TASK: Schedule = (flush) => {
    setTimeout(flush, 0);
};

// invariant: every listener the engine's answers tell before the next task is told once, all of them in that task, so
// invariant: frames that arrive in a row (the reads of a page opening, a gesture's rereads) are one React commit; a
// invariant: listener let go of before its turn is not told.
export class Batch {
    private readonly due = new Set<() => void>();
    private readonly batched = new WeakMap<Subscribe, Subscribe>();
    private readonly schedule: Schedule;
    private planned = false;

    constructor(schedule: Schedule = NEXT_TASK) {
        this.schedule = schedule;
    }

    // value: the same function for the same `subscribe` every time, so React keeps its subscription.
    readonly of = (subscribe: Subscribe): Subscribe => {
        let batched = this.batched.get(subscribe);
        if (!batched) {
            batched = (listener) => {
                const leave = subscribe(() => {
                    this.tell(listener);
                });
                return () => {
                    this.due.delete(listener);
                    leave();
                };
            };
            this.batched.set(subscribe, batched);
        }
        return batched;
    };

    private tell(listener: () => void): void {
        this.due.add(listener);
        if (this.planned) {
            return;
        }
        this.planned = true;
        this.schedule(() => {
            this.flush();
        });
    }

    private flush(): void {
        this.planned = false;
        const due = [...this.due];
        this.due.clear();
        for (const listener of due) {
            listener();
        }
    }
}

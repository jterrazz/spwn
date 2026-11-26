type Fiber = {
    tag: number;
    type: unknown;
    flags: number;
    child: Fiber | null;
    sibling: Fiber | null;
    alternate: Fiber | null;
    return: Fiber | null;
    stateNode: unknown;
};

export type Tally = { commits: number; renders: number; by: Readonly<Record<string, number>> };

export type Budget = Omit<Tally, 'by'>;

// value: nothing when `tally` holds within `budget`, else the tally whole, its most rendered components first, so a
// value: failure names what rendered; a tally never taken is over any budget.
export function overBudget(tally: Tally | undefined, budget: Budget): Partial<Tally> {
    if (tally === undefined) {
        return { commits: Number.POSITIVE_INFINITY };
    }
    return tally.commits <= budget.commits && tally.renders <= budget.renders ? {} : tally;
}

// value: React's work tags of a component that runs a render: a function, a class, a forwardRef and a simple memo;
// value: a memo with a compare (14) is left out, its inner function counted by its own fiber.
const RENDERS = new Set([0, 1, 11, 15]);

// value: React's `PerformedWork` flag, set on a fiber whose render ran in the commit.
const PERFORMED_WORK = 1;

const HOST_ROOT = 3;

function isFiber(value: unknown): value is Fiber {
    return typeof value === 'object' && value !== null && 'tag' in value && 'flags' in value;
}

function named(value: unknown): string | undefined {
    if (typeof value === 'function') {
        return value.name === '' ? undefined : value.name;
    }
    if (typeof value === 'object' && value !== null && 'render' in value) {
        return named(value.render);
    }
    return undefined;
}

function fiberOf(element: Element): Fiber | undefined {
    const key = Object.keys(element).find((one) => one.startsWith('__reactFiber$'));
    const fiber: unknown = key === undefined ? undefined : Reflect.get(element, key);
    return isFiber(fiber) ? fiber : undefined;
}

function rootOf(): Fiber | undefined {
    const element = [...document.body.querySelectorAll('*')].find(
        (one) => fiberOf(one) !== undefined,
    );
    let fiber = element === undefined ? undefined : fiberOf(element);
    while (fiber && fiber.tag !== HOST_ROOT) {
        fiber = fiber.return ?? undefined;
    }
    const root: unknown = fiber?.stateNode;
    const current: unknown =
        typeof root === 'object' && root !== null ? Reflect.get(root, 'current') : undefined;
    return isFiber(current) ? current : undefined;
}

// invariant: a commit's renders are read as React's DevTools read them: a fiber whose subtree was reused as it was
// invariant: shares its child with the tree before, so only the children of a fiber cloned in this commit are walked,
// invariant: and of those a component counts when its PerformedWork flag is set, or when it has no tree before (a mount).
class Walk {
    renders = 0;
    readonly by = new Map<string, number>();

    counted(fiber: Fiber): void {
        if (!RENDERS.has(fiber.tag)) {
            return;
        }
        this.renders += 1;
        const name = named(fiber.type) ?? 'anonymous';
        this.by.set(name, (this.by.get(name) ?? 0) + 1);
    }

    mounted(fiber: Fiber): void {
        this.counted(fiber);
        for (let { child } = fiber; child; child = child.sibling) {
            this.mounted(child);
        }
    }

    visited(next: Fiber, before: Fiber): void {
        if (next.flags % (PERFORMED_WORK * 2) === PERFORMED_WORK) {
            this.counted(next);
        }
        if (next.child === before.child) {
            return;
        }
        for (let { child } = next; child; child = child.sibling) {
            if (child.alternate) {
                this.visited(child, child.alternate);
            } else {
                this.mounted(child);
            }
        }
    }
}

// value: a counter of React's commits and component renders for a Profiler's `onRender`, deterministic: it reads the
// value: fibers each commit leaves, never a clock. `taken` hands what was counted since the last take, and starts again.
export function tallied() {
    let walk = new Walk();
    let commits = 0;
    return {
        onRender: () => {
            commits += 1;
            const current = rootOf();
            if (current === undefined) {
                return;
            }
            if (current.alternate) {
                walk.visited(current, current.alternate);
            } else {
                walk.mounted(current);
            }
        },
        taken: (): Tally => {
            const tally = {
                by: Object.fromEntries([...walk.by].toSorted(([, a], [, b]) => b - a)),
                commits,
                renders: walk.renders,
            };
            walk = new Walk();
            commits = 0;
            return tally;
        },
    };
}

/**
 * Narrowing for a value the spec has already proved present.
 *
 * Index access and optional fields type as `T | undefined` under the strict
 * rulebook, while a spec that has just asserted `toHaveLength(1)` knows the
 * entry is there. `required` carries that knowledge into the type and, when
 * the assumption breaks, fails on a sentence that names the missing thing
 * instead of on `Cannot read properties of undefined`.
 */

class MissingValueError extends Error {
    constructor(what: string) {
        super(`expected ${what} to be present`);
        this.name = 'MissingValueError';
    }
}

export function required<T>(value: T | undefined, what: string): T {
    if (value === undefined) {
        throw new MissingValueError(what);
    }

    return value;
}

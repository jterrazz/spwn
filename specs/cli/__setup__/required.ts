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

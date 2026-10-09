/** A field of a parsed payload, read without trusting its shape. */
export function fieldOf(value: unknown, key: string): unknown {
    const field: unknown =
        typeof value === 'object' && value !== null ? Reflect.get(value, key) : undefined;
    return field;
}

/** A string field, or '' when the field is absent or not a string. */
export function textOf(value: unknown, key: string): string {
    const field = fieldOf(value, key);
    return typeof field === 'string' ? field : '';
}

/** A list field, or [] when the field is absent or not a list. */
export function listOf(value: unknown, key: string): unknown[] {
    const field = fieldOf(value, key);
    return Array.isArray(field) ? field : [];
}

/** A number field, or undefined. */
export function numberOf(value: unknown, key: string): number | undefined {
    const field = fieldOf(value, key);
    return typeof field === 'number' ? field : undefined;
}

/** An object field as a record of its own fields, or {} when it is absent or not an object. */
export function recordOf(value: unknown, key: string): Record<string, unknown> {
    const field = fieldOf(value, key);
    if (typeof field !== 'object' || field === null || Array.isArray(field)) {
        return {};
    }
    return Object.fromEntries(Object.keys(field).map((name) => [name, fieldOf(field, name)]));
}

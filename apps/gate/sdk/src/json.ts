/** A value the SDK sends or receives as JSON, read without trusting its shape. */
export type Json = unknown;

/** The field of a JSON object, or undefined when the value holds none. */
export function fieldOf(value: Json, key: string): Json {
    if (typeof value !== 'object' || value === null) {
        return undefined;
    }
    const field: unknown = Reflect.get(value, key);
    return field;
}

/** The text parsed as JSON, or the text itself when it is none. */
export function parsedOrText(text: string): Json {
    try {
        const parsed: unknown = JSON.parse(text);
        return parsed;
    } catch {
        return text;
    }
}

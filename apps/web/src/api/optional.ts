/** A request whose failure the caller reads as "nothing": its value, or undefined. */
export async function optional<Value>(promise: Promise<Value>): Promise<undefined | Value> {
    try {
        return await promise;
    } catch {
        return undefined;
    }
}

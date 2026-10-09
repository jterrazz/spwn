/** The callback an event or a timer takes, which runs the async task and returns at once. */
export function inBackground<Arguments extends unknown[]>(
    task: (...args: Arguments) => Promise<unknown>,
): (...args: Arguments) => void {
    return (...args) => {
        void task(...args);
    };
}

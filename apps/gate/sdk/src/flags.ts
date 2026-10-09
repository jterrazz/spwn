/** The `--key value` and bare `--flag` arguments of a CLI call, by name; anything else is skipped. */
export function parseFlags(argv: readonly string[]): Record<string, string | true> {
    const flags: Record<string, string | true> = {};
    for (let index = 0; index < argv.length; index += 1) {
        const argument = argv[index] ?? '';
        if (!argument.startsWith('--')) {
            continue;
        }
        const next = argv[index + 1];
        if (next === undefined || next.startsWith('--')) {
            flags[argument.slice(2)] = true;
        } else {
            flags[argument.slice(2)] = next;
            index += 1;
        }
    }
    return flags;
}

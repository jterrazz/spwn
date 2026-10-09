/** The `--key value` and bare `--flag` arguments of a CLI call, by name; anything else is skipped. */
export function parseFlags(argv: readonly string[]): Record<string, string | true> {
    const flags: Record<string, string | true> = {};
    for (const [index, argument] of argv.entries()) {
        if (argument.startsWith('--')) {
            const next = argv[index + 1];
            flags[argument.slice(2)] = next === undefined || next.startsWith('--') ? true : next;
        }
    }
    return flags;
}

export function matchesQuery(query: string, texts: readonly string[]): boolean {
    const asked = query.trim().toLowerCase();
    return asked === '' || texts.some((text) => text.toLowerCase().includes(asked));
}

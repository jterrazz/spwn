import { matchesQuery } from '../../model/search';

export type Line = { number: number; text: string };

export function numbered(lines: readonly string[]): readonly Line[] {
    return lines.map((text, index) => ({ number: index + 1, text }));
}

export function filtered(lines: readonly Line[], query: string): readonly Line[] {
    return lines.filter((line) => matchesQuery(query, [line.text]));
}

export function countOf(shown: number, all: number, query: string): string {
    const lines = all === 1 ? 'line' : 'lines';
    return query.trim() === ''
        ? `${String(all)} ${lines}`
        : `${String(shown)} of ${String(all)} ${lines}`;
}

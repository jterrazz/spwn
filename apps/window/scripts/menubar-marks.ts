// value: the menu bar app's marks, read from apps/menubar/scripts/symbols.swift, their one owner, and written as
// value: the window's src/design/menubar-marks.ts (menubar-marks.sh); its spec fails when the two drift. The reader
// value: knows the forms the script writes a mark in (a string, `+`, `\(strokes)`, a literal array's
// value: `.map { "…" }.joined()`) and throws on any other, so a new form is read before it is drawn.

type Part = { literal: string } | { name: string };

function stringAt(text: string, start: number): { end: number; parts: Part[] } {
    const parts: Part[] = [];
    let literal = '';
    let at = start + 1;
    while (text[at] !== '"') {
        if (at >= text.length) {
            throw new Error(`symbols.swift: a string opened at ${String(start)} never closes`);
        }
        if (text[at] === '\\' && text[at + 1] === '(') {
            const close = text.indexOf(')', at);
            parts.push({ literal }, { name: text.slice(at + 2, close) });
            literal = '';
            at = close + 1;
        } else if (text[at] === '\\') {
            literal += text[at + 1] ?? '';
            at += 2;
        } else {
            literal += text[at] ?? '';
            at += 1;
        }
    }
    parts.push({ literal });
    return { end: at + 1, parts };
}

function filled(parts: readonly Part[], values: Readonly<Record<string, string>>): string {
    return parts
        .map((part) => {
            if ('literal' in part) {
                return part.literal;
            }
            const value = values[part.name];
            if (value === undefined) {
                throw new Error(`symbols.swift: the window cannot read \\(${part.name})`);
            }
            return value;
        })
        .join('');
}

function rowsOf(array: string): Record<string, string>[] {
    const tuples = [...array.matchAll(/\((?<cells>[^)]*)\)/gu)].map((match) =>
        (match.groups?.cells ?? '').split(','),
    );
    const rows = tuples.length > 0 ? tuples : array.split(',').map((cell) => [cell]);
    const cells = rows.map((row) => row.map((cell) => cell.trim()));
    const doubles = (cells[0] ?? []).map((_, column) =>
        cells.some((row) => (row[column] ?? '').includes('.')),
    );
    return cells.map((row) => {
        const shown = row.map((cell, column) => {
            const value = Number(cell);
            return doubles[column] === true && Number.isInteger(value)
                ? `${String(value)}.0`
                : String(value);
        });
        const columns = Object.fromEntries(
            shown.map((cell, column) => [`$0.${String(column)}`, cell]),
        );
        return shown.length === 1 ? { ...columns, $0: shown[0] ?? '' } : columns;
    });
}

function evaluated(expression: string, values: Readonly<Record<string, string>>): string {
    let out = '';
    let at = 0;
    while (at < expression.length) {
        const char = expression[at];
        if (char === ' ' || char === '+') {
            at += 1;
        } else if (char === '"') {
            const read = stringAt(expression, at);
            out += filled(read.parts, values);
            at = read.end;
        } else if (char === '[') {
            const close = expression.indexOf(']', at);
            const map = '.map { ';
            if (!expression.startsWith(map, close + 1)) {
                throw new Error(
                    `symbols.swift: the window cannot read ${expression.slice(close + 1)}`,
                );
            }
            const read = stringAt(expression, close + 1 + map.length);
            const joined = ' }.joined()';
            if (!expression.startsWith(joined, read.end)) {
                throw new Error(
                    `symbols.swift: the window cannot read ${expression.slice(read.end)}`,
                );
            }
            out += rowsOf(expression.slice(at + 1, close))
                .map((row) => filled(read.parts, { ...values, ...row }))
                .join('');
            at = read.end + joined.length;
        } else {
            throw new Error(`symbols.swift: the window cannot read ${expression.slice(at)}`);
        }
    }
    return out;
}

export function marksOf(swift: string): Record<string, string> {
    const strokes = /^let strokes = (?<expression>.+)$/mu.exec(swift)?.groups?.expression ?? '';
    const values = { strokes: evaluated(strokes, {}) };
    const block =
        /^let marks: \[String: String\] = \[\n(?<entries>[\s\S]*?)\n\]$/mu.exec(swift)?.groups
            ?.entries ?? '';
    return Object.fromEntries(
        block.split('\n').map((line) => {
            const entry = /^ {4}"(?<name>[a-z]+)": (?<expression>.+),$/u.exec(line)?.groups;
            if (entry?.name === undefined || entry.expression === undefined) {
                throw new Error(`symbols.swift: the window cannot read the mark ${line}`);
            }
            return [entry.name, evaluated(entry.expression, values)];
        }),
    );
}

export function fileOf(marks: Readonly<Record<string, string>>): string {
    const entries = Object.entries(marks)
        .toSorted(([a], [b]) => a.localeCompare(b))
        .map(([name, body]) => {
            // invariant: oxfmt's layout (100 columns; a key of 7 letters or more puts a long string on the next
            // invariant: line), so a fix leaves the file as written.
            const line = `    ${name}: '${body}',`;
            return line.length > 100 && name.length >= 7
                ? `    ${name}:\n        '${body}',`
                : line;
        });
    return [
        "// value: the menu bar app's marks, each the body of an SVG on a 20 px box drawn in #000, as",
        '// value: apps/menubar/scripts/symbols.swift writes them; written by scripts/menubar-marks.sh, never by hand.',
        'export const MENUBAR_MARKS: Readonly<Record<string, string>> = {',
        ...entries,
        '};',
        '',
    ].join('\n');
}

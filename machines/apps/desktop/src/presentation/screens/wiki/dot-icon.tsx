import styles from './wiki.module.css';

export type Grids = Readonly<Record<string, null | readonly string[]>>;

// value: the grids of the icons the engine ships by name (wiki/kinds.go, builtinIcons): its kinds, the six intents and
// value: a rule's force. The engine names them, and their drawing is the window's, as a mark is: identity, never a
// value: judgement. A corpus's own icons come with each answer, and win over these.
const BUILT_IN: Grids = {
    allowed: ['....#', '...##', '#.##.', '###..', '.#...'],
    claim: ['..#..', '..#..', '..#..', '.....', '..#..'],
    component: ['##.##', '##.##', '.....', '##.##', '##.##'],
    decide: ['..#..', '..#..', '#####', '..#..', '..#..'],
    decision: ['..#..', '.#.#.', '#...#', '.#.#.', '..#..'],
    define: ['###..', '#....', '###..', '#....', '###..'],
    figure: ['#....', '#..#.', '#.##.', '####.', '#####'],
    folder: ['##...', '#####', '#...#', '#...#', '#####'],
    group: ['##.##', '#...#', '.....', '#...#', '##.##'],
    guard: ['#####', '#...#', '#...#', '.#.#.', '..#..'],
    hard: ['#####', '#####', '#...#', '#####', '#####'],
    hardened: ['#####', '#.#.#', '#####', '#.#.#', '#####'],
    link: ['##...', '#.#..', '.#.#.', '..#.#', '...##'],
    locate: ['.###.', '#...#', '#.#.#', '#...#', '.###.'],
    page: ['####.', '#..##', '#...#', '#...#', '#####'],
    prove: ['....#', '...#.', '#.#..', '.#...', '.....'],
    refused: ['#...#', '.#.#.', '..#..', '.#.#.', '#...#'],
    rule: ['#.#.#', '#.#.#', '#.#.#', '#.#.#', '#.#.#'],
    scenario: ['#....', '##...', '###..', '##...', '#....'],
    soft: ['#...#', '.....', '..#..', '.....', '#...#'],
    table: ['#####', '.....', '#####', '.....', '#####'],
    tell: ['#####', '#...#', '#...#', '####.', '#....'],
    term: ['.###.', '#...#', '#####', '#....', '.###.'],
};

type Dot = { x: number; y: number };

function dotsOf(rows: readonly string[]): Dot[] {
    return rows.flatMap((row, y) =>
        row.split('').flatMap((cell, x) => (cell === '#' ? [{ x, y }] : [])),
    );
}

export function DotIcon({
    name,
    grids,
    size,
}: {
    name: string;
    grids: Grids;
    size: 'large' | 'mid' | 'small';
}) {
    const rows = grids[name] ?? BUILT_IN[name] ?? null;
    if (rows === null || rows.length === 0) {
        return null;
    }
    const width = Math.max(...rows.map((row) => row.length));
    return (
        <svg
            aria-hidden="true"
            className={`${styles.icon} ${styles[size]}`}
            viewBox={`0 0 ${String(width)} ${String(rows.length)}`}
        >
            {dotsOf(rows).map(({ x, y }) => (
                <circle cx={x + 0.5} cy={y + 0.5} key={`${String(x)}-${String(y)}`} r={0.36} />
            ))}
        </svg>
    );
}

const DATA_CODEWORDS = [19, 34, 55, 80, 108] as const;
const EC_CODEWORDS = [7, 10, 15, 20, 26] as const;
const ALIGNMENT = [0, 18, 22, 26, 30] as const;
const FORMAT_LOW_MASK_0 = 8;

export type QrMatrix = readonly (readonly boolean[])[];

type Grid = { size: number; dark: boolean[][]; fixed: boolean[][] };

function multiply(x: number, y: number): number {
    let z = 0;
    for (let bit = 7; bit >= 0; bit -= 1) {
        z = (z << 1) ^ ((z >>> 7) * 0x1_1d);
        z ^= ((y >>> bit) & 1) * x;
    }
    return z;
}

function divisorOf(degree: number): number[] {
    const result = Array.from({ length: degree }, () => 0);
    result[degree - 1] = 1;
    let root = 1;
    for (let i = 0; i < degree; i += 1) {
        for (let j = 0; j < degree; j += 1) {
            result[j] = multiply(result[j] ?? 0, root) ^ (result[j + 1] ?? 0);
        }
        root = multiply(root, 2);
    }
    return result;
}

function remainderOf(data: readonly number[], divisor: readonly number[]): number[] {
    const result = divisor.map(() => 0);
    for (const byte of data) {
        const factor = byte ^ (result.shift() ?? 0);
        result.push(0);
        for (const [index, coefficient] of divisor.entries()) {
            result[index] = (result[index] ?? 0) ^ multiply(coefficient, factor);
        }
    }
    return result;
}

function codewordsOf(bytes: readonly number[], capacity: number): number[] {
    const bits: number[] = [];
    const push = (value: number, length: number) => {
        for (let bit = length - 1; bit >= 0; bit -= 1) {
            bits.push((value >>> bit) & 1);
        }
    };
    push(0b0100, 4);
    push(bytes.length, 8);
    for (const byte of bytes) {
        push(byte, 8);
    }
    push(0, Math.min(4, capacity * 8 - bits.length));
    push(0, (8 - (bits.length % 8)) % 8);
    const words: number[] = [];
    for (let index = 0; index < bits.length; index += 8) {
        words.push(bits.slice(index, index + 8).reduce((word, bit) => word * 2 + bit, 0));
    }
    for (let pad = 0xec; words.length < capacity; pad = pad === 0xec ? 0x11 : 0xec) {
        words.push(pad);
    }
    return words;
}

function set(grid: Grid, x: number, y: number, dark: boolean) {
    const row = grid.dark[y];
    const held = grid.fixed[y];
    if (row && held && x >= 0 && x < grid.size) {
        row[x] = dark;
        held[x] = true;
    }
}

function square(
    grid: Grid,
    [cx, cy]: readonly [number, number],
    radius: number,
    light: readonly number[],
) {
    for (let dy = -radius; dy <= radius; dy += 1) {
        for (let dx = -radius; dx <= radius; dx += 1) {
            const ring = Math.max(Math.abs(dx), Math.abs(dy));
            set(grid, cx + dx, cy + dy, !light.includes(ring));
        }
    }
}

function drawPatterns(grid: Grid, version: number) {
    const { size } = grid;
    for (let i = 0; i < size; i += 1) {
        set(grid, 6, i, i % 2 === 0);
        set(grid, i, 6, i % 2 === 0);
    }
    square(grid, [3, 3], 4, [2, 4]);
    square(grid, [size - 4, 3], 4, [2, 4]);
    square(grid, [3, size - 4], 4, [2, 4]);
    const align = ALIGNMENT[version] ?? 0;
    if (align > 0) {
        square(grid, [align, align], 2, [1]);
    }
}

function drawFormat(grid: Grid) {
    const { size } = grid;
    let remainder = FORMAT_LOW_MASK_0;
    for (let i = 0; i < 10; i += 1) {
        remainder = (remainder << 1) ^ ((remainder >>> 9) * 0x5_37);
    }
    const bits = ((FORMAT_LOW_MASK_0 << 10) | remainder) ^ 0x54_12;
    const bit = (index: number) => ((bits >>> index) & 1) === 1;
    for (let i = 0; i <= 5; i += 1) {
        set(grid, 8, i, bit(i));
    }
    set(grid, 8, 7, bit(6));
    set(grid, 8, 8, bit(7));
    set(grid, 7, 8, bit(8));
    for (let i = 9; i < 15; i += 1) {
        set(grid, 14 - i, 8, bit(i));
    }
    for (let i = 0; i < 8; i += 1) {
        set(grid, size - 1 - i, 8, bit(i));
    }
    for (let i = 8; i < 15; i += 1) {
        set(grid, 8, size - 15 + i, bit(i));
    }
    set(grid, 8, size - 8, true);
}

function drawData(grid: Grid, words: readonly number[]) {
    const { size } = grid;
    let index = 0;
    for (let right = size - 1; right >= 1; right -= 2) {
        const column = right <= 6 ? right - 1 : right;
        const upward = ((column + 1) & 2) === 0;
        for (let step = 0; step < size; step += 1) {
            const y = upward ? size - 1 - step : step;
            for (const x of [column, column - 1]) {
                const row = grid.dark[y];
                if (row && grid.fixed[y]?.[x] === false) {
                    const word = words[index >>> 3] ?? 0;
                    const dark = ((word >>> (7 - (index % 8))) & 1) === 1;
                    row[x] = dark !== ((x + y) % 2 === 0);
                    index += 1;
                }
            }
        }
    }
}

export function qrOf(text: string): QrMatrix | undefined {
    const bytes = [...new TextEncoder().encode(text)];
    const version = DATA_CODEWORDS.findIndex((capacity) => capacity * 8 >= bytes.length * 8 + 12);
    if (version === -1) {
        return undefined;
    }
    const size = 21 + 4 * version;
    const blank = () =>
        Array.from({ length: size }, () => Array.from({ length: size }, () => false));
    const grid: Grid = { dark: blank(), fixed: blank(), size };
    drawPatterns(grid, version);
    drawFormat(grid);
    const data = codewordsOf(bytes, DATA_CODEWORDS[version] ?? 0);
    drawData(grid, [...data, ...remainderOf(data, divisorOf(EC_CODEWORDS[version] ?? 0))]);
    return grid.dark;
}

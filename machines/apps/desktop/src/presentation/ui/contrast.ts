export type Rgb = { red: number; green: number; blue: number };

export function rgbOf(hex: string): Rgb {
    const found = /^#(?<red>[0-9a-f]{2})(?<green>[0-9a-f]{2})(?<blue>[0-9a-f]{2})$/iu.exec(hex);
    if (found?.groups === undefined) {
        throw new Error(`${hex} is not a six-digit hex colour`);
    }
    const { blue, green, red } = found.groups;
    return {
        blue: Number.parseInt(blue ?? '', 16),
        green: Number.parseInt(green ?? '', 16),
        red: Number.parseInt(red ?? '', 16),
    };
}

function linear(channel: number): number {
    const unit = channel / 255;
    return unit <= 0.04045 ? unit / 12.92 : ((unit + 0.055) / 1.055) ** 2.4;
}

function luminance({ blue, green, red }: Rgb): number {
    return 0.2126 * linear(red) + 0.7152 * linear(green) + 0.0722 * linear(blue);
}

// value: WCAG 2's contrast ratio, (lighter + 0.05) / (darker + 0.05), the measure ADR-093's thresholds are written in.
export function ratio(one: Rgb, other: Rgb): number {
    const lights = [luminance(one), luminance(other)];
    return (Math.max(...lights) + 0.05) / (Math.min(...lights) + 0.05);
}

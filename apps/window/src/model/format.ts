import type { CoreFindingFix } from '@jterrazz/station-api';

// invariant: station's rule of the numbers, as its Go prints them (apps/engine/activity/words.go): disk and network sizes
// invariant: and rates are decimal, memory binary and written GB as macOS shows RAM; one decimal under 100.
const DECIMAL_UNITS = [
    { of: 1e12, unit: 'TB' },
    { of: 1e9, unit: 'GB' },
    { of: 1e6, unit: 'MB' },
    { of: 1e3, unit: 'KB' },
] as const;
const GIB = 1024 ** 3;
const MIB = 1024 ** 2;
const NS_PER_SECOND = 1_000_000_000;
const NS_PER_MINUTE = 60 * NS_PER_SECOND;
const NEVER = '0001-01-01T00:00:00Z';

function decimal(value: number): string {
    if (value >= 100 || Number.isInteger(value)) {
        return String(Math.round(value));
    }
    return String(Math.round(value * 10) / 10);
}

export function sentence(parts: readonly (string | undefined)[]): string {
    return parts.filter((part) => part !== undefined && part !== '').join(' · ');
}

export function fixLine(fix: CoreFindingFix | undefined): string {
    return fix ? sentence([fix.Command, fix.Call, fix.Path]) : '';
}

export function bytes(value: number): string {
    const scale = DECIMAL_UNITS.find(({ of }) => value >= of);
    if (!scale) {
        return `${String(Math.round(value))} B`;
    }
    return `${decimal(Math.round((value / scale.of) * 10) / 10)} ${scale.unit}`;
}

export function memory(value: number): string {
    if (value <= 0) {
        return '0 B';
    }
    if (value >= GIB / 10) {
        return `${decimal(Math.round((value / GIB) * 10) / 10)} GB`;
    }
    return `${String(Math.round(value / MIB))} MB`;
}

export function percent(value: number): string {
    if (value > 0 && value < 1) {
        return value.toFixed(1);
    }
    return String(Math.round(value));
}

export function rate(perSecond: number): { value: string; unit: string } {
    if (perSecond < 1e6) {
        return { unit: 'KB/s', value: String(Math.round(perSecond / 1e3)) };
    }
    return { unit: 'MB/s', value: decimal(Math.round(perSecond / 1e5) / 10) };
}

export function duration(nanoseconds: number): string {
    const minutes = Math.floor(nanoseconds / NS_PER_MINUTE);
    const days = Math.floor(minutes / 1440);
    const hours = Math.floor((minutes % 1440) / 60);
    if (days > 0) {
        return hours > 0 ? `${String(days)} d ${String(hours)} h` : `${String(days)} d`;
    }
    if (hours === 0) {
        return `${String(minutes)} min`;
    }
    return minutes % 60 > 0 ? `${String(hours)} h ${String(minutes % 60)} min` : `${String(hours)} h`;
}

export function seconds(nanoseconds: number): string {
    const value = nanoseconds / NS_PER_SECOND;
    if (value < 60) {
        return `${String(Math.round(value))} s`;
    }
    if (value < 3600) {
        return `${String(Math.round(value / 60))} min`;
    }
    return `${String(Math.round(value / 3600))} h`;
}

function dateOf(at: Date | string | undefined): Date | undefined {
    if (at instanceof Date) {
        return at;
    }
    if (at === undefined || at === '' || at === NEVER) {
        return undefined;
    }
    const date = new Date(at);
    return Number.isNaN(date.getTime()) ? undefined : date;
}

function two(value: number): string {
    return String(value).padStart(2, '0');
}

export function clock(at: Date | string | undefined): string | undefined {
    const date = dateOf(at);
    return date ? `${two(date.getHours())}:${two(date.getMinutes())}` : undefined;
}

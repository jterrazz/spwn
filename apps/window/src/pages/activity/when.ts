import { seconds } from '../../model/format';

const MONTHS = [
    'Jan',
    'Feb',
    'Mar',
    'Apr',
    'May',
    'Jun',
    'Jul',
    'Aug',
    'Sep',
    'Oct',
    'Nov',
    'Dec',
] as const;
const NS_PER_MS = 1_000_000;

function dateOf(at: string | undefined): Date | undefined {
    if (at === undefined || at === '') {
        return undefined;
    }
    const date = new Date(at);
    return Number.isNaN(date.getTime()) || date.getFullYear() < 2 ? undefined : date;
}

export function shortDay(at: string | undefined): string | undefined {
    const date = dateOf(at);
    return date ? `${String(date.getDate())} ${MONTHS[date.getMonth()] ?? ''}` : undefined;
}

export function between(from: string | undefined, to: string | undefined): number | undefined {
    const start = dateOf(from);
    const end = dateOf(to);
    if (!start || !end || end < start) {
        return undefined;
    }
    return (end.getTime() - start.getTime()) * NS_PER_MS;
}

export function agoOf(at: string | undefined, now: string | undefined): string | undefined {
    const span = between(at, now);
    return span === undefined ? undefined : `${seconds(span)} ago`;
}

export function firstUnitOf(text: string): { value: string; unit: string } {
    const [value = text, unit = ''] = text.split(' ');
    return { unit, value };
}

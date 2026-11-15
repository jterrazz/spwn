import type { AccountsProgramRow, AccountsStep } from '@jterrazz/station-api';

import { detailOf } from '../../engine/engine-words';
import { sentence } from '../../model/format';

function howOf({ How, EntryLabel }: AccountsProgramRow): string | undefined {
    return How === undefined
        ? undefined
        : detailOf('AccountsProgramHow', How).replace('%s', EntryLabel ?? '');
}

// invariant: the row's one line of the window's own: the engine's fields joined, and its Line where the row's
// invariant: action column does not already say it.
export function secondLine(row: AccountsProgramRow, withLine: boolean): string {
    return sentence([
        row.Who,
        (row.Scopes ?? []).join(', '),
        row.Plan,
        howOf(row),
        withLine ? row.Line : undefined,
    ]);
}

const SECOND = 1000;
const MINUTE = 60 * SECOND;

function twoDigits(value: number): string {
    return String(value).padStart(2, '0');
}

// invariant: the time a code has left is its Expires less the Now the same step was sent with, never this
// invariant: window's clock; between two steps it stays as the last one told.
export function codeLeft(step: Pick<AccountsStep, 'Expires' | 'Now'>): string {
    const left = Math.max(0, Date.parse(step.Expires) - Date.parse(step.Now));
    if (Number.isNaN(left)) {
        return '';
    }
    return `${twoDigits(Math.floor(left / MINUTE))}:${twoDigits(Math.floor((left % MINUTE) / SECOND))}`;
}

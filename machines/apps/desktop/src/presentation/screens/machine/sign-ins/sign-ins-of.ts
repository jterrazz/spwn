import type { AccountsProgramRow } from '@jterrazz/station-api';

import { sentence } from '../../../../domain/format';
import { detailOf } from '../../../words/engine-words';

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

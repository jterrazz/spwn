import type {
    CoreJournalReading,
    CoreJournalUnheard,
    JournalReadInput,
} from '@jterrazz/station-api';

export type Paging = { readonly chosen: null | string; readonly asked: readonly string[] };

export function pagingOf(chosen: null | string): Paging {
    return { asked: [], chosen };
}

export function askedOf(paging: Paging, chosen: null | string): readonly string[] {
    return paging.chosen === chosen ? paging.asked : [];
}

export function olderAsked(paging: Paging, chosen: null | string, before: string): Paging {
    const asked = askedOf(paging, chosen);
    return { asked: asked.includes(before) ? asked : [...asked, before], chosen };
}

export function pageInput(chosen: null | string, before?: string): JournalReadInput {
    return {
        ...(chosen === null ? {} : { Mac: chosen }),
        ...(before === undefined ? {} : { Before: before }),
    };
}

export function olderOffered({ Next }: Pick<CoreJournalReading, 'Next'>): string | undefined {
    return Next === '' ? undefined : Next;
}

export function unheardUnsaid(
    said: readonly string[],
    { Unheard }: Pick<CoreJournalReading, 'Unheard'>,
): readonly CoreJournalUnheard[] {
    return (Unheard ?? []).filter(({ Mac }) => !said.includes(Mac));
}

export function saidAfter(
    said: readonly string[],
    reading: Pick<CoreJournalReading, 'Unheard'>,
): readonly string[] {
    return [...said, ...unheardUnsaid(said, reading).map(({ Mac }) => Mac)];
}

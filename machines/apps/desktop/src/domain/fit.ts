// value: the revisions of `station.page/v1` this window draws, the oldest it accepts to the newest it knows, as the
// value: menu bar declares its own (`pageOldest`, `pageRevision`); `fit.test.ts` holds the newest to the tree's engine's.
export const PAGE_OLDEST = 5;
export const PAGE_NEWEST = 5;

// value: ADR-070's three verdicts, in the codes its `opened` frame is to carry as `fit`.
export type Fit = 'app-older' | 'engine-older' | 'fits';

export type Unfit = Exclude<Fit, 'fits'>;

export type Range = { oldest: number; newest: number };

const WINDOW: Range = { newest: PAGE_NEWEST, oldest: PAGE_OLDEST };

// invariant: the one compatibility verdict of docs/reference/api.md: below the oldest accepted, or no revision named,
// invariant: the engine is older; above the newest known, this window is; anywhere between, it fits.
export function fitOf(revision: number | undefined, range: Range = WINDOW): Fit {
    if (revision === undefined || revision < range.oldest) {
        return 'engine-older';
    }
    return revision > range.newest ? 'app-older' : 'fits';
}

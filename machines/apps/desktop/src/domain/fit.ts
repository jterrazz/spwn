// value: the one revision of `station.page/v1` this window draws, as the menu bar declares its own (`pageRevision`):
// value: until 1.0 there is no range (ADR-072); `fit.test.ts` holds it to the tree's engine's.
export const PAGE_REVISION = 5;

// value: the one revision of `station.state/v1` this window reads, as `engine.state` judges it (ADR-072).
export const STATE_REVISION = 1;

// value: ADR-070's three verdicts, in the codes its `opened` frame is to carry as `fit`.
export type Fit = 'app-older' | 'engine-older' | 'fits';

export type Unfit = Exclude<Fit, 'fits'>;

// invariant: the one compatibility verdict of docs/reference/api.md: below the revision drawn, or none named, the
// invariant: engine is older; above it, this window is; only that revision fits.
export function fitOf(revision: number | undefined, drawn: number = PAGE_REVISION): Fit {
    if (revision === undefined || revision < drawn) {
        return 'engine-older';
    }
    return revision > drawn ? 'app-older' : 'fits';
}

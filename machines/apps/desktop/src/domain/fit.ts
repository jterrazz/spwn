// value: the one revision of `station.state/v1` this window reads, as `engine.state` judges it (ADR-072).
export const STATE_REVISION = 1;

// value: ADR-070's three verdicts, the codes its `opened` frame carries as `fit`.
export type Fit = 'app-older' | 'engine-older' | 'fits';

export type Unfit = Exclude<Fit, 'fits'>;

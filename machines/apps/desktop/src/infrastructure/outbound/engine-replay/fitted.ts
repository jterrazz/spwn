import type { Unfit } from '../../../domain/fit';

const UNFIT: readonly Unfit[] = ['app-older', 'engine-older'];

// value: `?fit=engine-older` or `app-older`, the verdict a world's engine opens with; anything else fits.
export function fittedOf(fit: null | string): undefined | Unfit {
    return UNFIT.find((unfit) => unfit === fit);
}

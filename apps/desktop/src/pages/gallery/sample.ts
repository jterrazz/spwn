import { createContext, useContext } from 'react';

import type { Dependencies } from '../../engine/react';

export const SampleContext = createContext<Dependencies | null>(null);

export function useSample(): Dependencies {
    const sample = useContext(SampleContext);
    if (!sample) {
        throw new Error('A gallery sample is drawn inside the Gallery.');
    }
    return sample;
}

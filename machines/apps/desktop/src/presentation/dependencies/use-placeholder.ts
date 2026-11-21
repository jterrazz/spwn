import { useMemo, useState } from 'react';

import type { Reading } from '../../application/use-cases/entry';

export type Asking = { placeholder?: boolean };

function placeheld<T>(reading: Reading<T>, shown: Reading<T>): Reading<T> {
    if (reading.kind !== 'reading' || shown.kind !== 'read') {
        return reading;
    }
    return { ...shown, placeholder: true };
}

export function usePlaceholder<T>(
    reading: Reading<T>,
    { placeholder = false }: Asking,
): Reading<T> {
    const [shown, setShown] = useState(reading);
    if (placeholder && reading.kind === 'read' && reading !== shown) {
        setShown(reading);
    }
    const held = useMemo(() => placeheld(reading, shown), [reading, shown]);
    return placeholder ? held : reading;
}

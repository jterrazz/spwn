'use client';

import { useEffect, useMemo } from 'react';

/**
 * Sets the document title dynamically.
 * Parts are joined with ' · ' and always ends with 'spwn'.
 * Example: usePageTitle('neo', 'Rhea') → 'neo · Rhea · spwn'
 */
export function usePageTitle(...parts: (null | string | undefined)[]) {
    // Materialise the title once per render so the effect dep list is a single string.
    const title = useMemo(() => {
        const filtered = parts.filter(Boolean) as string[];
        return filtered.length === 0 ? 'spwn' : [...filtered, 'spwn'].join(' · ');
        // eslint-disable-next-line react-hooks/exhaustive-deps -- rest params change identity every call; we intentionally key on the joined string
    }, [parts.join(',')]);

    useEffect(() => {
        document.title = title;
    }, [title]);
}

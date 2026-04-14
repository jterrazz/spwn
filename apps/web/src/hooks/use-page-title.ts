'use client';

import { useEffect } from 'react';

/**
 * Sets the document title dynamically.
 * Parts are joined with ' · ' and always ends with 'spwn'.
 * Example: usePageTitle('neo', 'Rhea') → 'neo · Rhea · spwn'
 */
export function usePageTitle(...parts: (null | string | undefined)[]) {
    useEffect(() => {
        const filtered = parts.filter(Boolean) as string[];
        document.title = filtered.length === 0 ? 'spwn' : [...filtered, 'spwn'].join(' · ');
    }, [parts.join(',')]);
}

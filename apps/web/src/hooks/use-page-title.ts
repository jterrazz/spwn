'use client';

import { useEffect } from 'react';

function titleOf(parts: (null | string | undefined)[]): string {
    const named = parts.filter(
        (part): part is string => part !== null && part !== undefined && part !== '',
    );
    return [...named, 'spwn'].join(' · ');
}

/** Sets the document title: the parts joined with ' · ', ending with 'spwn' — usePageTitle('neo', 'Rhea') → 'neo · Rhea · spwn'. */
export function usePageTitle(...parts: (null | string | undefined)[]) {
    const title = titleOf(parts);

    useEffect(() => {
        document.title = title;
    }, [title]);
}

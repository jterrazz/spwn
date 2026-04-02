"use client";

import { useEffect } from "react";

/**
 * Sets the document title dynamically.
 * Parts are joined with ' · ' and always ends with 'Observatory'.
 * Example: usePageTitle('neo', 'Rhea') → 'neo · Rhea · Observatory'
 */
export function usePageTitle(...parts: (string | undefined | null)[]) {
  useEffect(() => {
    const filtered = parts.filter(Boolean) as string[];
    if (filtered.length === 0) {
      document.title = "Observatory · spwn";
    } else {
      document.title = [...filtered, "Observatory"].join(" · ");
    }
  }, [parts.join(",")]);
}

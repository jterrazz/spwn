"use client";

import { useEffect } from "react";

/**
 * Sets the document title dynamically.
 * Parts are joined with ' · ' and always ends with 'spwn'.
 * Example: usePageTitle('neo', 'Rhea') → 'neo · Rhea · spwn'
 */
export function usePageTitle(...parts: (string | undefined | null)[]) {
  useEffect(() => {
    const filtered = parts.filter(Boolean) as string[];
    if (filtered.length === 0) {
      document.title = "spwn";
    } else {
      document.title = [...filtered, "spwn"].join(" · ");
    }
  }, [parts.join(",")]);
}

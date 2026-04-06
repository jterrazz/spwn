"use client";

import { useState, useEffect, useRef } from "react";

export interface ProgressStep {
  /** Seconds after start when this message activates */
  after: number;
  text: string;
}

/**
 * Returns a message string that automatically advances through the given
 * steps based on elapsed time while `active` is true. Resets when `active`
 * flips back to false.
 */
export function useProgressMessages(active: boolean, messages: ProgressStep[]): string {
  const fallback = messages[0]?.text ?? "";
  const [message, setMessage] = useState(fallback);
  // Keep a stable ref to messages so the effect doesn't re-run on every render
  const msgsRef = useRef(messages);
  msgsRef.current = messages;

  useEffect(() => {
    if (!active) {
      setMessage(msgsRef.current[0]?.text ?? "");
      return;
    }
    const start = Date.now();
    // Set immediately
    setMessage(msgsRef.current[0]?.text ?? "");
    const id = setInterval(() => {
      const elapsed = (Date.now() - start) / 1000;
      let current = msgsRef.current[0]?.text ?? "";
      for (const m of msgsRef.current) {
        if (elapsed >= m.after) current = m.text;
      }
      setMessage(current);
    }, 1000);
    return () => clearInterval(id);
  }, [active]);

  return message;
}

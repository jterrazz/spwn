'use client';

import { useEffect, useState } from 'react';

export interface ProgressStep {
    /** Seconds after start when this message activates */
    after: number;
    text: string;
}

function messageAt(steps: ProgressStep[], elapsed: number): string {
    return steps.findLast((step) => elapsed >= step.after)?.text ?? steps[0]?.text ?? '';
}

/** Returns a message string that automatically advances through the given steps based on elapsed time while `active` is true. Resets when `active` flips back to false. */
export function useProgressMessages(active: boolean, messages: ProgressStep[]): string {
    const [elapsed, setElapsed] = useState(0);

    useEffect(() => {
        if (!active) {
            return;
        }
        const start = Date.now();
        const id = setInterval(() => {
            setElapsed((Date.now() - start) / 1000);
        }, 1000);
        return () => {
            clearInterval(id);
            setElapsed(0);
        };
    }, [active]);

    return messageAt(messages, active ? elapsed : 0);
}

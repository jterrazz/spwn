'use client';

import { useCallback, useState } from 'react';

const SHOWN_FOR_MS = 2500;

/** A one-line notice shown for a moment after an action: its outcome, or 'Error: …'. */
export function useFeedback() {
    const [feedback, setFeedback] = useState<null | string>(null);
    const show = useCallback((message: string) => {
        setFeedback(message);
        setTimeout(() => {
            setFeedback(null);
        }, SHOWN_FOR_MS);
    }, []);
    return { feedback, show };
}

export function FeedbackLine({ feedback }: { feedback: null | string }) {
    if (feedback === null || feedback === '') {
        return null;
    }
    const tone = feedback.startsWith('Error')
        ? 'border border-red-500/20 bg-red-500/10 text-red-400'
        : 'border border-green-500/20 bg-green-500/10 text-green-400';
    return (
        <div
            className={`animate-in fade-in slide-in-from-top-2 rounded-lg px-4 py-2 font-mono text-xs duration-200 ${tone}`}
        >
            {feedback}
        </div>
    );
}

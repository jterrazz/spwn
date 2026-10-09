'use client';

import { useEffect } from 'react';

import { inBackground } from '../events/in-background';

/** Runs an async task once the component mounts, then every `every` milliseconds when given, until it unmounts or the task changes. */
export function useBackgroundTask(task: () => Promise<void>, every?: number): void {
    useEffect(() => {
        void task();
        if (every === undefined) {
            return;
        }
        const id = setInterval(inBackground(task), every);
        return () => {
            clearInterval(id);
        };
    }, [task, every]);
}

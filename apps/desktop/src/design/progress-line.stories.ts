import type { ProgressLineProps } from './progress-line';
import { SECRET_SAMPLE } from './story';
import type { Story } from './story';

export const PROGRESS_LINE_STORIES: readonly Story<ProgressLineProps>[] = [
    { name: 'running', props: { task: 'Updating the mise apps' } },
    { name: 'running, its detail', props: { detail: 'node 24.11.1', task: 'Installing' } },
    {
        name: 'failed',
        props: { detail: 'brew exited 1', failed: true, task: 'Updating the brew apps' },
    },
    { name: 'withheld', props: { detail: SECRET_SAMPLE, task: 'Signing in' } },
    {
        name: 'long text',
        props: {
            detail: 'downloading cirruslabs/ubuntu:24.04 from the registry, 3.1 of 4.8 GB',
            task: 'Creating the workspace a-machine-name-long-enough-to-be-cut',
        },
    },
];

import type { MemoryTrackProps } from './memory-track';
import type { Story } from './story';

export const MEMORY_TRACK_STORIES: readonly Story<MemoryTrackProps>[] = [
    {
        name: 'a Mac: macOS, then its machines',
        props: {
            label: 'Memory of mac-mini',
            segments: [
                { at: 0, identity: 'system', key: 'macos', used: 1, width: 0.0625 },
                { at: 0.0625, identity: 'pink', key: 'infra', used: 0.67, width: 0.375 },
                { at: 0.4375, identity: 'sky', key: 'linux-3', used: 0.66, width: 0.125 },
                { at: 0.5625, identity: 'lilac', key: 'review-bot', used: 0.68, width: 0.0625 },
            ],
        },
    },
    {
        name: 'one machine at its place',
        props: {
            label: 'Memory of linux-1',
            segments: [{ at: 0.25, identity: 'teal', key: 'linux-1', used: 0.66, width: 0.1667 }],
        },
    },
    {
        name: 'a share being made, nothing used yet',
        props: {
            label: 'Memory of linux-2',
            segments: [{ at: 0.4167, identity: 'indigo', key: 'linux-2', used: 0, width: 0.1667 }],
        },
    },
    {
        name: 'a share whose use is unknown',
        props: {
            label: 'Memory of agent-7f',
            segments: [{ at: 0.0833, identity: 'blue', key: 'agent-7f', width: 0.1667 }],
        },
    },
    { name: 'no share: the empty track', props: { label: 'Memory of api', segments: [] } },
    {
        name: 'past the end, held inside the track',
        props: {
            label: 'Memory of over',
            segments: [{ at: 0.9, identity: 'purple', key: 'over', used: 3, width: 0.5 }],
        },
    },
];

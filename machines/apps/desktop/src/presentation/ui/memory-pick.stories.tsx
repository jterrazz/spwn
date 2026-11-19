import type { MemoryPickProps } from './memory-pick';
import type { Story } from './story';
import { LONG_NAME } from './story';

export const MEMORY_PICK_STORIES: readonly Story<MemoryPickProps>[] = [
    {
        name: 'a Mac with room, 8 GB chosen',
        props: {
            chosen: 8,
            free: 28,
            holders: [
                { identity: 'system', key: 'macos', label: 'macOS', squares: 4 },
                { identity: 'blue', key: 'infra', label: 'infra', squares: 24 },
            ],
            identity: 'purple',
            line: 'leaves 28 GB free on this Mac',
            name: 'mac-mini',
        },
    },
    {
        name: 'no room, nothing chosen',
        props: {
            chosen: 0,
            free: 1,
            holders: [
                { identity: 'system', key: 'macos', label: 'macOS', squares: 4 },
                { identity: 'teal', key: 'linux-1', label: 'linux-1', squares: 11 },
            ],
            line: '',
            name: 'macbook-pro',
        },
    },
    {
        name: 'a long name',
        props: {
            chosen: 4,
            free: 12,
            holders: [{ identity: 'system', key: 'macos', label: 'macOS', squares: 4 }],
            identity: 'blue',
            line: 'leaves 12 GB free on a-machine-name-long-enough-to-be-cut-before-it-ends-in-any-column',
            name: LONG_NAME,
        },
    },
];

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
            line: 'room for one more gate',
            name: 'jterrazz-mac-mini',
            title: 'Memory · 8 GB',
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
            line: 'no room for another machine',
            name: 'jterrazz-macbook-pro',
            title: 'Memory',
        },
    },
    {
        name: 'a long name',
        props: {
            chosen: 4,
            free: 12,
            holders: [{ identity: 'system', key: 'macos', label: 'macOS', squares: 4 }],
            identity: 'blue',
            line: 'room for a small machine',
            name: LONG_NAME,
            title: 'Memory · 4 GB',
        },
    },
];

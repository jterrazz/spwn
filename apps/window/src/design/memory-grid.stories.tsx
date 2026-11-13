import type { MemoryGridProps } from './memory-grid';
import type { Story } from './story';

export const MEMORY_GRID_STORIES: readonly Story<MemoryGridProps>[] = [
    {
        name: 'a Mac and its machines',
        props: {
            free: 34,
            holders: [
                { identity: 'system', key: 'macos', label: 'macOS', squares: 6 },
                { identity: 'blue', key: 'infra', label: 'infra', squares: 16 },
                { identity: 'purple', key: 'linux-1', label: 'linux-1', squares: 8 },
            ],
            name: 'jterrazz-mac-mini',
            total: 64,
        },
    },
    {
        name: 'a machine being created',
        props: {
            free: 4,
            holders: [
                { identity: 'system', key: 'macos', label: 'macOS', squares: 4 },
                { identity: 'teal', key: 'linux-5', label: 'linux-5', pending: true, squares: 8 },
            ],
            name: 'jterrazz-macbook-pro',
            total: 16,
        },
    },
    {
        name: 'no machine',
        props: {
            free: 12,
            holders: [{ identity: 'system', key: 'macos', label: 'macOS', squares: 4 }],
            name: 'jterrazz-macbook-pro',
            total: 16,
        },
    },
];

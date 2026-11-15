import type { BarProps, ShareBarProps } from './bar';
import type { Story } from './story';

export const BAR_STORIES: readonly Story<BarProps>[] = [
    { name: 'plain', props: { label: 'Disk', percent: 41 } },
    { name: 'an identity', props: { identity: 'purple', label: 'Memory of gate', percent: 72 } },
    { name: 'warn', props: { label: 'Memory of gate2', percent: 91, tone: 'warn' } },
    { name: 'bad', props: { label: 'Disk of relay', percent: 98, tone: 'bad' } },
    { name: 'ok', props: { label: 'Battery', percent: 64, tone: 'ok' } },
    { name: 'new', props: { label: 'Download', percent: 30, tone: 'new' } },
    { name: 'fixed width', props: { label: 'Disk', percent: 41, width: 'fixed' } },
    { name: 'a sliver, never nothing', props: { label: 'Swap', percent: 0.2 } },
    { name: 'past full, held at full', props: { label: 'Over', percent: 140 } },
    { name: 'missing: no total', props: { label: 'Unknown', percent: Number.NaN } },
    { name: 'missing: a total of zero', props: { label: 'Nothing', percent: 1 / 0 } },
];

export const SHARE_BAR_STORIES: readonly Story<ShareBarProps>[] = [
    {
        name: 'parts by their size',
        props: {
            label: 'Memory',
            parts: [
                { identity: 'blue', key: 'a', size: 8 },
                { identity: 'purple', key: 'b', size: 10 },
                { key: 'free', rest: true, size: 10 },
            ],
        },
    },
    {
        name: 'parts of a whole',
        props: {
            label: 'Disk',
            of: 994,
            parts: [
                { identity: 'teal', key: 'apps', label: 'Apps', size: 212 },
                { identity: 'pink', key: 'vms', label: 'Machines', size: 180 },
            ],
        },
    },
    { name: 'empty', props: { label: 'Nothing', parts: [] } },
    {
        name: 'missing: a total of zero',
        props: { label: 'Nothing measured', of: 0, parts: [{ key: 'apps', size: 3 }] },
    },
];

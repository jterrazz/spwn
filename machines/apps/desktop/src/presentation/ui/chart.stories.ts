import type { ChartProps } from './chart';
import type { Story } from './story';

const RISING = [40, 52, 61, 70, 77, 80, 84, 88];

export const CHART_STORIES: readonly Story<ChartProps>[] = [
    { name: 'tile', props: { label: 'CPU', values: RISING } },
    { name: 'tile, warn', props: { label: 'CPU', tone: 'warn', values: RISING } },
    { name: 'tile, bad', props: { label: 'CPU', tone: 'bad', values: [88, 92, 97, 99, 98, 99] } },
    {
        name: 'row',
        props: { label: 'CPU of relay', size: 'row', values: [3, 4, 3, 5, 4, 3, 2, 3] },
    },
    {
        name: 'activity, new',
        props: {
            label: 'Network',
            size: 'activity',
            tone: 'new',
            values: Array.from({ length: 60 }, (_, index) => 20 + ((index * 37) % 60)),
        },
    },
    {
        name: 'marks under it',
        props: { label: 'CPU by hour', marks: ['9:00', '12:00', '15:00'], values: RISING },
    },
    {
        name: 'missing: asleep twice',
        props: { label: 'CPU', values: [30, 42, null, null, 38, 35, null, 40] },
    },
    { name: 'missing: never measured', props: { label: 'CPU', values: [null, null, null] } },
    { name: 'empty', props: { label: 'CPU', values: [] } },
];

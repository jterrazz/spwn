import { Bar, ShareBar } from './bar';
import { Chart } from './chart';
import { MissingMark } from './missing';
import type { Story } from './story';
import type { TileProps } from './tile';

export const TILE_STORIES: readonly Story<TileProps>[] = [
    {
        name: 'a share bar',
        props: {
            foot: 'of 128 · room for 4 more gates',
            label: 'Memory free',
            unit: 'GB',
            value: 48,
            viz: (
                <ShareBar
                    label="Memory"
                    parts={[
                        { identity: 'blue', key: 'a', size: 8 },
                        { identity: 'purple', key: 'b', size: 10 },
                        { key: 'free', rest: true, size: 10 },
                    ]}
                />
            ),
        },
    },
    {
        name: 'a chart, warn',
        props: {
            foot: 'Peak 91 % · 14 min ago',
            label: 'CPU',
            unit: '%',
            value: 88,
            viz: <Chart label="CPU" tone="warn" values={[40, 52, 61, 70, 77, 80, 84, 88]} />,
        },
    },
    {
        name: 'an aside',
        props: {
            aside: { text: 'On', tone: 'ok' },
            foot: 'on power',
            label: 'Awake, this Mac',
            unit: 'h left',
            value: 5,
        },
    },
    {
        name: 'a bar',
        props: {
            foot: 'of 994',
            label: 'Disk free',
            unit: 'GB',
            value: 582,
            viz: <Bar label="Disk" percent={41} width="fixed" />,
        },
    },
    {
        name: 'missing',
        props: { foot: 'no call measures it yet', label: 'Battery', value: <MissingMark /> },
    },
    {
        name: 'long text',
        props: {
            foot: 'the foot of a tile runs on when it has more to say than its width',
            label: 'Memory free across every machine',
            unit: 'GB',
            value: 1024,
        },
    },
];

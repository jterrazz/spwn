import type { ComponentProps } from 'react';

import type { Facts } from './facts';
import { MissingMark } from './missing';
import type { Story } from './story';
import { Value } from './value';

export const FACTS_STORIES: readonly Story<ComponentProps<typeof Facts>>[] = [
    {
        name: 'grid',
        props: {
            facts: [
                { label: 'Purpose', value: 'Design of the Station window' },
                { label: 'Created', value: '6 Nov, by studio-os-e8' },
                { label: 'Hold', value: 'Yours' },
                { label: 'Image', value: <Value>cirruslabs/ubuntu:24.04</Value> },
            ],
        },
    },
    {
        name: 'column',
        props: {
            facts: [
                { label: 'System', value: 'Ubuntu 24.04 · arm64' },
                { label: 'Resources', value: '4 CPU · 4 GB · 20 GB disk' },
            ],
            layout: 'column',
        },
    },
    {
        name: 'line',
        props: {
            facts: [
                { label: 'Name on the tailnet', value: 'a-long-machine-name.tail1234.ts.net' },
                { label: 'Serve', value: 'on' },
                { label: 'Range', value: '40000 to 40999 · 998 free · 20 per machine' },
            ],
            layout: 'line',
        },
    },
    {
        name: 'missing',
        props: {
            facts: [
                { label: 'System', value: 'macOS 26.1' },
                { label: 'Uptime', value: <MissingMark /> },
            ],
        },
    },
    {
        name: 'long text',
        props: {
            facts: [
                {
                    label: 'Machine file',
                    value: (
                        <Value>~/.config/station/machinefiles/a-long-machine-file-name.yaml</Value>
                    ),
                },
                {
                    label: 'Purpose',
                    value: 'A workspace for the design of the window, its gallery and every reference image it is held to',
                },
            ],
        },
    },
    { name: 'empty', props: { facts: [] } },
];

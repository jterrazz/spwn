import type { ComponentProps } from 'react';

import { AppIcon } from './app-icon';
import { Bar } from './bar';
import { Button } from './button';
import { Chart } from './chart';
import { MachineChip } from './machine-chip';
import { MachineIcon } from './machine-icon';
import type { RowHead, RowProps } from './row';
import { LONG_NAME, nothing, SECRET_SAMPLE } from './story';
import type { Story } from './story';
import { Value } from './value';

export const ROW_STORIES: readonly Story<RowProps>[] = [
    {
        name: 'a title and its sentence',
        props: { sentence: 'One tool that installs every runtime', title: 'mise' },
    },
    {
        name: 'a state first, and a gesture',
        props: {
            sentence: 'workspace · shared 11 h ago',
            state: { tone: 'warn', word: 'Needs you' },
            stateAt: 'start',
            title: 'GitHub shared with workspace ends in 40 min',
            trail: <Button>Extend 12 h</Button>,
        },
    },
    {
        name: 'columns: kind, value, a quiet state',
        props: {
            kind: 'manager',
            sentence: 'A calm state is a quiet word',
            state: { tone: 'ok', word: 'Present' },
            title: 'mise',
            value: <Value>2025.11.3</Value>,
        },
    },
    {
        name: 'a column kept empty',
        props: {
            kind: null,
            sentence: 'A deviation is a pill',
            state: { tone: 'warn', word: 'Outdated' },
            title: 'node',
            value: <Value>24.11.0</Value>,
        },
    },
    {
        name: 'a size, plain figures',
        props: { kind: 'cache', state: null, title: 'Docker images', value: '12.4 GB' },
    },
    { name: 'a tag', props: { sentence: 'From the catalog', tag: '24.11.1', title: 'node' } },
    {
        name: 'a tag as a hint',
        props: {
            sentence: 'https://vite.example.ts.net',
            tag: 'Private',
            tagLook: 'hint',
            title: 'Vite',
        },
    },
    {
        name: 'catalog template, slim',
        props: {
            cells: ['One tool that installs every runtime'],
            kind: 'manager',
            lead: <AppIcon name="mise" />,
            size: 'slim',
            state: { tone: 'ok', word: 'Installed' },
            template: 'catalog',
            title: 'mise',
            value: <Value look="raised">2025.11.3</Value>,
        },
    },
    {
        name: 'meters template, opens',
        props: {
            cells: [
                <Chart key="cpu" label="CPU" size="row" values={[50, 60, 64]} />,
                <Bar identity="purple" key="memory" label="Memory" percent={72} />,
            ],
            lead: <MachineIcon identity="purple" kind="workspace" />,
            leadSize: 'machine',
            onOpen: nothing,
            template: 'meters',
            title: 'gate',
        },
    },
    {
        name: 'meters template, nested under its Mac',
        props: {
            cells: ['12 %', '2 / 4'],
            lead: <MachineIcon identity="blue" kind="server" />,
            leadSize: 'machine',
            nested: true,
            template: 'meters',
            title: 'infra',
        },
    },
    {
        name: 'opens, and a chip that opens another',
        props: {
            link: <MachineChip kind="workspace" name="gate4" onOpen={nothing} />,
            onOpen: nothing,
            sentence: 'Opens the account; its chip opens the machine that holds it',
            title: 'GitHub',
        },
    },
    {
        name: 'a value lead',
        props: {
            lead: <Value look="raised">:3000</Value>,
            leadSize: 'value',
            sentence: 'node, vite',
            title: 'web',
        },
    },
    {
        name: 'missing',
        props: { sentence: '', state: null, title: 'process not given yet', value: null },
    },
    { name: 'withheld', props: { title: 'Token', value: <Value>{SECRET_SAMPLE}</Value> } },
    {
        name: 'long text',
        props: {
            sentence:
                'A path keeps its end: it is cut in the middle, and the sentence under it is cut at its end',
            state: { tone: 'plain', word: 'Not installed' },
            title: LONG_NAME,
            value: <Value>~/.config/station/machinefiles/jterrazz-macbook-pro.yaml</Value>,
        },
    },
];

export const ROW_HEAD_STORIES: readonly Story<ComponentProps<typeof RowHead>>[] = [
    {
        name: 'meters, a lead',
        props: {
            columns: [
                { label: 'Machine' },
                { label: 'CPU', numeric: true },
                { label: 'Memory, GB', numeric: true },
            ],
            lead: true,
            template: 'meters',
        },
    },
    {
        name: 'catalog',
        props: {
            columns: [
                { label: 'Name' },
                { label: 'What it is' },
                { label: 'Kind' },
                { label: 'Version' },
                { label: 'State' },
            ],
            template: 'catalog',
        },
    },
];

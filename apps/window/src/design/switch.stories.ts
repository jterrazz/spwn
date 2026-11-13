import { nothing } from './story';
import type { Story } from './story';
import type { SwitchProps } from './switch';

const WRAPPED = [
    'Package managers',
    'Runtimes',
    'Shell & terminal',
    'Command line',
    'Git',
    'Editors & IDEs',
    'Containers & VMs',
    'Deploy',
    'AI agents',
    'AI tooling',
    'Browsers',
    'Communication',
    'Productivity',
].map((label) => ({ id: label.toLowerCase(), label }));

const TABS = ['Overview', 'Activity', 'Ports'].map((label) => ({ id: label, label }));

const FILES = [
    { count: 2, id: 'mine', label: 'Yours' },
    { count: 5, id: 'built-in', label: 'Built-in' },
];

export const SWITCH_STORIES: readonly Story<SwitchProps<string>>[] = [
    {
        name: 'filter, all chosen',
        props: { all: 'All', label: 'Which files', onChange: nothing, options: FILES, value: null },
    },
    {
        name: 'filter, one chosen',
        props: {
            all: 'All',
            label: 'Which files',
            onChange: nothing,
            options: FILES,
            value: 'mine',
        },
    },
    {
        name: 'tabs',
        props: {
            label: 'Tabs',
            onChange: nothing,
            options: TABS,
            value: 'Overview',
            variant: 'tabs',
        },
    },
    {
        name: 'on a panel',
        props: {
            label: 'Chapter',
            onChange: nothing,
            options: TABS,
            surface: 'panel',
            value: 'Activity',
        },
    },
    {
        name: 'disabled',
        props: {
            disabled: true,
            label: 'Nothing to choose yet',
            onChange: nothing,
            options: TABS,
            value: 'Overview',
        },
    },
    {
        name: 'wraps, long text',
        props: {
            label: 'Categories',
            onChange: nothing,
            options: WRAPPED,
            value: 'runtimes',
            wrap: true,
        },
    },
    { name: 'empty', props: { label: 'No option', onChange: nothing, options: [], value: '' } },
];

import { MachineIcon } from './machine-icon';
import type { NavItemProps } from './nav-item';
import { LONG_NAME, nothing } from './story';
import type { Story } from './story';

export const NAV_ITEM_STORIES: readonly Story<NavItemProps>[] = [
    { name: 'current', props: { current: true, icon: 'home', label: 'Home', onSelect: nothing } },
    {
        name: 'an urgent count',
        props: {
            count: { urgent: true, value: 2 },
            current: false,
            icon: 'inbox',
            label: 'Inbox',
            onSelect: nothing,
        },
    },
    {
        name: 'a count, indented',
        props: {
            count: { urgent: false, value: 15 },
            current: false,
            icon: 'apps',
            indent: true,
            label: 'Apps',
            onSelect: nothing,
        },
    },
    {
        name: 'a machine and its state',
        props: {
            current: false,
            label: 'jterrazz-mac-mini',
            lead: <MachineIcon kind="mac" />,
            onSelect: nothing,
            state: { tone: 'ok', word: 'Up' },
        },
    },
    {
        name: 'a machine down',
        props: {
            current: false,
            indent: true,
            label: 'relay',
            lead: <MachineIcon identity="lilac" kind="server" />,
            onSelect: nothing,
            state: { tone: 'bad', word: 'Down' },
        },
    },
    {
        name: 'long text',
        props: {
            current: false,
            label: LONG_NAME,
            lead: <MachineIcon identity="indigo" kind="workspace" />,
            onSelect: nothing,
            state: { tone: 'plain', word: 'Stopped' },
        },
    },
];

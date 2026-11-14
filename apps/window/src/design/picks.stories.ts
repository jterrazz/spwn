import type { PicksProps } from './picks';
import { LONG_NAME, nothing } from './story';
import type { Story } from './story';

export const PICKS_STORIES: readonly Story<PicksProps<string>>[] = [
    {
        name: 'cards, one chosen, one held',
        props: {
            label: 'On',
            look: 'cards',
            onChange: nothing,
            options: [
                { id: 'this', lines: ['4 GB free'], title: 'This Mac' },
                { id: 'mini', lines: ['36 GB free'], title: 'jterrazz-mac-mini' },
                {
                    disabled: true,
                    id: 'air',
                    lines: ['macbook-air is asleep'],
                    title: 'macbook-air',
                },
            ],
            value: 'mini',
        },
    },
    {
        name: 'cards of two lines',
        props: {
            label: 'Lifetime',
            look: 'cards',
            onChange: nothing,
            options: [
                { id: 'ephemeral', lines: ['Deleted when it powers off'], title: 'ephemeral' },
                { id: 'persistent', lines: ['Stays until you delete it'], title: 'persistent' },
            ],
            value: 'ephemeral',
        },
    },
    {
        name: 'chips, one with no room',
        props: {
            label: 'Memory presets',
            look: 'chips',
            onChange: nothing,
            options: [
                { id: 'small', title: '4 GB · small' },
                { id: 'workspace', title: '8 GB · workspace' },
                { disabled: true, id: 'gate', title: '12 GB · no room' },
            ],
            value: 'workspace',
        },
    },
    {
        name: 'nothing chosen, a long title',
        props: {
            label: 'On',
            look: 'cards',
            onChange: nothing,
            options: [
                { id: 'long', lines: ['20 GB free'], title: LONG_NAME },
                { id: 'mini', title: 'jterrazz-mac-mini' },
            ],
            value: null,
        },
    },
];

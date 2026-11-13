import type { ChoiceCardProps } from './choice-card';
import { nothing } from './story';
import type { Story } from './story';

export const CHOICE_CARD_STORIES: readonly Story<ChoiceCardProps>[] = [
    {
        name: 'apps',
        props: {
            count: '15 apps',
            description: 'What a machine runs: tools, runtimes and Mac apps',
            foot: 'node, go, docker and 12 more',
            icon: 'apps',
            onChoose: nothing,
            title: 'Apps',
        },
    },
    {
        name: 'settings',
        props: {
            count: '7 settings',
            description: 'How a machine behaves',
            foot: 'the Dock, the battery and 5 more',
            icon: 'settings',
            onChoose: nothing,
            title: 'Settings',
        },
    },
    {
        name: 'empty',
        props: {
            count: '0 skills',
            description: 'What an agent knows how to do',
            foot: '',
            icon: 'skills',
            onChoose: nothing,
            title: 'Skills',
        },
    },
    {
        name: 'long text',
        props: {
            count: '1,204 apps',
            description:
                'Everything station knows how to install on a machine, from the runtimes to the Mac apps and the agent skills',
            foot: 'node, go, docker, kubectl, terraform, mise, uv and 1,197 more',
            icon: 'apps',
            onChoose: nothing,
            title: 'Every app of the catalog',
        },
    },
];

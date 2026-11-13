import { SECRET_SAMPLE } from './story';
import type { Story } from './story';
import type { ValueProps } from './value';

const PATH = '~/.config/station/machinefiles/jterrazz-macbook-pro.yaml';

export const VALUE_STORIES: readonly Story<ValueProps>[] = [
    { name: 'plain', props: { children: '24.11.1' } },
    { name: 'raised', props: { children: '24.11.1', look: 'raised' } },
    { name: 'boxed, a value you set', props: { children: 'vault:claude', look: 'boxed' } },
    {
        name: 'raised, an identity',
        props: { children: 'jterrazz-base', identity: 'teal', look: 'raised' },
    },
    {
        name: 'raised, the system identity',
        props: { children: 'macos', identity: 'system', look: 'raised' },
    },
    { name: 'long text, cut in the middle', props: { children: PATH } },
    { name: 'long text, raised', props: { children: PATH, look: 'raised' } },
    { name: 'withheld', props: { children: SECRET_SAMPLE } },
    { name: 'empty', props: { children: '' } },
];

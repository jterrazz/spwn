import type { MachineChipProps } from './machine-chip';
import { LONG_NAME, nothing } from './story';
import type { Story } from './story';

export const MACHINE_CHIP_STORIES: readonly Story<MachineChipProps>[] = [
    {
        name: 'a workspace',
        props: { identity: 'blue', kind: 'workspace', name: 'gate4', onOpen: nothing },
    },
    {
        name: 'a server',
        props: { identity: 'lilac', kind: 'server', name: 'relay', onOpen: nothing },
    },
    { name: 'a Mac', props: { kind: 'mac', name: 'jterrazz-macbook-pro', onOpen: nothing } },
    {
        name: 'the system identity',
        props: { identity: 'system', kind: 'workspace', name: 'base', onOpen: nothing },
    },
    {
        name: 'long text',
        props: { identity: 'teal', kind: 'workspace', name: LONG_NAME, onOpen: nothing },
    },
];

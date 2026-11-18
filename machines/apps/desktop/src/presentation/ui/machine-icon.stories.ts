import type { ComponentProps } from 'react';

import { IDENTITIES } from './identity';
import type { MachineIcon } from './machine-icon';
import type { Story } from './story';

export const MACHINE_ICON_STORIES: readonly Story<ComponentProps<typeof MachineIcon>>[] = [
    { name: 'a Mac', props: { kind: 'mac' } },
    { name: 'a server', props: { identity: 'lilac', kind: 'server' } },
    ...IDENTITIES.map((identity) => ({
        name: `a workspace, ${identity}`,
        props: { identity, kind: 'workspace' as const },
    })),
    { name: 'a workspace, system', props: { identity: 'system', kind: 'workspace' } },
    { name: 'large', props: { identity: 'sky', kind: 'server', size: 'large' } },
    { name: 'large, a Mac', props: { kind: 'mac', size: 'large' } },
];

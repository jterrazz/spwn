import { SECRET_SAMPLE } from '../../design/story';
import type { Story } from '../../design/story';
import { Openable } from '../../engine/opener';
import type { OpenButtonProps } from './opens';

const SSH = Openable.ofSsh({ Address: 'linux-1', Command: 'ssh admin@linux-1', User: 'admin' });

const FIX = Openable.ofFix({ hint: 'npm login', provider: 'npm', state: 'broken' });

const SECRET_FIX = Openable.ofFix({
    hint: `npm login --token ${SECRET_SAMPLE}`,
    provider: 'npm',
    state: 'broken',
});

export const OPEN_BUTTON_STORIES: readonly Story<OpenButtonProps>[] = [
    {
        name: 'enabled: the value as its title',
        props: {
            held: 'The engine gives no command to reach linux-1',
            label: 'Open in Terminal',
            openable: SSH,
            variant: 'primary',
        },
    },
    {
        name: 'enabled, plain',
        props: {
            held: 'The engine gives no command that signs in',
            label: 'Sign in',
            openable: FIX,
        },
    },
    {
        name: 'held: no value, its reason as its title',
        props: {
            held: 'Not opened: its address is local to the Mac it is shared on',
            label: 'Open',
            openable: undefined,
            variant: 'link',
        },
    },
    {
        name: 'withheld: a value shaped like a secret',
        props: {
            held: 'The engine gives no command that signs in',
            label: 'Sign in',
            openable: SECRET_FIX,
        },
    },
];

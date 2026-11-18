import type { ConfirmDialogProps } from './confirm-dialog';
import { nothing, SECRET_SAMPLE } from './story';
import type { Story } from './story';

const SHARED = { onCancel: nothing, onConfirm: nothing };

export const CONFIRM_DIALOG_STORIES: readonly Story<ConfirmDialogProps>[] = [
    { name: 'a yes', props: { ...SHARED, title: 'Stop linux-1?', verb: 'Stop' } },
    {
        name: 'a yes that destroys',
        props: { ...SHARED, destroys: true, title: 'Erase the caches?', verb: 'Erase' },
    },
    {
        name: 'the name typed, disabled until it matches',
        props: {
            ...SHARED,
            said: 'deleting a server needs its name',
            title: 'Delete infra?',
            typed: 'infra',
            verb: 'Delete infra',
        },
    },
    {
        name: 'a delete nothing holds, its consequence alone',
        props: {
            ...SHARED,
            said: 'api and everything on its disk are deleted for good.',
            title: 'Delete api?',
            typed: 'api',
            verb: 'Delete',
        },
    },
    {
        name: "a delete of a held machine, the engine's hold on its own warning line",
        props: {
            ...SHARED,
            held: 'It is held by its lot, by studio-os-fe: work may be in progress.',
            said: 'linux-1 and everything on its disk are deleted for good.',
            title: 'Delete linux-1?',
            typed: 'linux-1',
            verb: 'Delete',
        },
    },
    {
        name: 'withheld consequence',
        props: { ...SHARED, said: SECRET_SAMPLE, title: 'Revoke the token?', verb: 'Revoke' },
    },
];

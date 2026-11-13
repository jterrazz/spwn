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
        name: 'withheld consequence',
        props: { ...SHARED, said: SECRET_SAMPLE, title: 'Revoke the token?', verb: 'Revoke' },
    },
];

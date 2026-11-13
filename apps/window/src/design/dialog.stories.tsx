import { Button } from './button';
import type { DialogProps, ModalProps } from './dialog';
import { FormField } from './form-field';
import { nothing } from './story';
import type { Story } from './story';

import styles from './story.module.css';

const ACTIONS = (
    <>
        <Button>Keep gate3</Button>
        <Button variant="primary">Delete gate3</Button>
    </>
);

export const DIALOG_STORIES: readonly Story<DialogProps>[] = [
    {
        name: 'a field to fill',
        props: {
            actions: ACTIONS,
            children: (
                <FormField
                    label="Type its name to confirm"
                    onChange={nothing}
                    placeholder="gate3"
                    value=""
                />
            ),
            onClose: nothing,
            title: 'Delete gate3?',
        },
    },
    {
        name: 'a consequence under its title',
        props: {
            actions: ACTIONS,
            children: null,
            consequence: 'Its disk and everything on it go with it.',
            onClose: nothing,
            title: 'Delete gate3?',
        },
    },
    {
        name: 'long text',
        props: {
            actions: ACTIONS,
            children: null,
            consequence:
                'Every app, every setting and every file the machine holds goes with it, and nothing of it can be brought back afterwards.',
            onClose: nothing,
            title: 'Delete a-machine-name-long-enough-to-wrap-the-title?',
        },
    },
];

export const MODAL_STORIES: readonly Story<ModalProps>[] = [
    {
        name: 'labelled, any content',
        props: {
            children: <p className={styles.modal}>A modal holds what a dialog does not draw.</p>,
            label: 'A modal',
            onClose: nothing,
        },
    },
];

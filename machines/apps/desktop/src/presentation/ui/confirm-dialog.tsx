import { useState } from 'react';

import { Button, Screen } from './button';
import { asSentence } from './case';
import { Dialog } from './dialog';
import { FormField } from './form-field';
import { shownOrWithheld } from './secret';
import { StateSign } from './state-sign';

import styles from './confirm-dialog.module.css';

export type ConfirmDialogProps = {
    title: string;
    verb: string;
    typed?: string | undefined;
    said?: string | undefined;
    held?: string | undefined;
    destroys?: boolean;
    onCancel: () => void;
    onConfirm: (typed: string) => void;
};

function Held({ held }: { held: string | undefined }) {
    if ((held ?? '') === '') {
        return null;
    }
    return (
        <p className={styles.held}>
            <StateSign tone="warn" />
            <span>{shownOrWithheld(held ?? '')}</span>
        </p>
    );
}

export function ConfirmDialog({
    title,
    verb,
    typed,
    said,
    held,
    destroys = false,
    onCancel,
    onConfirm,
}: ConfirmDialogProps) {
    const [text, setText] = useState('');
    const ready = typed === undefined || text.trim() === typed;
    const confirm = () => {
        onConfirm(typed === undefined ? '' : text.trim());
    };
    return (
        <Screen>
            <Dialog
                actions={
                    <>
                        <Button onClick={onCancel}>Cancel</Button>
                        <Button
                            disabled={!ready}
                            onClick={confirm}
                            variant={destroys || typed !== undefined ? 'destroy' : 'primary'}
                        >
                            {verb}
                        </Button>
                    </>
                }
                consequence={
                    (said ?? '') === '' ? undefined : asSentence(shownOrWithheld(said ?? ''))
                }
                onClose={onCancel}
                opensOn="dialog"
                title={title}
            >
                <Held held={held} />
                {typed === undefined ? null : (
                    <FormField
                        label={`Type ${typed} to confirm`}
                        labelInside
                        onChange={setText}
                        onEnter={ready ? confirm : undefined}
                        value={text}
                    />
                )}
            </Dialog>
        </Screen>
    );
}

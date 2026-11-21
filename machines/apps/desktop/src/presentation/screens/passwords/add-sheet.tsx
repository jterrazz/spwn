import type { AccountsAddFrom, AccountsVaultOffer } from '@jterrazz/station-api';
import { useState } from 'react';

import type { EngineError } from '../../../domain/engine-error';
import { useWatch } from '../../dependencies/dependencies';
import { Unanswered } from '../../kit/readings';
import { Skeleton, SkeletonRows } from '../../kit/skeleton';
import { Button, Dialog } from '../../ui/index';
import { codesOf } from '../../words/engine-words';
import { BLANK, chosenOf, Form, wayOf } from './add-form';
import type { Draft, Place } from './add-form';
import { useAdd } from './use-add';
import type { Asked } from './use-add';

const TITLE = 'Add a password';

// invariant: a refusal is drawn on the input its `field` names, and in the sheet's foot when it names none.
function placeOf(error: EngineError | undefined): Place | undefined {
    if (error === undefined) {
        return undefined;
    }
    return error.wire?.field ?? 'foot';
}

type ActionsProps = {
    asked: Asked | undefined;
    sending: boolean;
    send: (asked: Asked) => Promise<void>;
    close: () => void;
};

function Actions({ asked, sending, send, close }: ActionsProps) {
    return (
        <>
            <Button onClick={close}>Cancel</Button>
            <Button
                busy={sending}
                disabled={asked === undefined}
                onClick={() => {
                    if (asked !== undefined) {
                        void send(asked);
                    }
                }}
                variant="primary"
            >
                Add password
            </Button>
        </>
    );
}

type LoadedProps = { offer: AccountsVaultOffer; first: AccountsAddFrom; onClose: () => void };

function Loaded({ offer, first, onClose }: LoadedProps) {
    const [draft, setDraft] = useState<Draft>(BLANK);
    const { forget, refused, secret, send, sending } = useAdd(onClose);
    const chosen = chosenOf(offer, draft);
    const from = draft.from ?? first;
    const { asked, proposal } = wayOf(from, chosen);
    const label = draft.labelled ? draft.label : proposal;
    const place = placeOf(refused);
    const close = () => {
        forget();
        onClose();
    };
    const refusal = (at: Place) =>
        place === at && refused !== undefined ? <Unanswered error={refused} size="panel" /> : null;
    const actions = (
        <Actions
            asked={asked === undefined ? undefined : { ...asked, Label: label }}
            close={close}
            send={send}
            sending={sending}
        />
    );
    return (
        <Dialog
            actions={actions}
            consequence={offer.Intro}
            onClose={close}
            opensOn="dialog"
            title={TITLE}
        >
            <Form
                change={(next) => {
                    if (next.from !== undefined) {
                        forget();
                    }
                    setDraft({ ...draft, ...next });
                }}
                chosen={chosen}
                from={from}
                label={label}
                offer={offer}
                refusal={refusal}
                secret={secret}
            />
        </Dialog>
    );
}

// invariant: the vault is This Mac's alone, so the sheet reads the socket's own engine and names no target.
export function AddPasswordSheet({ onClose }: { onClose: () => void }) {
    const offered = useWatch('vault.new', {}, undefined);
    const first = codesOf('AccountsAddFrom')[0];
    const cancel = <Button onClick={onClose}>Cancel</Button>;
    if (offered.kind === 'failed') {
        return (
            <Dialog actions={cancel} onClose={onClose} title={TITLE}>
                <Unanswered error={offered.error} size="panel" />
            </Dialog>
        );
    }
    if (offered.kind === 'reading' || first === undefined) {
        return (
            <Dialog actions={cancel} onClose={onClose} title={TITLE}>
                <Skeleton what="the programs a password is for">
                    <SkeletonRows lead="icon" />
                </Skeleton>
            </Dialog>
        );
    }
    return <Loaded first={first} offer={offered.out} onClose={onClose} />;
}

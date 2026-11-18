import type { AccountsGrantOffer, AccountsTarget, AccountsVaultEntry } from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactNode } from 'react';

import { useWatch } from '../../dependencies/dependencies';
import { useGesture } from '../../dependencies/use-gesture';
import { GestureButton, GestureOutcome } from '../../kit/gestures';
import { Reading, Unanswered } from '../../kit/readings';
import { Button, Dialog, RadioRows } from '../../ui/index';
import { targetRowsOf, VAULT_REREADS } from './passwords-of';

import styles from './passwords.module.css';

type GrantProps = {
    entry: AccountsVaultEntry;
    target: AccountsTarget;
    onDone: () => void;
    cancel: ReactNode;
};

function Grant({ entry, target, onDone, cancel }: GrantProps) {
    const grant = useGesture('grants.add', {
        input: { Entry: entry.ID, To: target.To },
        onDone,
        rereads: VAULT_REREADS,
        subject: target.To,
    });
    const label = `Grant to ${target.To}`;
    return (
        <div className={styles.foot}>
            <GestureOutcome gesture={grant} />
            <div className={styles.buttons}>
                {cancel}
                <GestureButton
                    gesture={grant}
                    label={label}
                    title={`${label}?`}
                    variant="primary"
                />
            </div>
        </div>
    );
}

type OfferProps = {
    offer: AccountsGrantOffer;
    chosen: null | string;
    onChoose: (id: string) => void;
};

function Offer({ offer, chosen, onChoose }: OfferProps) {
    return (
        <div className={styles.picks}>
            <RadioRows
                label="Machines"
                onChange={onChoose}
                rows={targetRowsOf(offer.Targets ?? [])}
                value={chosen}
            />
            {offer.Presence === '' ? null : <p className={styles.presence}>{offer.Presence}</p>}
        </div>
    );
}

export function GrantSheet({ entry, onClose }: { entry: AccountsVaultEntry; onClose: () => void }) {
    const offered = useWatch('grants.new', { Entry: entry.ID }, undefined);
    const [chosen, setChosen] = useState<null | string>(null);
    const cancel = <Button onClick={onClose}>Cancel</Button>;
    const title = `Grant ${entry.Label}`;
    if (offered.kind === 'failed') {
        return (
            <Dialog actions={cancel} onClose={onClose} title={title}>
                <Unanswered error={offered.error} size="panel" />
            </Dialog>
        );
    }
    if (offered.kind === 'reading') {
        return (
            <Dialog actions={cancel} onClose={onClose} title={title}>
                <Reading size="tab" what={`where ${entry.Label} can go`} />
            </Dialog>
        );
    }
    const target = (offered.out.Targets ?? []).find((one) => one.To === chosen);
    const actions =
        target === undefined ? (
            <>
                {cancel}
                <Button disabled variant="primary">
                    Grant
                </Button>
            </>
        ) : (
            <Grant cancel={cancel} entry={entry} key={chosen} onDone={onClose} target={target} />
        );
    return (
        <Dialog actions={actions} onClose={onClose} title={title}>
            <Offer chosen={chosen} offer={offered.out} onChoose={setChosen} />
        </Dialog>
    );
}

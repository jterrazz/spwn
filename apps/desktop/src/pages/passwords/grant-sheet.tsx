import type {
    AccountsGrantOffer,
    AccountsTarget,
    AccountsVaultEntry,
    GrantsAddInput,
} from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactNode } from 'react';

import { Button, Dialog, Picks } from '../../design/index';
import { wordOf } from '../../engine/engine-words';
import { useWatch } from '../../engine/react';
import { useGesture } from '../../engine/use-gesture';
import { GestureButton, GestureOutcome } from '../kit/gestures';
import { Reading, Unanswered } from '../kit/readings';
import { targetId, targetLines, VAULT_REREADS } from './passwords-of';

import styles from './passwords.module.css';

function canPick({ Fit }: Pick<AccountsTarget, 'Fit'>): boolean {
    switch (Fit) {
        case 'fits': {
            return true;
        }
        case 'no': {
            return false;
        }
    }
}

function grantOf(entry: AccountsVaultEntry, target: AccountsTarget): GrantsAddInput {
    const workspace = target.Workspace ?? '';
    return {
        Entry: entry.ID,
        To: target.To,
        ToKind: target.ToKind,
        ...(workspace === '' ? {} : { Workspace: workspace }),
    };
}

type GrantProps = {
    entry: AccountsVaultEntry;
    target: AccountsTarget;
    onDone: () => void;
    cancel: ReactNode;
};

function Grant({ entry, target, onDone, cancel }: GrantProps) {
    const grant = useGesture('grants.add', {
        input: grantOf(entry, target),
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

type Group = { to: AccountsTarget['ToKind']; targets: readonly AccountsTarget[] };

// invariant: the groups keep the order the engine lists their first target in, and each its targets' order.
function groupsOf(offer: AccountsGrantOffer): readonly Group[] {
    const targets = offer.Targets ?? [];
    const kinds = [...new Set(targets.map((target) => target.ToKind))];
    return kinds.map((to) => ({ targets: targets.filter((target) => target.ToKind === to), to }));
}

type OfferProps = {
    offer: AccountsGrantOffer;
    chosen: null | string;
    onChoose: (id: string) => void;
};

function Offer({ offer, chosen, onChoose }: OfferProps) {
    return (
        <div className={styles.picks}>
            {groupsOf(offer).map(({ to, targets }) => {
                const label = wordOf('AccountsGrantTo', to).Word;
                return (
                    <div key={to}>
                        <p className={styles.group}>{label}</p>
                        <Picks
                            label={label}
                            look="cards"
                            onChange={onChoose}
                            options={targets.map((target) => ({
                                disabled: !canPick(target),
                                id: targetId(target),
                                lines: targetLines(target),
                                title: target.To,
                            }))}
                            value={chosen}
                        />
                    </div>
                );
            })}
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
    const target = (offered.out.Targets ?? []).find((one) => targetId(one) === chosen);
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

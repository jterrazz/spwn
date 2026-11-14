import type {
    AccountsAddFrom,
    AccountsKeyKind,
    AccountsVaultProgram,
    AccountsVaultRead,
} from '@jterrazz/station-api';
import { useId, useState } from 'react';
import type { ReactNode, RefCallback } from 'react';

import { Button, capitalised, Dialog, Field, FormField, Picks } from '../../design/index';
import type { EngineError } from '../../engine/engine-error';
import { codesOf, wordOf } from '../../engine/engine-words';
import { useWatch } from '../../engine/react';
import { Reading, Unanswered } from '../kit/readings';
import { useAdd } from './use-add';
import type { Asked } from './use-add';

import styles from './passwords.module.css';

const TITLE = 'Add a password';

type Program = AccountsVaultProgram['Program'];
type Place = 'foot' | 'label' | 'value';

// invariant: a refusal is drawn on the input its `field` names, and in the sheet's foot when it names none.
function placeOf(error: EngineError | undefined): Place | undefined {
    if (error === undefined) {
        return undefined;
    }
    return error.wire?.field ?? 'foot';
}

// invariant: a program the engine cannot take from its sign-in offers the typed way alone.
function waysOf(program: AccountsVaultProgram | undefined): readonly AccountsAddFrom[] {
    return program?.Captures === true ? codesOf('AccountsAddFrom') : ['typed'];
}

function wayPick(from: AccountsAddFrom) {
    const { Word, Detail } = wordOf('AccountsAddFrom', from);
    return { id: from, lines: Detail === undefined ? [] : [Detail], title: capitalised(Word) };
}

type Draft = {
    program: Program | undefined;
    from: AccountsAddFrom | undefined;
    kind: AccountsKeyKind | undefined;
    label: string;
};

// invariant: the way picked while its program offers it, else the first way the program offers.
function fromOf(draft: Draft, ways: readonly AccountsAddFrom[]): AccountsAddFrom | undefined {
    return draft.from !== undefined && ways.includes(draft.from) ? draft.from : ways[0];
}

function askedOf(draft: Draft, ways: readonly AccountsAddFrom[]): Asked | undefined {
    const { program, kind, label } = draft;
    const from = fromOf(draft, ways);
    if (program === undefined || kind === undefined || from === undefined) {
        return undefined;
    }
    return { From: from, Kind: kind, Label: label, Program: program };
}

function SecretInput({
    secret,
    refusal,
}: {
    secret: RefCallback<HTMLInputElement>;
    refusal: ReactNode;
}) {
    const id = useId();
    return (
        <Field after={refusal} label="Value" labelFor={id}>
            <input
                autoComplete="off"
                className={styles.secret}
                id={id}
                ref={secret}
                spellCheck={false}
                type="password"
            />
        </Field>
    );
}

type FormProps = {
    read: AccountsVaultRead;
    draft: Draft;
    change: (next: Partial<Draft>) => void;
    secret: RefCallback<HTMLInputElement>;
    refusal: (at: Place) => ReactNode;
};

function Form({ read, draft, change, secret, refusal }: FormProps) {
    const programs = read.Programs ?? [];
    const ways = waysOf(programs.find((one) => one.Program === draft.program));
    const from = fromOf(draft, ways);
    return (
        <div className={styles.picks}>
            <Field label="Program">
                <Picks
                    label="Program"
                    look="chips"
                    onChange={(program) => {
                        change({ from: undefined, program });
                    }}
                    options={programs.map((one) => ({ id: one.Program, title: one.Label }))}
                    value={draft.program ?? null}
                />
            </Field>
            <Field label="From">
                <Picks
                    label="From"
                    look="cards"
                    onChange={(way) => {
                        change({ from: way });
                    }}
                    options={ways.map((way) => wayPick(way))}
                    value={from ?? null}
                />
            </Field>
            <Field label="Kind">
                <Picks
                    label="Kind"
                    look="chips"
                    onChange={(kind) => {
                        change({ kind });
                    }}
                    options={codesOf('AccountsKeyKind').map((kind) => ({
                        id: kind,
                        title: wordOf('AccountsKeyKind', kind).Word,
                    }))}
                    value={draft.kind ?? null}
                />
            </Field>
            {from === 'typed' ? <SecretInput refusal={refusal('value')} secret={secret} /> : null}
            <FormField
                label="Label"
                look="boxed"
                onChange={(label) => {
                    change({ label });
                }}
                value={draft.label}
            />
            {refusal('label')}
            {refusal('foot')}
        </div>
    );
}

function Loaded({ read, onClose }: { read: AccountsVaultRead; onClose: () => void }) {
    const first = read.Programs?.[0]?.Program;
    const [draft, setDraft] = useState<Draft>({
        from: undefined,
        kind: undefined,
        label: '',
        program: first,
    });
    const { forget, refused, secret, send, sending } = useAdd(onClose);
    const ways = waysOf((read.Programs ?? []).find((one) => one.Program === draft.program));
    const asked = askedOf(draft, ways);
    const place = placeOf(refused);
    const close = () => {
        forget();
        onClose();
    };
    const refusal = (at: Place) =>
        place === at && refused !== undefined ? <Unanswered error={refused} size="panel" /> : null;
    const actions = (
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
    return (
        <Dialog actions={actions} onClose={close} opensOn="dialog" title={TITLE}>
            <Form
                change={(next) => {
                    setDraft({ ...draft, ...next });
                }}
                draft={draft}
                read={read}
                refusal={refusal}
                secret={secret}
            />
        </Dialog>
    );
}

// invariant: the vault is This Mac's alone, so the sheet reads the socket's own engine and names no target.
export function AddPasswordSheet({ onClose }: { onClose: () => void }) {
    const listed = useWatch('vault.list', {}, undefined);
    const cancel = <Button onClick={onClose}>Cancel</Button>;
    if (listed.kind === 'failed') {
        return (
            <Dialog actions={cancel} onClose={onClose} title={TITLE}>
                <Unanswered error={listed.error} size="panel" />
            </Dialog>
        );
    }
    if (listed.kind === 'reading') {
        return (
            <Dialog actions={cancel} onClose={onClose} title={TITLE}>
                <Reading size="tab" what="the programs a password is for" />
            </Dialog>
        );
    }
    return <Loaded onClose={onClose} read={listed.out} />;
}

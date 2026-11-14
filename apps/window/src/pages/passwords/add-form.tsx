import type {
    AccountsAddFrom,
    AccountsCapture,
    AccountsKeyKind,
    AccountsTypedProgram,
    AccountsVaultOffer,
} from '@jterrazz/station-api';
import { useId } from 'react';
import type { ReactNode, RefCallback } from 'react';

import { AppIcon, capitalised, Field, FormField, Picks } from '../../design/index';
import { codesOf, wordOf } from '../../engine/engine-words';
import { sentence } from '../../model/format';
import type { Asked } from './use-add';

import styles from './passwords.module.css';

type Program = AccountsTypedProgram['Program'];

export type Place = 'foot' | 'label' | 'value';

export type Draft = {
    from: AccountsAddFrom | undefined;
    capture: Program | undefined;
    program: Program | undefined;
    kind: AccountsKeyKind | undefined;
    label: string;
    labelled: boolean;
};

export const BLANK: Draft = {
    capture: undefined,
    from: undefined,
    kind: undefined,
    label: '',
    labelled: false,
    program: undefined,
};

export type Chosen = {
    capture: AccountsCapture | undefined;
    program: AccountsTypedProgram | undefined;
    kind: AccountsKeyKind | undefined;
};

export function chosenOf(offer: AccountsVaultOffer, draft: Draft): Chosen {
    const captures = offer.Captures ?? [];
    const programs = offer.Programs ?? [];
    const program = programs.find((one) => one.Program === draft.program) ?? programs[0];
    const kinds = program?.Kinds ?? [];
    return {
        capture: captures.find((one) => one.Program === draft.capture) ?? captures[0],
        kind: draft.kind !== undefined && kinds.includes(draft.kind) ? draft.kind : program?.Kind,
        program,
    };
}

export type Way = { asked: Omit<Asked, 'Label'> | undefined; proposal: string };

// invariant: the request of a way carries what that way draws: a sign-in's program alone, a typed value's program and kind.
export function wayOf(from: AccountsAddFrom, { capture, program, kind }: Chosen): Way {
    switch (from) {
        case 'sign-in': {
            return {
                asked: capture === undefined ? undefined : { From: from, Program: capture.Program },
                proposal: capture?.Label ?? '',
            };
        }
        case 'typed': {
            return {
                asked:
                    program === undefined || kind === undefined
                        ? undefined
                        : { From: from, Kind: kind, Program: program.Program },
                proposal: '',
            };
        }
    }
}

function wayPick(from: AccountsAddFrom) {
    const { Word, Detail } = wordOf('AccountsAddFrom', from);
    return { id: from, lines: Detail === undefined ? [] : [Detail], title: capitalised(Word) };
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

export type FormProps = {
    offer: AccountsVaultOffer;
    chosen: Chosen;
    from: AccountsAddFrom;
    label: string;
    change: (next: Partial<Draft>) => void;
    secret: RefCallback<HTMLInputElement>;
    refusal: (at: Place) => ReactNode;
};

function Captures({ offer, chosen, change }: Pick<FormProps, 'change' | 'chosen' | 'offer'>) {
    const captures = offer.Captures ?? [];
    if (captures.length === 0) {
        return (
            <Field label="Sign-in">
                <p className={styles.sentence}>{offer.CaptureLine}</p>
            </Field>
        );
    }
    return (
        <Field label="Sign-in">
            <Picks
                label="Sign-in"
                look="cards"
                onChange={(capture) => {
                    change({ capture });
                }}
                options={captures.map((one) => ({
                    id: one.Program,
                    lead: <AppIcon name={one.Program} />,
                    lines: [sentence([one.Who, (one.Scopes ?? []).join(', ')])].filter(
                        (line) => line !== '',
                    ),
                    title: one.Program,
                }))}
                value={chosen.capture?.Program ?? null}
            />
            {chosen.capture === undefined ? null : (
                <p className={styles.sentence}>{chosen.capture.KindSentence}</p>
            )}
        </Field>
    );
}

function Typed({
    offer,
    chosen,
    change,
    secret,
    refusal,
}: Pick<FormProps, 'change' | 'chosen' | 'offer' | 'refusal' | 'secret'>) {
    const kinds = chosen.program?.Kinds ?? [];
    return (
        <>
            <Field label="Program">
                <Picks
                    label="Program"
                    look="chips"
                    onChange={(program) => {
                        change({ kind: undefined, program });
                    }}
                    options={(offer.Programs ?? []).map((one) => ({
                        id: one.Program,
                        title: one.Label,
                    }))}
                    value={chosen.program?.Program ?? null}
                />
            </Field>
            {kinds.length > 1 ? (
                <Field label="Kind">
                    <Picks
                        label="Kind"
                        look="chips"
                        onChange={(kind) => {
                            change({ kind });
                        }}
                        options={kinds.map((kind) => ({
                            id: kind,
                            title: wordOf('AccountsKeyKind', kind).Word,
                        }))}
                        value={chosen.kind ?? null}
                    />
                </Field>
            ) : null}
            <SecretInput refusal={refusal('value')} secret={secret} />
        </>
    );
}

function Body(props: FormProps) {
    switch (props.from) {
        case 'sign-in': {
            return <Captures {...props} />;
        }
        case 'typed': {
            return <Typed {...props} />;
        }
    }
}

export function Form(props: FormProps) {
    const { from, label, change, refusal } = props;
    return (
        <div className={styles.picks}>
            <Field label="From">
                <Picks
                    label="From"
                    look="cards"
                    onChange={(way) => {
                        change({ from: way });
                    }}
                    options={codesOf('AccountsAddFrom').map((way) => wayPick(way))}
                    value={from}
                />
            </Field>
            <Body {...props} />
            <FormField
                label="Label"
                look="boxed"
                onChange={(next) => {
                    change({ label: next, labelled: true });
                }}
                value={label}
            />
            {refusal('label')}
            {refusal('foot')}
        </div>
    );
}

import type { AccountsProgramRow, CallName } from '@jterrazz/station-api';
import { useState } from 'react';

import { AppIcon, asSentence, Button, Panel, Verdict } from '../../design/index';
import { wordOf } from '../../engine/engine-words';
import { useWatch } from '../../engine/react';
import { engineShown, grantEndOf, ownState, verdictShown } from '../../engine/shown';
import { useGesture } from '../../engine/use-gesture';
import type { Gesture } from '../../engine/use-gesture';
import type { Found } from '../../model/machines';
import { GestureButton, GestureRow, HeldOutcomes } from '../kit/gestures';
import { Reading, Unanswered } from '../kit/readings';
import { Page } from '../page';
import { aimed } from './aim';
import { LoginSheet } from './login-sheet';
import { secondLine } from './sign-ins-of';

import styles from './sign-ins.module.css';

const LIST_REREADS: readonly CallName[] = ['accounts.list'];

type Asked = { program: AccountsProgramRow['Program']; action: AccountsProgramRow['Action'] };

type RowProps = { row: AccountsProgramRow; found: Found; onSignIn: (asked: Asked) => void };

function actionWord(row: Pick<AccountsProgramRow, 'Action'>): string {
    return wordOf('AccountsProgramAction', row.Action).Word;
}

function rowShown(row: AccountsProgramRow, withLine: boolean) {
    return {
        lead: <AppIcon name={row.Program} />,
        sentence: secondLine(row, withLine),
        state: { ...engineShown('AccountsProgramState', row.State), calm: 'pill' as const },
        title: row.Program,
        trailSize: 'wide' as const,
        value: grantEndOf(row.Until),
    };
}

function offersNothing({ Action }: Pick<AccountsProgramRow, 'Action'>): boolean {
    switch (Action) {
        case 'none': {
            return true;
        }
        case 'remove':
        case 'sign-in':
        case 'sign-in-again':
        case 'sign-out': {
            return false;
        }
    }
}

function Trail({
    row,
    signOut,
    onSignIn,
}: Omit<RowProps, 'found'> & { signOut: Gesture<unknown> }) {
    const word = actionWord(row);
    const name = `${word} ${row.Program}`;
    switch (row.Action) {
        case 'sign-out': {
            return (
                <GestureButton
                    gesture={signOut}
                    label={word}
                    name={name}
                    title={row.Ask ?? `${name}?`}
                    verb={word}
                />
            );
        }
        case 'sign-in':
        case 'sign-in-again': {
            return (
                <Button
                    name={name}
                    onClick={() => {
                        onSignIn({ action: row.Action, program: row.Program });
                    }}
                >
                    {word}
                </Button>
            );
        }
        case 'remove': {
            return (
                <Button disabled name={name} title={ownState('not-wired').word}>
                    {word}
                </Button>
            );
        }
        case 'none': {
            return <span className={styles.said}>{row.Line ?? ''}</span>;
        }
    }
}

// invariant: a row keeps one sign-out gesture whatever its action, so the mark of a sign-out stays on the row the
// invariant: engine answers with, though that row now offers a sign-in.
function ProgramRow({ row, found, onSignIn }: RowProps) {
    const signOut = useGesture('accounts.logout', {
        ...aimed(found, { Program: row.Program }),
        rereads: LIST_REREADS,
        subject: row.Program,
    });
    return (
        <GestureRow
            {...rowShown(row, !offersNothing(row))}
            gesture={signOut}
            trail={<Trail onSignIn={onSignIn} row={row} signOut={signOut} />}
        />
    );
}

export function SignInsTab({ found }: { found: Found }) {
    const name = found.vm?.Name ?? found.mac.Name;
    const { input, target } = aimed(found, {});
    const listed = useWatch('accounts.list', input, target);
    const [asked, setAsked] = useState<Asked | undefined>(undefined);
    if (listed.kind === 'failed') {
        return (
            <Page>
                <Unanswered error={listed.error} size="tab" />
            </Page>
        );
    }
    if (listed.kind === 'reading') {
        return (
            <Page>
                <Reading size="tab" what={`the sign-ins of ${name}`} />
            </Page>
        );
    }
    const { Verdict: verdict, Programs } = listed.out;
    const programs = Programs ?? [];
    return (
        <Page>
            <Verdict
                line=""
                size="tab"
                title={asSentence(verdict.Sentence)}
                {...verdictShown(verdict)}
            />
            <HeldOutcomes>
                <Panel note={String(programs.length)} title="Sign-ins">
                    {programs.map((row) => (
                        <ProgramRow found={found} key={row.Program} onSignIn={setAsked} row={row} />
                    ))}
                </Panel>
            </HeldOutcomes>
            {asked === undefined ? null : (
                <LoginSheet
                    action={asked.action}
                    found={found}
                    onClose={() => {
                        setAsked(undefined);
                    }}
                    program={asked.program}
                />
            )}
        </Page>
    );
}

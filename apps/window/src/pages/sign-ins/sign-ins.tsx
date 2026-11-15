import type { AccountsProgramRow, CallName } from '@jterrazz/station-api';
import { useState } from 'react';

import { AppIcon, asSentence, Button, Panel, Verdict } from '../../design/index';
import { countOf, wordOf } from '../../engine/engine-words';
import { useWatch } from '../../engine/react';
import { engineShown, grantedCount, verdictShown } from '../../engine/shown';
import { useGesture } from '../../engine/use-gesture';
import type { Gesture } from '../../engine/use-gesture';
import type { Found } from '../../model/machines';
import type { Route } from '../../shell/route';
import { GestureButton, GestureRow, HeldOutcomes } from '../kit/gestures';
import { Reading, Unanswered } from '../kit/readings';
import { Page } from '../page';
import { aimed } from './aim';
import { LoginSheet } from './login-sheet';
import { secondLine } from './sign-ins-of';

import styles from './sign-ins.module.css';

const LIST_REREADS: readonly CallName[] = ['accounts.list'];
const GRANT_REREADS: readonly CallName[] = ['accounts.list', 'vault.list', 'grants.new'];

type Go = (to: Route) => void;

type Asked = { program: AccountsProgramRow['Program']; action: AccountsProgramRow['Action'] };

type RowProps = {
    row: AccountsProgramRow;
    found: Found;
    go: Go;
    onSignIn: (asked: Asked) => void;
};

function actionWord(row: Pick<AccountsProgramRow, 'Action'>): string {
    return wordOf('AccountsProgramAction', row.Action).Word;
}

export function rowShown(row: AccountsProgramRow, withLine: boolean) {
    return {
        lead: <AppIcon name={row.Program} />,
        sentence: secondLine(row, withLine),
        state: { ...engineShown('AccountsProgramState', row.State), calm: 'pill' as const },
        title: row.Program,
        trailSize: 'wide' as const,
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

type TrailProps = Omit<RowProps, 'found' | 'go'> & {
    signOut: Gesture<unknown>;
    removeGrant: Gesture<unknown>;
    onEntry: (() => void) | undefined;
};

function RemoveGrant({ row, removeGrant, onEntry }: Omit<TrailProps, 'onSignIn' | 'signOut'>) {
    const word = actionWord(row);
    const label = row.EntryLabel ?? row.Entry ?? '';
    return (
        <>
            {onEntry === undefined ? null : (
                <Button name={`Open the password ${label}`} onClick={onEntry} variant="link">
                    {label}
                </Button>
            )}
            <GestureButton
                consequence={row.Ask === undefined ? undefined : ''}
                gesture={removeGrant}
                label={word}
                name={`${word} ${row.Program}`}
                title={row.Ask ?? `${word} ${label} from this machine?`}
                verb={word}
            />
        </>
    );
}

function Trail({ row, signOut, removeGrant, onSignIn, onEntry }: TrailProps) {
    const word = actionWord(row);
    const name = `${word} ${row.Program}`;
    switch (row.Action) {
        case 'sign-out': {
            return (
                <GestureButton
                    consequence={row.Ask === undefined ? undefined : ''}
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
            return <RemoveGrant onEntry={onEntry} removeGrant={removeGrant} row={row} />;
        }
        case 'none': {
            return <span className={styles.said}>{row.Line ?? ''}</span>;
        }
    }
}

function shownGesture(
    row: Pick<AccountsProgramRow, 'Action'>,
    gestures: { signOut: Gesture<unknown>; removeGrant: Gesture<unknown> },
): Gesture<unknown> {
    switch (row.Action) {
        case 'remove': {
            return gestures.removeGrant;
        }
        case 'none':
        case 'sign-in':
        case 'sign-in-again':
        case 'sign-out': {
            return gestures.signOut;
        }
    }
}

// invariant: the vault and its grants are This Mac's, so a granted row opens its password only on This Mac and its VMs.
function entryOpener(row: AccountsProgramRow, found: Found, go: Go): (() => void) | undefined {
    const entry = row.Entry ?? '';
    if (!found.mac.ThisMac || entry === '') {
        return undefined;
    }
    return () => {
        go({ item: entry, page: 'passwords' });
    };
}

// invariant: a row keeps one sign-out gesture and one grant removal whatever its action, so the mark of either stays
// invariant: on the row the engine answers with, though that row now offers another gesture.
function ProgramRow({ row, found, go, onSignIn }: RowProps) {
    const signOut = useGesture('accounts.logout', {
        ...aimed(found, { Program: row.Program }),
        rereads: LIST_REREADS,
        subject: row.Program,
    });
    const removeGrant = useGesture('grants.remove', {
        input: { Entry: row.Entry ?? '', To: found.vm?.Name ?? found.mac.Name, ToKind: 'machine' },
        rereads: GRANT_REREADS,
        subject: row.Entry ?? row.Program,
    });
    return (
        <GestureRow
            {...rowShown(row, !offersNothing(row))}
            gesture={shownGesture(row, { removeGrant, signOut })}
            trail={
                <Trail
                    onEntry={entryOpener(row, found, go)}
                    onSignIn={onSignIn}
                    removeGrant={removeGrant}
                    row={row}
                    signOut={signOut}
                />
            }
        />
    );
}

// value: the count of rows the wire says are granted, joined with fixed words; the vault is This Mac's, so a far
// value: Mac's machine offers no Passwords page to open.
function GrantedFoot({
    found,
    go,
    programs,
}: {
    found: Found;
    go: Go;
    programs: readonly AccountsProgramRow[];
}) {
    const count = grantedCount(programs);
    if (count === 0 || !found.mac.ThisMac) {
        return null;
    }
    const one = count === 1;
    const name = found.vm?.Name ?? found.mac.Name;
    return (
        <p className={styles.foot}>
            {`${countOf('password', count)} from this Mac ${one ? 'is' : 'are'} granted to ${name} · `}
            <Button
                onClick={() => {
                    go({ page: 'passwords' });
                }}
                variant="link"
            >
                Passwords
            </Button>
        </p>
    );
}

export function SignInsTab({ found, go }: { found: Found; go: Go }) {
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
                        <ProgramRow
                            found={found}
                            go={go}
                            key={row.Program}
                            onSignIn={setAsked}
                            row={row}
                        />
                    ))}
                    <GrantedFoot found={found} go={go} programs={programs} />
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

import type {
    AccountsTarget,
    AccountsVaultEntry,
    AccountsVaultGrant,
    CallName,
} from '@jterrazz/station-api';

import { sentence } from '../../domain/format';
import { shortDay } from '../../domain/when';
import { codesOf, wordOf } from '../../engine/engine-words';

type GrantTo = AccountsVaultGrant['ToKind'];

export const VAULT_REREADS: readonly CallName[] = ['vault.list', 'accounts.list', 'grants.new'];

function grantsOf(entry: Pick<AccountsVaultEntry, 'Grants'>): readonly AccountsVaultGrant[] {
    return entry.Grants ?? [];
}

function countTo(to: GrantTo, count: number): string {
    const { One, Other } = wordOf('AccountsGrantTo', to);
    return `${String(count)} ${(count === 1 ? One : Other) ?? to}`;
}

// invariant: a count of the entry's grant rows by the kind of what holds them, worded by the engine's nouns; an
// invariant: entry with no row says its Reach in the engine's word.
export function grantedTo(entry: Pick<AccountsVaultEntry, 'Grants' | 'Reach'>): string {
    const grants = grantsOf(entry);
    if (grants.length === 0) {
        return wordOf('AccountsEntryReach', entry.Reach).Word;
    }
    return sentence(
        codesOf('AccountsGrantTo')
            .map((to) => ({ count: grants.filter((grant) => grant.ToKind === to).length, to }))
            .filter(({ count }) => count > 0)
            .map(({ to, count }) => countTo(to, count)),
    );
}

export function kindWord(entry: Pick<AccountsVaultEntry, 'Kind'>): string {
    return wordOf('AccountsKeyKind', entry.Kind).Word;
}

export function addedDay(entry: Pick<AccountsVaultEntry, 'Added'>): string {
    return shortDay(entry.Added) ?? '';
}

export function addedOn(entry: Pick<AccountsVaultEntry, 'Added'>): string | undefined {
    const day = addedDay(entry);
    return day === '' ? undefined : `added ${day}`;
}

export function aboutEntry(entry: AccountsVaultEntry): string {
    return sentence([entry.Program, kindWord(entry), entry.Who, addedOn(entry)]);
}

// invariant: a grant's line joins the engine's noun for what holds it, the workspace an agent works in, and the
// invariant: engine's own sentence of an agent's reach.
export function grantLine(grant: Pick<AccountsVaultGrant, 'ToKind' | 'Workspace'>): string {
    const to = wordOf('AccountsGrantTo', grant.ToKind);
    const where =
        grant.Workspace === undefined || grant.Workspace === ''
            ? undefined
            : `on ${grant.Workspace}`;
    return sentence([to.One ?? grant.ToKind, where, to.Detail]);
}

export function targetLines(target: AccountsTarget): readonly string[] {
    const where =
        target.Workspace === undefined || target.Workspace === ''
            ? undefined
            : `on ${target.Workspace}`;
    return [sentence([where, target.Line])].filter((line) => line !== '');
}

export function targetId(target: Pick<AccountsTarget, 'To' | 'ToKind'>): string {
    return `${target.ToKind}:${target.To}`;
}

import type { AccountsTarget, AccountsVaultEntry, CallName } from '@jterrazz/station-api';

import { sentence } from '../../../domain/format';
import { shortDay } from '../../../domain/when';
import { countOf, wordOf } from '../../words/engine-words';

export const VAULT_REREADS: readonly CallName[] = ['vault.list', 'accounts.list', 'grants.new'];

// invariant: a count of the machines the entry is granted to; an entry with none says its Reach in the engine's word.
export function grantedTo(entry: Pick<AccountsVaultEntry, 'Grants' | 'Reach'>): string {
    const count = (entry.Grants ?? []).length;
    if (count === 0) {
        return wordOf('AccountsEntryReach', entry.Reach).Word;
    }
    return countOf('machine', count);
}

export type TargetRow = { held: boolean; id: string; line: string; title: string };

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

// invariant: the targets that fit come first, the rest after them, each group in the engine's order.
export function targetRowsOf(
    targets: readonly Pick<AccountsTarget, 'Fit' | 'Line' | 'To'>[],
): readonly TargetRow[] {
    const rows = targets.map((target) => ({
        held: !canPick(target),
        id: target.To,
        line: target.Line,
        title: target.To,
    }));
    return [...rows.filter((row) => !row.held), ...rows.filter((row) => row.held)];
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

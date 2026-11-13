import type {
    AccountsAccount,
    AccountsMachine,
    AccountsReport,
    AccountsShared,
} from '@jterrazz/station-api';

import { screen } from '../../design/secret';
import type { Screened } from '../../design/secret';
import { identitiesOf } from '../../engine/provisional';
import {
    ACCOUNT_ORIGIN_WORDS,
    ACCOUNT_REASON_WORDS,
    ACCOUNT_STATE_WORDS,
} from '../../engine/words';

export type AccountsOf = {
    machine: AccountsMachine | undefined;
    shares: readonly AccountsShared[];
};

export function accountsOf(report: AccountsReport, name: string, vm: boolean): AccountsOf {
    const machine = (report.Machines ?? []).find((candidate) => candidate.name === name);
    const shares = (report.Shares ?? []).filter((share) => !vm || share.vm === name);
    return { machine, shares };
}

export function sharesOf(report: AccountsReport, account: string): readonly AccountsShared[] {
    return (report.Shares ?? []).filter((share) => share.account === account);
}

export function wordOf(account: AccountsAccount): string {
    const reason = account.reason ?? '';
    return reason === ''
        ? ACCOUNT_STATE_WORDS[account.state]
        : (ACCOUNT_REASON_WORDS[reason] ?? reason);
}

export function aboutOf(account: AccountsAccount): Screened {
    const origin = account.origin ?? '';
    return screen([
        ['identity', identitiesOf(account).join(', ')],
        ['scopes', (account.scopes ?? []).join(', ')],
        ['origin', origin === '' ? undefined : (ACCOUNT_ORIGIN_WORDS[origin] ?? origin)],
        ['source', account.source],
        ['detail', account.detail],
        ['notes', (account.notes ?? []).join(' · ')],
    ]);
}

export function whoOf(account: AccountsAccount | undefined): Screened {
    if (account === undefined) {
        return { shown: [], withheld: [] };
    }
    return screen([
        ['identity', identitiesOf(account).join(', ')],
        ['scopes', (account.scopes ?? []).join(', ')],
    ]);
}

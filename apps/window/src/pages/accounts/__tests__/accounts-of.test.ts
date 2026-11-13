import type { AccountsAccount, AccountsReport } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { holdsFor } from '../../../engine/shown';
import { aboutOf, accountsOf, sharesOf, whoOf, wordOf } from '../accounts-of';

const GITHUB: AccountsAccount = {
    identity: ['jterrazz'],
    origin: 'shared',
    provider: 'github',
    scopes: ['repo', 'workflow'],
    state: 'ok',
};

const REPORT: AccountsReport = {
    Engine: '',
    Machines: [
        { accounts: [GITHUB], kind: 'computer', name: 'mbp', this_mac: true },
        { accounts: [GITHUB], kind: 'vm', name: 'workspace' },
    ],
    Now: '2026-11-12T10:00:00Z',
    ShareList: ['github'],
    Shares: [
        { account: 'github', at: '', until: '2026-11-12T19:00:00Z', vm: 'workspace' },
        { account: 'claude', at: '', until: '2026-11-18T10:00:00Z', vm: 'gate' },
    ],
};

describe('accounts of', () => {
    test("a Mac's tab holds every share it made", () => {
        // Given - the report read for the Mac
        const of = accountsOf(REPORT, 'mbp', false);

        // Then - both shares, and the Mac's own machine
        expect(of.machine?.kind).toBe('computer');
        expect(of.shares).toHaveLength(2);
    });

    test("a VM's tab holds the shares made with it alone", () => {
        // Given - the report of a Mac that shared with two machines
        const { shares } = accountsOf(REPORT, 'workspace', true);

        // Then - the github share of workspace
        expect(shares.map((share) => share.account)).toStrictEqual(['github']);
    });

    test("a reason is the pill's word before the state", () => {
        // Given - the github account, signed in
        // Then - a share ending soon says so, a plain one says its state
        expect(wordOf({ ...GITHUB, reason: 'expiring', state: 'degraded' })).toBe('ends soon');
        expect(wordOf(GITHUB)).toBe('signed in');
        expect(wordOf({ ...GITHUB, reason: 'brand-new' })).toBe('brand-new');
    });

    test('a detail that carries a token is withheld, by its name', () => {
        // Given - an account whose detail leaked its token
        const about = aboutOf({
            ...GITHUB,
            detail: ['ghp', '0123456789abcdefghijABCDEFGHIJ'].join('_'),
        });

        // Then - the identity, scopes and origin are shown; the detail is not
        expect(about.shown).toStrictEqual(['jterrazz', 'repo, workflow', 'a copy, shared']);
        expect(about.withheld).toStrictEqual(['detail']);
    });

    test('an identity the engine could not read is left out, never drawn as a question mark', () => {
        // Given - an account whose identity came unread
        const unread = { ...GITHUB, identity: ['?'] };

        // When - what it is and who it is are written
        const about = aboutOf(unread);
        const who = whoOf(unread);

        // Then - the scopes and origin alone, then the scopes alone
        expect(about.shown).toStrictEqual(['repo, workflow', 'a copy, shared']);
        expect(who.shown).toStrictEqual(['repo, workflow']);
    });

    test('a shared account names who it is on its Mac, and every machine it is shared with', () => {
        // Given - the report of a Mac that shared github with workspace

        // When - who github is, and its shares, are read
        const who = whoOf(GITHUB);
        const shares = sharesOf(REPORT, 'github');

        // Then - its identity and scopes, and workspace alone
        expect(who.shown).toStrictEqual(['jterrazz', 'repo, workflow']);
        expect(shares.map((share) => share.vm)).toStrictEqual(['workspace']);
        expect(whoOf(undefined).shown).toStrictEqual([]);
    });

    test('a share says how long it still holds, against the report hour', () => {
        // Given - the report read at 10:00
        // Then - nine hours, six days, and an end already past
        expect(holdsFor('2026-11-12T19:00:00Z', REPORT.Now)).toStrictEqual({
            tone: 'warn',
            word: 'ends in 9 h',
        });
        expect(holdsFor('2026-11-18T10:00:00Z', REPORT.Now).word).toBe('ends in 6 d');
        expect(holdsFor('2026-11-12T09:00:00Z', REPORT.Now)).toStrictEqual({
            tone: 'plain',
            word: 'ended at 2026-11-12T09:00:00Z',
        });
    });
});

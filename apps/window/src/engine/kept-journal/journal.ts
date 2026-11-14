import type {
    AppsUpdateInput,
    MachinesListOutput,
    MachinesMacView,
    StationWord,
} from '@jterrazz/station-api';

import { EngineError } from '../engine-error';
import { READING } from '../entry';
import type { Reading } from '../entry';
import { useWatch } from '../react';
import type { Shown } from '../shown';
import { ANSWERS } from './answers';

// outside: PROVISIONAL, the window's copy of a read the engine does not have yet: its kept journal, read by day
// outside: (the engine's lot L6). Nothing of it is on the wire: these types are the window's wish, typed by hand, and
// outside: answers.ts holds what the engine would answer over the bench's worlds, every word in it as the engine would
// outside: send it. This folder is deleted whole when the engine's call lands; the call is then bridged, recorded
// outside: over the worlds, and its generated types and words.json replace these.

export type JournalOutcome = 'done' | 'failed' | 'held';

export type JournalSubjectKind = 'app' | 'machine' | 'setting' | 'skill';

export type JournalSubject = { Kind: JournalSubjectKind; Name: string };

export type JournalUpgrade = { From: string; To: string };

export type JournalRetry = {
    Call: 'apps.update';
    Input: AppsUpdateInput;
    Machine: string;
    Verb: string;
};

// value: `By` is who did it ("you", a session, a run), empty when the group names who; `On` the machines it
// value: concerns; `Detail` the engine's sentence on it (why it was held or failed, what it freed).
export type JournalLine = {
    At: string;
    Text: string;
    Subject: JournalSubject | null;
    By: string;
    Purpose: string;
    On: null | string[];
    Outcome: JournalOutcome;
    Upgrade: JournalUpgrade | null;
    Detail: string;
    Retry: JournalRetry | null;
};

// value: `Night` marks the group of the last nightly run, the one Home draws.
export type JournalGroup = {
    Title: string;
    Sum: string;
    Night: boolean;
    Lines: JournalLine[] | null;
};

export type JournalReadInput = { Mac?: string };

export type JournalReadOutput = { Groups: JournalGroup[] | null };

export const JOURNAL_WORDS: { JournalOutcome: Readonly<Record<JournalOutcome, StationWord>> } = {
    JournalOutcome: {
        done: { Tone: 'plain', Word: 'Done' },
        failed: { Tone: 'bad', Word: 'Failed' },
        held: { Tone: 'warn', Word: 'Held' },
    },
};

export function outcomeShown(outcome: JournalOutcome): Shown {
    const said = JOURNAL_WORDS.JournalOutcome[outcome];
    return { tone: said.Tone, word: said.Word };
}

export type JournalAnswer = {
    fleet: string;
    out: (macs: readonly MachinesMacView[]) => JournalReadOutput;
};

const NOT_ON_THE_WIRE = new EngineError(
    undefined,
    'the kept journal is not on the wire yet, and no answer is written for these machines',
);

// outside: an answer is chosen by the machines machines.list lists, each Mac's VMs in turn, one not seen marked `?`,
// outside: so no Mac's own name is written in it; the Mac asked is applied as the engine would apply it, keeping the
// outside: lines of that Mac and its machines, each line's machines narrowed to them, and no group's sum.
export function fleetOf(macs: readonly MachinesMacView[]): string {
    return macs
        .map((mac) => {
            const vms = (mac.VMs ?? []).map((vm) => vm.Name).join(',');
            return mac.Reachable && !mac.Asleep ? vms : `${vms}?`;
        })
        .join(' | ');
}

function narrowedTo(out: JournalReadOutput, names: ReadonlySet<string>): JournalReadOutput {
    const groups = (out.Groups ?? []).map((group) => ({
        ...group,
        Lines: (group.Lines ?? [])
            .map((line) => ({ ...line, On: (line.On ?? []).filter((name) => names.has(name)) }))
            .filter((line) => line.On.length > 0),
        Sum: '',
    }));
    return { Groups: groups.filter((group) => group.Lines.length > 0) };
}

export function answerFor(
    listing: MachinesListOutput,
    input: JournalReadInput,
): JournalReadOutput | undefined {
    const macs = listing.All.Macs ?? [];
    const answer = ANSWERS.find((one) => one.fleet === fleetOf(macs));
    if (answer === undefined) {
        return undefined;
    }
    const out = answer.out(macs);
    if (input.Mac === undefined) {
        return out;
    }
    const mac = macs.find((one) => one.Name === input.Mac);
    const names = mac ? [mac.Name, ...(mac.VMs ?? []).map((vm) => vm.Name)] : [];
    return narrowedTo(out, new Set(names));
}

export function useJournal(input: JournalReadInput): Reading<JournalReadOutput> {
    const listing = useWatch('machines.list', {});
    if (listing.kind !== 'read') {
        return listing;
    }
    const out = answerFor(listing.out, input);
    if (out === undefined) {
        return listing.done ? { error: NOT_ON_THE_WIRE, kind: 'failed' } : READING;
    }
    return { done: listing.done, kind: 'read', out };
}

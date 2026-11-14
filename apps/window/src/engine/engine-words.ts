import type {
    AccountsStepKind,
    MachinefileAction,
    PageNoun,
    StationWord,
    StationWords,
} from '@jterrazz/station-api';
import words from '@jterrazz/station-api/words.json';

import calls from '../../../engine/remote/api/calls.json';

export type Worded = keyof StationWords;

export type CodeOf<E extends Worded> = keyof StationWords[E];

type Table<E extends Worded> = Readonly<Record<CodeOf<E>, StationWord>>;

const WORDS: { readonly [E in Worded]: Table<E> } = words;

export function wordOf<E extends Worded>(of: E, code: CodeOf<E>): StationWord {
    const table: Table<E> = WORDS[of];
    return table[code];
}

export function wordOfSent(of: Worded, sent: string): string {
    const table: Readonly<Record<string, StationWord>> = WORDS[of];
    return Object.hasOwn(table, sent) ? (table[sent]?.Word ?? sent) : sent;
}

export function codesOf<E extends Worded>(of: E): readonly CodeOf<E>[] {
    const table: Table<E> = WORDS[of];
    return Object.keys(table).filter((code): code is Extract<CodeOf<E>, string> =>
        Object.hasOwn(table, code),
    );
}

export function detailOf<E extends Worded>(of: E, code: CodeOf<E>): string {
    return wordOf(of, code).Detail ?? '';
}

// outside: words.json gives an action and a step kind a Heading the generated StationWord does not type.
function headingFrom(said: StationWord, code: string): string {
    return 'Heading' in said && typeof said.Heading === 'string' ? said.Heading : code;
}

export function headingOf(action: MachinefileAction): string {
    return headingFrom(wordOf('MachinefileAction', action), action);
}

export function stepHeadingOf(kind: AccountsStepKind): string {
    return headingFrom(wordOf('AccountsStepKind', kind), kind);
}

export function nounOf(noun: PageNoun, count: number): string {
    const { One, Other } = wordOf('PageNoun', noun);
    return (count === 1 ? One : Other) ?? noun;
}

const CONSEQUENCES: ReadonlyMap<string, string> = new Map(
    calls.calls.flatMap((call) =>
        'consequence' in call ? [[call.name, call.consequence] as const] : [],
    ),
);

export function consequenceOf(call: string, subject: string): string | undefined {
    return CONSEQUENCES.get(call)?.replace('%s', () => subject);
}

export function countOf(noun: PageNoun, count: number): string {
    return `${String(count)} ${nounOf(noun, count)}`;
}

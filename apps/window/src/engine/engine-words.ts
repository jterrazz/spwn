import type { StationWord, StationWords } from '@jterrazz/station-api';
import words from '@jterrazz/station-api/words.json';

type Worded = keyof StationWords;

type CodeOf<E extends Worded> = keyof StationWords[E];

type Table<E extends Worded> = Readonly<Record<CodeOf<E>, StationWord>>;

const WORDS: { readonly [E in Worded]: Table<E> } = words;

export function wordOf<E extends Worded>(of: E, code: CodeOf<E>): StationWord {
    const table: Table<E> = WORDS[of];
    return table[code];
}

export function codesOf<E extends Worded>(of: E): readonly CodeOf<E>[] {
    const table: Table<E> = WORDS[of];
    return Object.keys(table).filter((code): code is Extract<CodeOf<E>, string> =>
        Object.hasOwn(table, code),
    );
}

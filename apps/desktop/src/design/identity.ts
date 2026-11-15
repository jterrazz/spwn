export const IDENTITIES = [
    'blue',
    'purple',
    'teal',
    'indigo',
    'pink',
    'sky',
    'lilac',
    'slate',
] as const;

export type Identity = 'system' | (typeof IDENTITIES)[number];

// value: the mockup's seven machine colours, in its order; slate is a colour of what is no machine.
const MACHINE_IDENTITIES = [
    'blue',
    'purple',
    'teal',
    'indigo',
    'pink',
    'sky',
    'lilac',
] as const satisfies readonly Identity[];

// invariant: a machine's colour is the next of the seven by its place among the machines listed, every Mac's in
// invariant: turn, so two machines of a Mac differ while a Mac holds seven or fewer; a Mac itself has none.
export function identityOf(name: string, listed: readonly string[]): Identity | undefined {
    const at = listed.indexOf(name);
    return at === -1 ? undefined : MACHINE_IDENTITIES[at % MACHINE_IDENTITIES.length];
}

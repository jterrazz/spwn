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

export function identityOf(name: string): Identity {
    let sum = 0;
    for (const char of name) {
        sum += char.codePointAt(0) ?? 0;
    }
    return IDENTITIES[sum % IDENTITIES.length] ?? 'system';
}

const SECRET_SHAPES: readonly RegExp[] = [
    /\b(?:gh[oprsu]_|github_pat_)\w{20,}/u,
    /\bsk-[\w-]{16,}/u,
    /\bnpm_\w{20,}/u,
    /\bxox[abpr]-[\w-]{10,}/u,
    /\beyJ[\w-]{8,}\.[\w-]{8,}\.[\w-]+/u,
    /\bAKIA[\dA-Z]{16}\b/u,
    /-----BEGIN [A-Z ]*PRIVATE KEY-----/u,
    // hazard: a run of letters and digits is withheld unless it is a path written from a root or a digest named
    // hazard: by its algorithm; a bare hex run cannot be told from a hex token, so it is withheld.
    /(?<![\w+./=~-])(?!\/)(?<!sha(?:1|256|384|512):)(?=[\w+/=-]*\d)(?=[\w+/=-]*[A-Za-z])[\w+/=-]{32,}/u,
];

export function looksSecret(value: string): boolean {
    return SECRET_SHAPES.some((shape) => shape.test(value));
}

export const WITHHELD = 'Withheld';

export function shownOrWithheld(text: string): string {
    return looksSecret(text) ? WITHHELD : text;
}

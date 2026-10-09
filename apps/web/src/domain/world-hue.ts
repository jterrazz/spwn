const UINT32 = 4_294_967_296;
const INT32_MAX = 2_147_483_647;

/** The 31-multiplier string hash, unsigned 32-bit — the same value for the same id on every surface. */
export function stringHash(text: string): number {
    let hash = 0;
    for (const character of text) {
        hash = (hash * 31 + (character.codePointAt(0) ?? 0)) % UINT32;
    }
    return hash;
}

/** The same hash read as a signed 32-bit integer, made positive. */
export function positiveHash(text: string): number {
    const hash = stringHash(text);
    return Math.abs(hash > INT32_MAX ? hash - UINT32 : hash);
}

// invariant: the sidebar, the page headers and the carousel draw the same hue for the same world id, so a world reads as one planet everywhere
export function worldHue(id: null | string | undefined): number {
    return id === null || id === undefined || id === '' ? 0 : stringHash(id) % 360;
}

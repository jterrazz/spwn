export function capitalised(text: string): string {
    return text.charAt(0).toUpperCase() + text.slice(1);
}

// invariant: an engine message may open on a machine's or a file's name, so a sentence keeps its first letter.
export function asSentence(text: string): string {
    const shown = text.trim();
    return shown === '' || /[.!?…:]$/u.test(shown) ? shown : `${shown}.`;
}

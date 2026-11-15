import { Value } from '../../design/index';

export type Span = { at: number; code: boolean; text: string };

export function codeSpansOf(text: string): Span[] {
    const parts = text.split('`');
    if (parts.length % 2 === 0) {
        return [{ at: 0, code: false, text }];
    }
    let at = 0;
    return parts
        .map((part, index) => {
            const span = { at, code: index % 2 === 1, text: part };
            at += part.length + 1;
            return span;
        })
        .filter((part) => part.text !== '');
}

export function Marked({ children }: { children: string }) {
    return (
        <>
            {codeSpansOf(children).map((part) =>
                part.code ? (
                    <Value key={part.at}>{part.text}</Value>
                ) : (
                    <span key={part.at}>{part.text}</span>
                ),
            )}
        </>
    );
}

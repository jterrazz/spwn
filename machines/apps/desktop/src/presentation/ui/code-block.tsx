import { useLayoutEffect, useRef, useState } from 'react';
import type { RefObject } from 'react';

import type { Measured } from './measure';
import { heightOf, spanOf } from './windowed';

import styles from './code-block.module.css';

export type CodeLine = { number: number; text: string; change?: 'added' | 'removed' | undefined };

export type CodeBlockProps = {
    lines: readonly CodeLine[];
    label: string;
    numbered?: boolean;
    log?: boolean;
    tall?: number | undefined;
    empty?: string;
    foot?: string;
    memory?: string | undefined;
};

// invariant: the height of a line in code-block.module.css, `--row`: the window is counted in it.
const ROW = 21;
const MOST = 26 * ROW;
const LEAST = 3;
const OVERSCAN = 12;
const CHANGE_MARKS = { added: '+', removed: '−' } as const;

const scrolls = new Map<string, number>();

function pixels(name: '--extent' | '--height' | '--offset', value: number): Measured {
    const size = `${String(value)}px`;
    if (name === '--extent') {
        return { '--extent': size };
    }
    return name === '--height' ? { '--height': size } : { '--offset': size };
}

function useRestored(
    box: RefObject<HTMLDivElement | null>,
    memory: string | undefined,
    ready: boolean,
) {
    const restored = useRef(false);
    useLayoutEffect(() => {
        const node = box.current;
        if (!node || !ready || restored.current || memory === undefined) {
            return;
        }
        restored.current = true;
        node.scrollTo({ top: scrolls.get(memory) ?? 0 });
    }, [box, memory, ready]);
}

function Line({ line, numbered }: { line: CodeLine; numbered: boolean }) {
    return (
        <div className={`${styles.line} ${line.change === undefined ? '' : styles[line.change]}`}>
            {numbered ? <span className={styles.number}>{line.number}</span> : null}
            {line.change === undefined ? null : (
                <span className={styles.mark}>{CHANGE_MARKS[line.change]}</span>
            )}
            <span className={styles.text}>{line.text}</span>
        </div>
    );
}

type Viewed = { lines: readonly CodeLine[]; numbered: boolean; height: number; top: number };

function Shown({ lines, numbered, height, top }: Viewed) {
    const { start, end } = spanOf({
        count: lines.length,
        height,
        overscan: OVERSCAN,
        row: ROW,
        top,
    });
    return (
        <div className={styles.extent} style={pixels('--extent', lines.length * ROW)}>
            <div className={styles.span} style={pixels('--offset', start * ROW)}>
                {lines.slice(start, end).map((line) => (
                    <Line key={line.number} line={line} numbered={numbered} />
                ))}
            </div>
        </div>
    );
}

export function CodeBlock({
    lines,
    label,
    numbered = false,
    log = false,
    tall,
    empty = '',
    foot = '',
    memory,
}: CodeBlockProps) {
    const box = useRef<HTMLDivElement>(null);
    const [top, setTop] = useState(0);
    const height = heightOf(Math.max(tall ?? lines.length, LEAST), ROW, MOST);
    useRestored(box, memory, lines.length > 0);
    return (
        <div className={styles.code}>
            <div
                aria-label={label}
                className={styles.lines}
                onScroll={(event) => {
                    const at = event.currentTarget.scrollTop;
                    if (memory !== undefined) {
                        scrolls.set(memory, at);
                    }
                    setTop(at);
                }}
                ref={box}
                role={log ? 'log' : 'region'}
                style={pixels('--height', height)}
                tabIndex={0}
            >
                {lines.length === 0 ? (
                    <p className={styles.none}>{empty}</p>
                ) : (
                    <Shown height={height} lines={lines} numbered={numbered} top={top} />
                )}
            </div>
            {foot === '' ? null : <p className={styles.foot}>{foot}</p>}
        </div>
    );
}

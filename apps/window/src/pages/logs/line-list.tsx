import { useLayoutEffect, useMemo, useRef, useState } from 'react';
import type { RefObject } from 'react';

import { countOf, filtered, numbered } from './lines';
import { heightOf, spanOf } from './windowed';

import styles from './logs.module.css';

// invariant: the height of a line in logs.module.css, `--row`: the window is counted in it.
const ROW = 21;
const MOST = 26 * ROW;
const OVERSCAN = 12;

const scrolls = new Map<string, number>();

type LineListProps = { lines: readonly string[]; query: string; memory: string; foot?: string };

function useRestored(box: RefObject<HTMLDivElement | null>, memory: string, ready: boolean) {
    const restored = useRef(false);
    useLayoutEffect(() => {
        const node = box.current;
        if (!node || !ready || restored.current) {
            return;
        }
        restored.current = true;
        node.scrollTo({ top: scrolls.get(memory) ?? 0 });
    }, [box, memory, ready]);
}

export function LineList({ lines, query, memory, foot = '' }: LineListProps) {
    const all = useMemo(() => numbered(lines), [lines]);
    const shown = useMemo(() => filtered(all, query), [all, query]);
    const box = useRef<HTMLDivElement>(null);
    const [top, setTop] = useState(0);
    const height = heightOf(Math.max(all.length, 3), ROW, MOST);
    useRestored(box, memory, all.length > 0);
    const { start, end } = spanOf({
        count: shown.length,
        height,
        overscan: OVERSCAN,
        row: ROW,
        top,
    });
    return (
        <div className={styles.box}>
            <div
                aria-label="Log lines"
                className={styles.list}
                onScroll={(event) => {
                    const at = event.currentTarget.scrollTop;
                    scrolls.set(memory, at);
                    setTop(at);
                }}
                ref={box}
                role="log"
                style={{ height }}
                tabIndex={0}
            >
                {shown.length === 0 ? (
                    <p className={styles.none}>
                        {all.length === 0
                            ? 'The engine read no line.'
                            : `No line holds "${query}".`}
                    </p>
                ) : (
                    <div className={styles.extent} style={{ height: shown.length * ROW }}>
                        <div
                            className={styles.span}
                            style={{ transform: `translateY(${String(start * ROW)}px)` }}
                        >
                            {shown.slice(start, end).map((line) => (
                                <div className={styles.line} key={line.number}>
                                    <span className={styles.number}>{line.number}</span>
                                    <span className={styles.text}>{line.text}</span>
                                </div>
                            ))}
                        </div>
                    </div>
                )}
            </div>
            <p className={styles.foot}>
                {[countOf(shown.length, all.length, query), ...(foot === '' ? [] : [foot])].join(
                    ' · ',
                )}
            </p>
        </div>
    );
}

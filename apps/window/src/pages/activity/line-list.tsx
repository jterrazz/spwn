import { useMemo } from 'react';

import { CodeBlock } from '../../design/index';
import { SEPARATOR } from '../../design/separator';
import { countOf, filtered, numbered } from './lines';

type LineListProps = { lines: readonly string[]; query: string; memory: string; foot?: string };

export function LineList({ lines, query, memory, foot = '' }: LineListProps) {
    const all = useMemo(() => numbered(lines), [lines]);
    const shown = useMemo(() => filtered(all, query), [all, query]);
    return (
        <CodeBlock
            empty={all.length === 0 ? 'The engine read no line.' : `No line holds "${query}".`}
            foot={[countOf(shown.length, all.length, query), ...(foot === '' ? [] : [foot])].join(
                SEPARATOR,
            )}
            label="Log lines"
            lines={shown}
            log
            memory={memory}
            numbered
            tall={all.length}
        />
    );
}

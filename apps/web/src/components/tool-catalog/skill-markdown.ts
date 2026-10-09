/** The few markdown forms a skill file uses: headings, list items, fenced code and paragraphs. */
export type SkillBlock =
    | { kind: 'code'; text: string }
    | { kind: 'heading'; level: 2 | 3; text: string }
    | { kind: 'item'; marker: string; text: string }
    | { kind: 'paragraph'; text: string };

const FENCE = '```';
const NUMBERED = /^(?<number>\d+)\.\s+(?<text>.+)/u;

function lineBlock(line: string): SkillBlock | undefined {
    const trimmed = line.trimStart();
    if (line.startsWith('### ')) {
        return { kind: 'heading', level: 3, text: line.slice(4) };
    }
    if (line.startsWith('## ')) {
        return { kind: 'heading', level: 2, text: line.slice(3) };
    }
    if (line.startsWith('# ') || line.trim() === '') {
        return undefined;
    }
    if (trimmed.startsWith('- ')) {
        return { kind: 'item', marker: '-', text: trimmed.slice(2) };
    }
    const numbered = NUMBERED.exec(trimmed)?.groups;
    // Stryker disable next-line ConditionalExpression: the pattern's two groups match together or not at all
    if (numbered?.number !== undefined && numbered.text !== undefined) {
        return { kind: 'item', marker: `${numbered.number}.`, text: numbered.text };
    }
    return { kind: 'paragraph', text: line };
}

/** A skill file's markdown as the blocks the tool page renders; its title, the page already shows, is left out. */
export function skillBlocks(content: string): SkillBlock[] {
    const blocks: SkillBlock[] = [];
    let code: null | string[] = null;
    for (const line of content.split('\n')) {
        if (line.trimStart().startsWith(FENCE)) {
            if (code !== null) {
                blocks.push({ kind: 'code', text: code.join('\n') });
            }
            code = code === null ? [] : null;
        } else if (code === null) {
            const block = lineBlock(line);
            if (block !== undefined) {
                blocks.push(block);
            }
        } else {
            code.push(line);
        }
    }
    if (code !== null) {
        blocks.push({ kind: 'code', text: code.join('\n') });
    }
    return blocks;
}

/** A line of text split into its plain runs and its `code` spans. */
export function inlineRuns(text: string): { code: boolean; text: string }[] {
    return text
        .split(/(?<code>`[^`]+`)/u)
        .filter((run) => run !== '')
        .map((run) =>
            run.startsWith('`') && run.endsWith('`')
                ? { code: true, text: run.slice(1, -1) }
                : { code: false, text: run },
        );
}

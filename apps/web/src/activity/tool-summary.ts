import { textOf } from '../api/fields';

const SUMMARIES: Record<string, { field: string; max?: number }> = {
    Agent: { field: 'description', max: 40 },
    Bash: { field: 'command', max: 60 },
    Edit: { field: 'file_path' },
    Glob: { field: 'pattern' },
    Grep: { field: 'pattern', max: 40 },
    Read: { field: 'file_path' },
    Write: { field: 'file_path' },
};

export function truncateOneLine(text: string, max: number): string {
    const line = text.split('\n')[0] ?? '';
    return line.length > max ? `${line.slice(0, max)}...` : line;
}

export function truncateContent(content: string, maxLines: number): string {
    const lines = content.split('\n');
    if (lines.length <= maxLines) {
        return content;
    }
    return `${lines.slice(0, maxLines).join('\n')}\n... (${lines.length - maxLines} more lines)`;
}

/** The one input field that tells a tool call apart — the file read, the command run, the pattern searched — shortened to a line. */
export function toolInputSummary(tool: string, input: Record<string, unknown>): string {
    const summary = SUMMARIES[tool];
    if (summary === undefined) {
        return '';
    }
    const value = textOf(input, summary.field);
    return summary.max === undefined ? value : truncateOneLine(value, summary.max);
}

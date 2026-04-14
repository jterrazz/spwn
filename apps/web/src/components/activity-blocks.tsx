'use client';

import {
    IconAlertTriangle,
    IconBrain,
    IconChevronDown,
    IconChevronRight,
    IconFileText,
    IconGlobe,
    IconPencil,
    IconSearch,
    IconTerminal,
    IconTool,
} from '@tabler/icons-react';
import { useState } from 'react';

import type { ActivityBlock, ActivityMessage } from '@/lib/activity-types';

// ── Tool icon mapping ──

const TOOL_ICONS: Record<string, typeof IconTool> = {
    Bash: IconTerminal,
    Read: IconFileText,
    Write: IconPencil,
    Edit: IconPencil,
    Grep: IconSearch,
    Glob: IconSearch,
    WebFetch: IconGlobe,
    WebSearch: IconGlobe,
    Agent: IconBrain,
};

function ToolIcon({ tool }: { tool: string }) {
    const Icon = TOOL_ICONS[tool] || IconTool;
    return <Icon size={13} />;
}

// ── Collapsible wrapper ──

function CollapsibleBlock({
    label,
    icon,
    defaultOpen = false,
    accent = 'text-muted-foreground/40',
    children,
}: {
    label: string;
    icon: React.ReactNode;
    defaultOpen?: boolean;
    accent?: string;
    children: React.ReactNode;
}) {
    const [open, setOpen] = useState(defaultOpen);

    return (
        <div className="rounded-lg border border-white/[0.06] bg-white/[0.02] overflow-hidden">
            <button
                className={`w-full flex items-center gap-2 px-3 py-1.5 text-[11px] font-mono ${accent} hover:bg-white/[0.03] transition-colors`}
                onClick={() => setOpen(!open)}
            >
                {open ? <IconChevronDown size={12} /> : <IconChevronRight size={12} />}
                <span className="opacity-60">{icon}</span>
                <span className="flex-1 text-left truncate">{label}</span>
            </button>
            {open && (
                <div className="border-t border-white/[0.04] px-3 py-2 text-[11px] font-mono text-muted-foreground/50 max-h-[300px] overflow-auto">
                    {children}
                </div>
            )}
        </div>
    );
}

// ── Individual block renderers ──

function ThinkingBlockView({ block }: { block: { content: string } }) {
    return (
        <CollapsibleBlock
            accent="text-purple-400/50"
            icon={<IconBrain size={13} />}
            label="Thinking..."
        >
            <pre className="whitespace-pre-wrap break-words leading-relaxed text-purple-300/40">
                {block.content}
            </pre>
        </CollapsibleBlock>
    );
}

function ToolUseBlockView({
    block,
    result,
}: {
    block: { tool: string; input: Record<string, unknown>; id: string };
    result?: { content: string; isError: boolean };
}) {
    // Build a concise label from the tool input
    const inputSummary = getToolInputSummary(block.tool, block.input);

    return (
        <CollapsibleBlock
            accent={result?.isError ? 'text-red-400/60' : 'text-blue-400/50'}
            defaultOpen={Boolean(result?.isError)}
            icon={<ToolIcon tool={block.tool} />}
            label={`${block.tool}${inputSummary ? `  ${inputSummary}` : ''}`}
        >
            {/* Input */}
            <div className="space-y-1">
                {Object.entries(block.input || {}).map(([key, value]) => (
                    <div className="flex gap-2" key={key}>
                        <span className="text-muted-foreground/30 shrink-0">{key}:</span>
                        <span className="text-foreground/50 break-all">
                            {typeof value === 'string'
                                ? value.length > 200
                                    ? `${value.slice(0, 200)}...`
                                    : value
                                : JSON.stringify(value)}
                        </span>
                    </div>
                ))}
            </div>

            {/* Result */}
            {result && (
                <>
                    <div className="border-t border-white/[0.04] my-2" />
                    <pre
                        className={`whitespace-pre-wrap break-words leading-relaxed ${
                            result.isError ? 'text-red-400/60' : 'text-green-400/40'
                        }`}
                    >
                        {truncateContent(result.content, 20)}
                    </pre>
                </>
            )}
        </CollapsibleBlock>
    );
}

function TextBlockView({ block }: { block: { content: string } }) {
    return (
        <pre className="text-xs font-mono whitespace-pre-wrap break-words leading-relaxed text-foreground/70">
            {block.content}
        </pre>
    );
}

function ErrorBlockView({ block }: { block: { content: string } }) {
    return (
        <div className="flex items-start gap-2 rounded-lg border border-red-500/15 bg-red-500/[0.06] px-3 py-2">
            <IconAlertTriangle className="text-red-400/60 shrink-0 mt-0.5" size={14} />
            <pre className="text-[11px] font-mono whitespace-pre-wrap break-words text-red-400/70">
                {block.content}
            </pre>
        </div>
    );
}

function StatusBlockView({ block }: { block: { status: string; tool?: string } }) {
    const labels: Record<string, string> = {
        thinking: 'Thinking...',
        tool_calling: block.tool ? `Running ${block.tool}...` : 'Executing...',
        responding: 'Responding...',
        done: 'Done',
    };

    if (block.status === 'done') {
        return null;
    }

    return (
        <div className="flex items-center gap-2 py-1">
            <div className="w-2 h-2 rounded-full bg-foreground/30 animate-pulse" />
            <span className="text-[11px] text-muted-foreground/40">
                {labels[block.status] || block.status}
            </span>
        </div>
    );
}

// ── Main activity renderer ──

export function ActivityBlocksRenderer({ blocks }: { blocks: ActivityBlock[] }) {
    // Pair tool_use blocks with their results
    const resultMap = new Map<string, { content: string; isError: boolean }>();
    for (const block of blocks) {
        if (block.type === 'tool_result') {
            resultMap.set(block.id, { content: block.content, isError: block.isError });
        }
    }

    return (
        <div className="space-y-2">
            {blocks.map((block, i) => {
                // Skip tool_result blocks - they're rendered inline with tool_use
                if (block.type === 'tool_result') {
                    return null;
                }

                switch (block.type) {
                    case 'thinking': {
                        return <ThinkingBlockView block={block} key={i} />;
                    }
                    case 'tool_use': {
                        return (
                            <ToolUseBlockView
                                block={block}
                                key={i}
                                result={resultMap.get(block.id)}
                            />
                        );
                    }
                    case 'text': {
                        return <TextBlockView block={block} key={i} />;
                    }
                    case 'error': {
                        return <ErrorBlockView block={block} key={i} />;
                    }
                    case 'status': {
                        return <StatusBlockView block={block} key={i} />;
                    }
                    default: {
                        return null;
                    }
                }
            })}
        </div>
    );
}

// ── Activity message renderer (replaces simple <pre> in chat) ──

export function ActivityMessageView({ message }: { message: ActivityMessage }) {
    const hasBlocks = message.blocks.length > 0;
    const hasOnlyText = message.blocks.every((b) => b.type === 'text');

    // If it's just plain text (no tool calls, no thinking), render simply
    if (!hasBlocks || hasOnlyText) {
        const text = message.blocks
            .filter((b): b is { type: 'text'; content: string } => b.type === 'text')
            .map((b) => b.content)
            .join('');

        return (
            <pre className="text-xs font-mono whitespace-pre-wrap break-words leading-relaxed">
                {text || '...'}
            </pre>
        );
    }

    // Rich rendering with activity blocks
    return (
        <div className="space-y-2">
            <ActivityBlocksRenderer blocks={message.blocks} />
            {message.cost !== undefined && (
                <div className="flex items-center gap-3 text-[9px] font-mono text-muted-foreground/20 pt-1">
                    {message.duration !== undefined && (
                        <span>{(message.duration / 1000).toFixed(1)}s</span>
                    )}
                    <span>${message.cost.toFixed(4)}</span>
                </div>
            )}
        </div>
    );
}

// ── Helpers ──

function getToolInputSummary(tool: string, input: Record<string, unknown>): string {
    switch (tool) {
        case 'Read': {
            return (input.file_path as string) || '';
        }
        case 'Edit': {
            return (input.file_path as string) || '';
        }
        case 'Write': {
            return (input.file_path as string) || '';
        }
        case 'Bash': {
            return truncateOneLine((input.command as string) || '', 60);
        }
        case 'Grep': {
            return truncateOneLine((input.pattern as string) || '', 40);
        }
        case 'Glob': {
            return (input.pattern as string) || '';
        }
        case 'Agent': {
            return truncateOneLine((input.description as string) || '', 40);
        }
        default: {
            return '';
        }
    }
}

function truncateOneLine(text: string, max: number): string {
    const line = text.split('\n')[0];
    return line.length > max ? `${line.slice(0, max)}...` : line;
}

function truncateContent(content: string, maxLines: number): string {
    const lines = content.split('\n');
    if (lines.length <= maxLines) {
        return content;
    }
    return `${lines.slice(0, maxLines).join('\n')}\n... (${lines.length - maxLines} more lines)`;
}

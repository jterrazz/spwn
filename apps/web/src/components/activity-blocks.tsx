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

import type { ActivityBlock, ActivityMessage } from '../activity/blocks';
import { toolInputSummary, truncateContent } from '../activity/tool-summary';

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
    const Icon = TOOL_ICONS[tool] ?? IconTool;
    return <Icon size={13} />;
}

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
        <div className="overflow-hidden rounded-lg border border-white/[0.06] bg-white/[0.02]">
            <button
                className={`flex w-full items-center gap-2 px-3 py-1.5 font-mono text-[11px] ${accent} transition-colors hover:bg-white/[0.03]`}
                onClick={() => {
                    setOpen(!open);
                }}
                type="button"
            >
                {open ? <IconChevronDown size={12} /> : <IconChevronRight size={12} />}
                <span className="opacity-60">{icon}</span>
                <span className="flex-1 truncate text-left">{label}</span>
            </button>
            {open && (
                <div className="text-muted-foreground/50 max-h-[300px] overflow-auto border-t border-white/[0.04] px-3 py-2 font-mono text-[11px]">
                    {children}
                </div>
            )}
        </div>
    );
}

function ThinkingBlockView({ block }: { block: { content: string } }) {
    return (
        <CollapsibleBlock
            accent="text-purple-400/50"
            icon={<IconBrain size={13} />}
            label="Thinking..."
        >
            <pre className="leading-relaxed break-words whitespace-pre-wrap text-purple-300/40">
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
    result?: undefined | { content: string; isError: boolean };
}) {
    const inputSummary = toolInputSummary(block.tool, block.input);

    return (
        <CollapsibleBlock
            accent={result?.isError === true ? 'text-red-400/60' : 'text-blue-400/50'}
            defaultOpen={result?.isError === true}
            icon={<ToolIcon tool={block.tool} />}
            label={`${block.tool}${inputSummary === '' ? '' : `  ${inputSummary}`}`}
        >
            <div className="space-y-1">
                {Object.entries(block.input).map(([key, value]) => {
                    let displayValue: string;
                    if (typeof value === 'string') {
                        displayValue = value.length > 200 ? `${value.slice(0, 200)}...` : value;
                    } else {
                        displayValue = JSON.stringify(value);
                    }
                    return (
                        <div className="flex gap-2" key={key}>
                            <span className="text-muted-foreground/30 shrink-0">{key}:</span>
                            <span className="text-foreground/50 break-all">{displayValue}</span>
                        </div>
                    );
                })}
            </div>

            {result !== undefined && (
                <>
                    <div className="my-2 border-t border-white/[0.04]" />
                    <pre
                        className={`leading-relaxed break-words whitespace-pre-wrap ${
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
        <pre className="text-foreground/70 font-mono text-xs leading-relaxed break-words whitespace-pre-wrap">
            {block.content}
        </pre>
    );
}

function ErrorBlockView({ block }: { block: { content: string } }) {
    return (
        <div className="flex items-start gap-2 rounded-lg border border-red-500/15 bg-red-500/[0.06] px-3 py-2">
            <IconAlertTriangle className="mt-0.5 shrink-0 text-red-400/60" size={14} />
            <pre className="font-mono text-[11px] break-words whitespace-pre-wrap text-red-400/70">
                {block.content}
            </pre>
        </div>
    );
}

function StatusBlockView({ block }: { block: { status: string; tool?: string } }) {
    const labels: Record<string, string> = {
        thinking: 'Thinking...',
        tool_calling:
            block.tool === undefined || block.tool === ''
                ? 'Executing...'
                : `Running ${block.tool}...`,
        responding: 'Responding...',
        done: 'Done',
    };

    if (block.status === 'done') {
        return null;
    }

    return (
        <div className="flex items-center gap-2 py-1">
            <div className="bg-foreground/30 h-2 w-2 animate-pulse rounded-full" />
            <span className="text-muted-foreground/40 text-[11px]">
                {(labels[block.status] ?? '') !== '' || block.status}
            </span>
        </div>
    );
}

function blockKey(block: ActivityBlock, i: number): string {
    if (block.type === 'tool_use') {
        return `tool-${block.id}`;
    }
    return `${block.type}-${i}`;
}

export function ActivityBlocksRenderer({ blocks }: { blocks: ActivityBlock[] }) {
    const resultMap = new Map<string, { content: string; isError: boolean }>();
    for (const block of blocks) {
        if (block.type === 'tool_result') {
            resultMap.set(block.id, { content: block.content, isError: block.isError });
        }
    }

    return (
        <div className="space-y-2">
            {blocks.map((block, i) => {
                if (block.type === 'tool_result') {
                    return null;
                }

                const key = blockKey(block, i);
                switch (block.type) {
                    case 'thinking': {
                        return <ThinkingBlockView block={block} key={key} />;
                    }
                    case 'tool_use': {
                        return (
                            <ToolUseBlockView
                                block={block}
                                key={key}
                                result={resultMap.get(block.id)}
                            />
                        );
                    }
                    case 'text': {
                        return <TextBlockView block={block} key={key} />;
                    }
                    case 'error': {
                        return <ErrorBlockView block={block} key={key} />;
                    }
                    case 'status': {
                        return <StatusBlockView block={block} key={key} />;
                    }
                    default: {
                        return null;
                    }
                }
            })}
        </div>
    );
}

export function ActivityMessageView({ message }: { message: ActivityMessage }) {
    const hasBlocks = message.blocks.length > 0;
    const hasOnlyText = message.blocks.every((b) => b.type === 'text');

    if (!hasBlocks || hasOnlyText) {
        const text = message.blocks
            .filter((b): b is { type: 'text'; content: string } => b.type === 'text')
            .map((b) => b.content)
            .join('');

        return (
            <pre className="font-mono text-xs leading-relaxed break-words whitespace-pre-wrap">
                {text || '...'}
            </pre>
        );
    }

    return (
        <div className="space-y-2">
            <ActivityBlocksRenderer blocks={message.blocks} />
            {message.cost !== undefined && (
                <div className="text-muted-foreground/20 flex items-center gap-3 pt-1 font-mono text-[9px]">
                    {message.duration !== undefined && (
                        <span>{(message.duration / 1000).toFixed(1)}s</span>
                    )}
                    <span>${message.cost.toFixed(4)}</span>
                </div>
            )}
        </div>
    );
}

'use client';

import {
    IconBook,
    IconBrain,
    IconChevronDown,
    IconChevronRight,
    IconFile,
    IconFolder,
    IconFolderOpen,
    IconNotebook,
    IconUser,
} from '@tabler/icons-react';
import { useState } from 'react';

import { apiGet, encPath } from '../../api/client';
import { textOf } from '../../api/fields';
import { inBackground } from '../../events/in-background';

const LAYER_ICONS: Record<string, typeof IconFolder> = {
    core: IconUser,
    journal: IconNotebook,
    knowledge: IconBook,
    playbooks: IconBook,
    skills: IconBrain,
};

const LAYER_ORDER = ['identity', 'skills', 'playbooks', 'journal'];

function rank(layer: string): number {
    const index = LAYER_ORDER.indexOf(layer);
    return index === -1 ? LAYER_ORDER.length : index;
}

function toggled(names: Set<string>, name: string): Set<string> {
    const next = new Set(names);
    if (next.has(name)) {
        next.delete(name);
    } else {
        next.add(name);
    }
    return next;
}

async function contentOf(agent: string, path: string): Promise<string> {
    try {
        return textOf(
            await apiGet<unknown>(`/api/agents/${encPath(agent)}/files/${path}`),
            'content',
        );
    } catch {
        return '⚠ Failed to load file content';
    }
}

function Chevron({ open, size }: { open: boolean; size: number }) {
    const Icon = open ? IconChevronDown : IconChevronRight;
    return <Icon className="text-muted-foreground/40 shrink-0" size={size} />;
}

interface MindFileProps {
    agent: string;
    layer: string;
    file: string;
}

function MindFile({ agent, layer, file }: MindFileProps) {
    const [open, setOpen] = useState(false);
    const [content, setContent] = useState<null | string>(null);

    const toggle = async () => {
        setOpen(!open);
        if (!open && content === null) {
            setContent(
                await contentOf(
                    agent,
                    file.endsWith('.md') ? `${layer}/${file}` : `${layer}/${file}.md`,
                ),
            );
        }
    };

    return (
        <div>
            <button
                className="flex w-full items-center gap-2.5 px-4 py-2 pl-10 text-left transition-colors hover:bg-white/[0.02]"
                onClick={inBackground(toggle)}
                type="button"
            >
                <Chevron open={open} size={12} />
                <IconFile className="text-muted-foreground/30 shrink-0" size={13} />
                <span className="text-foreground/60 font-mono text-[11px]">{file}</span>
            </button>
            {open && (
                <div className="border-t border-white/[0.03] bg-white/[0.01] px-4 py-3 pl-16">
                    {content === null ? (
                        <div className="text-muted-foreground/30 flex items-center gap-2 text-xs">
                            <div className="border-foreground/20 border-t-foreground/50 h-3 w-3 animate-spin rounded-full border-2" />
                            Loading...
                        </div>
                    ) : (
                        <pre className="text-foreground/50 max-h-96 overflow-x-auto overflow-y-auto font-mono text-[11px] leading-relaxed whitespace-pre-wrap">
                            {content === '' ? 'No content' : content}
                        </pre>
                    )}
                </div>
            )}
        </div>
    );
}

function Layer({
    agent,
    layer,
    files,
    open,
    onToggle,
}: {
    agent: string;
    layer: string;
    files: string[];
    open: boolean;
    onToggle: () => void;
}) {
    const LayerIcon = open ? IconFolderOpen : (LAYER_ICONS[layer] ?? IconFolder);
    return (
        <div className="glass-subtle overflow-hidden">
            <button
                className="flex w-full items-center gap-2.5 px-4 py-3 text-left transition-colors hover:bg-white/[0.02]"
                onClick={onToggle}
                type="button"
            >
                <Chevron open={open} size={14} />
                <LayerIcon className="text-foreground/50 shrink-0" size={16} />
                <span className="text-foreground/70 flex-1 font-mono text-xs">{layer}/</span>
                <span className="text-muted-foreground/30 font-mono text-[10px]">
                    {files.length} files
                </span>
            </button>
            {open && files.length > 0 && (
                <div className="border-t border-white/[0.04]">
                    {files.map((file) => (
                        <MindFile agent={agent} file={file} key={file} layer={layer} />
                    ))}
                </div>
            )}
        </div>
    );
}

/** The agent's mind on disk, layer by layer, each file opened in place. */
export function MindFiles({
    agent,
    mindTree,
}: {
    agent: string;
    mindTree: Record<string, string[]>;
}) {
    const [open, setOpen] = useState<Set<string>>(new Set());
    const layers = Object.keys(mindTree).toSorted((a, b) => rank(a) - rank(b));

    if (layers.length === 0) {
        return (
            <div className="py-12 text-center">
                <IconFolder className="text-muted-foreground/15 mx-auto mb-3" size={32} />
                <p className="text-muted-foreground/40 text-sm">No mind files found</p>
                <p className="text-muted-foreground/25 mt-1 font-mono text-xs">
                    Create files with: spwn agent dream {agent}
                </p>
            </div>
        );
    }
    return (
        <div className="space-y-2">
            {layers.map((layer) => (
                <Layer
                    agent={agent}
                    files={mindTree[layer] ?? []}
                    key={layer}
                    layer={layer}
                    onToggle={() => {
                        setOpen((previous) => toggled(previous, layer));
                    }}
                    open={open.has(layer)}
                />
            ))}
        </div>
    );
}

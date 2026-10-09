'use client';

import {
    IconChevronDown,
    IconChevronRight,
    IconFile,
    IconFolder,
    IconFolderOpen,
} from '@tabler/icons-react';
import { useState } from 'react';

import { formatModified } from './tree';
import type { TreeNode } from './tree';

interface FileTreeNodeProps {
    node: TreeNode;
    depth: number;
    onFileClick: (path: string) => void;
    selectedPath: null | string;
}

function FolderNode({ node, depth, onFileClick, selectedPath }: FileTreeNodeProps) {
    const [expanded, setExpanded] = useState(depth === 0);
    const Chevron = expanded ? IconChevronDown : IconChevronRight;
    const Folder = expanded ? IconFolderOpen : IconFolder;
    return (
        <div>
            <button
                className="flex w-full items-center gap-2 px-3 py-1.5 text-left transition-colors hover:bg-white/[0.03]"
                onClick={() => {
                    setExpanded(!expanded);
                }}
                style={{ paddingLeft: `${12 + depth * 16}px` }}
                type="button"
            >
                <Chevron className="text-muted-foreground/40 shrink-0" size={12} />
                <Folder className="text-foreground/50 shrink-0" size={14} />
                <span className="text-foreground/60 font-mono text-[11px]">{node.name}/</span>
                <span className="text-muted-foreground/25 ml-auto font-mono text-[9px]">
                    {node.children.length}
                </span>
            </button>
            {expanded && (
                <div>
                    {node.children.map((child) => (
                        <FileTreeNode
                            depth={depth + 1}
                            key={child.path}
                            node={child}
                            onFileClick={onFileClick}
                            selectedPath={selectedPath}
                        />
                    ))}
                </div>
            )}
        </div>
    );
}

/** One entry of the knowledge tree: a folder that unfolds, or a file that opens beside it. */
export function FileTreeNode(props: FileTreeNodeProps) {
    const { node, depth, onFileClick, selectedPath } = props;
    if (node.isDir) {
        return <FolderNode {...props} />;
    }
    const selected = selectedPath === node.path;
    return (
        <button
            className={`flex w-full items-center gap-2 px-3 py-1.5 text-left transition-colors ${selected ? 'border-l-2 border-blue-400/50 bg-white/[0.06]' : 'hover:bg-white/[0.03]'}`}
            onClick={() => {
                onFileClick(node.path);
            }}
            style={{ paddingLeft: `${12 + depth * 16}px` }}
            type="button"
        >
            <span className="w-3" />
            <IconFile className="text-muted-foreground/30 shrink-0" size={13} />
            <span
                className={`flex-1 font-mono text-[11px] ${selected ? 'text-foreground/80' : 'text-foreground/55'}`}
            >
                {node.name}
            </span>
            {node.file !== undefined && (
                <span className="text-muted-foreground/20 font-mono text-[9px]">
                    {formatModified(node.file.modified)}
                </span>
            )}
        </button>
    );
}

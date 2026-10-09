'use client';

import { IconBook2, IconFile } from '@tabler/icons-react';
import Link from 'next/link';
import { useCallback, useMemo, useState } from 'react';

import { apiGet } from '../api/client';
import { textOf } from '../api/fields';
import { inBackground } from '../events/in-background';
import { useBackgroundTask } from '../hooks/use-background-task';
import { FileTreeNode } from './knowledge/file-tree-node';
import { buildTree, formatSize } from './knowledge/tree';
import type { KnowledgeFile, TreeNode } from './knowledge/tree';

interface KnowledgeBrowserProps {
    worldId: string;
    searchQuery: string;
}

function useKnowledgeFiles(base: string) {
    const [files, setFiles] = useState<KnowledgeFile[]>([]);
    const [loading, setLoading] = useState(true);
    const load = useCallback(async () => {
        try {
            const listed = await apiGet<{ files?: KnowledgeFile[] }>(base);
            setFiles(listed.files ?? []);
        } catch {
            setFiles([]);
        }
        setLoading(false);
    }, [base]);
    useBackgroundTask(load);
    return { files, loading };
}

function useOpenedFile(base: string) {
    const [path, setPath] = useState<null | string>(null);
    const [content, setContent] = useState<null | string>(null);

    const open = async (next: string) => {
        if (path === next) {
            setPath(null);
            setContent(null);
            return;
        }
        setPath(next);
        setContent(null);
        try {
            setContent(textOf(await apiGet<unknown>(`${base}/${next}`), 'content'));
        } catch {
            setContent('⚠ Failed to load file');
        }
    };
    return { content, open, path };
}

function TreeSkeleton() {
    return (
        <div className="space-y-2 p-4">
            {[1, 2, 3, 4].map((index) => (
                <div className="flex items-center gap-2 px-3 py-1.5" key={index}>
                    <div className="h-3 w-3 animate-pulse rounded bg-white/[0.06]" />
                    <div
                        className="h-3 animate-pulse rounded bg-white/[0.06]"
                        style={{ width: `${40 + index * 15}%` }}
                    />
                </div>
            ))}
        </div>
    );
}

function NoKnowledge() {
    return (
        <div className="flex h-full flex-col items-center justify-center px-4 text-center">
            <IconBook2 className="text-muted-foreground/15 mb-3" size={28} />
            <p className="text-muted-foreground/35 text-xs">No knowledge files yet</p>
            <p className="text-muted-foreground/20 mt-1 max-w-[200px] text-[10px]">
                Talk to the Architect to start building your knowledge base
            </p>
            <Link
                className="mt-3 font-mono text-[10px] text-blue-400/50 transition-colors hover:text-blue-400/80"
                href="/architect"
            >
                Go to Architect →
            </Link>
        </div>
    );
}

function OpenedFile({
    path,
    size,
    content,
}: {
    path: string;
    size: number | undefined;
    content: null | string;
}) {
    return (
        <div className="flex-1 overflow-y-auto">
            <div className="flex items-center gap-2 border-b border-white/[0.04] bg-white/[0.02] px-4 py-2">
                <IconFile className="text-muted-foreground/30" size={13} />
                <span className="text-foreground/60 flex-1 font-mono text-[11px]">{path}</span>
                {size !== undefined && (
                    <span className="text-muted-foreground/20 font-mono text-[9px]">
                        {formatSize(size)}
                    </span>
                )}
            </div>
            <div className="p-4">
                {content === null ? (
                    <div className="text-muted-foreground/30 flex items-center gap-2 text-xs">
                        <div className="border-foreground/20 border-t-foreground/50 h-3 w-3 animate-spin rounded-full border-2" />
                        Loading...
                    </div>
                ) : (
                    <pre className="text-foreground/55 font-mono text-[11px] leading-relaxed whitespace-pre-wrap">
                        {content === '' ? 'No content' : content}
                    </pre>
                )}
            </div>
        </div>
    );
}

function Tree({
    tree,
    loading,
    selected,
    onOpen,
}: {
    tree: TreeNode[];
    loading: boolean;
    selected: null | string;
    onOpen: (path: string) => void;
}) {
    if (loading) {
        return <TreeSkeleton />;
    }
    if (tree.length === 0) {
        return <NoKnowledge />;
    }
    return (
        <div className="py-1">
            {tree.map((node) => (
                <FileTreeNode
                    depth={0}
                    key={node.path}
                    node={node}
                    onFileClick={onOpen}
                    selectedPath={selected}
                />
            ))}
        </div>
    );
}

/** A world's knowledge base: its files as a tree, filtered by the page's search, each opened beside it. */
export function KnowledgeBrowser({ worldId, searchQuery }: KnowledgeBrowserProps) {
    const base = `/api/worlds/${worldId}/knowledge`;
    const { files, loading } = useKnowledgeFiles(base);
    const opened = useOpenedFile(base);
    const tree = useMemo(() => {
        const query = searchQuery.trim().toLowerCase();
        return buildTree(
            query === '' ? files : files.filter((file) => file.path.toLowerCase().includes(query)),
        );
    }, [files, searchQuery]);

    return (
        <div className="glass-subtle h-[600px] overflow-hidden rounded-xl">
            <div className="flex items-center gap-2.5 border-b border-white/[0.06] px-4 py-3">
                <IconBook2 className="text-muted-foreground/40" size={16} />
                <h3 className="font-heading text-foreground/60 flex-1 text-xs tracking-wide">
                    Knowledge
                </h3>
                <span className="text-muted-foreground/25 rounded-full border border-white/[0.05] bg-white/[0.03] px-2 py-0.5 font-mono text-[9px]">
                    managed by architect
                </span>
            </div>
            <div className="flex h-[calc(100%-40px)]">
                <div
                    className={`overflow-y-auto border-r border-white/[0.05] ${opened.path === null ? 'w-full' : 'w-1/3 min-w-[200px]'}`}
                >
                    <Tree
                        loading={loading}
                        onOpen={inBackground(opened.open)}
                        selected={opened.path}
                        tree={tree}
                    />
                </div>
                {opened.path !== null && (
                    <OpenedFile
                        content={opened.content}
                        path={opened.path}
                        size={files.find((file) => file.path === opened.path)?.size}
                    />
                )}
            </div>
        </div>
    );
}

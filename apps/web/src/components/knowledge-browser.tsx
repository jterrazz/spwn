'use client';

import {
    IconBook2,
    IconChevronDown,
    IconChevronRight,
    IconFile,
    IconFolder,
    IconFolderOpen,
    IconSearch,
} from '@tabler/icons-react';
import Link from 'next/link';
import { useEffect, useMemo, useState } from 'react';

import { goApiUrl } from '@/api/client';

interface KnowledgeFile {
    path: string;
    size: number;
    modified: string;
}

interface TreeNode {
    name: string;
    path: string;
    isDir: boolean;
    children: TreeNode[];
    file?: KnowledgeFile;
}

// Sort: dirs first, then alphabetically (recurses into children).
function sortTreeNodes(nodes: TreeNode[]): TreeNode[] {
    nodes.sort((a, b) => {
        if (a.isDir && !b.isDir) {
            return -1;
        }
        if (!a.isDir && b.isDir) {
            return 1;
        }
        return a.name.localeCompare(b.name);
    });
    for (const n of nodes) {
        if (n.isDir) {
            sortTreeNodes(n.children);
        }
    }
    return nodes;
}

function buildTree(files: KnowledgeFile[]): TreeNode[] {
    const root: TreeNode[] = [];

    for (const file of files) {
        const parts = file.path.split('/');
        let current = root;

        for (const [i, part] of parts.entries()) {
            const isLast = i === parts.length - 1;
            const existing = current.find((n) => n.name === part);

            if (isLast) {
                if (existing) {
                    existing.file = file;
                } else {
                    current.push({ name: part, path: file.path, isDir: false, children: [], file });
                }
            } else if (existing) {
                current = existing.children;
            } else {
                const dir: TreeNode = {
                    name: part,
                    path: parts.slice(0, i + 1).join('/'),
                    isDir: true,
                    children: [],
                };
                current.push(dir);
                current = dir.children;
            }
        }
    }

    return sortTreeNodes(root);
}

function formatSize(bytes: number): string {
    if (bytes < 1024) {
        return `${bytes} B`;
    }
    return `${(bytes / 1024).toFixed(1)} KB`;
}

function formatModified(dateStr: string): string {
    try {
        const d = new Date(dateStr);
        const now = new Date();
        const diffMs = now.getTime() - d.getTime();
        const diffMins = Math.floor(diffMs / 60_000);
        if (diffMins < 1) {
            return 'just now';
        }
        if (diffMins < 60) {
            return `${diffMins}m ago`;
        }
        const diffHours = Math.floor(diffMins / 60);
        if (diffHours < 24) {
            return `${diffHours}h ago`;
        }
        const diffDays = Math.floor(diffHours / 24);
        if (diffDays < 30) {
            return `${diffDays}d ago`;
        }
        return d.toLocaleDateString();
    } catch {
        return '';
    }
}

function FileTreeNode({
    node,
    depth,
    onFileClick,
    selectedPath,
}: {
    node: TreeNode;
    depth: number;
    onFileClick: (path: string) => void;
    selectedPath: null | string;
}) {
    const [expanded, setExpanded] = useState(depth === 0);

    if (node.isDir) {
        return (
            <div>
                <button
                    className="flex w-full items-center gap-2 px-3 py-1.5 text-left transition-colors hover:bg-white/[0.03]"
                    onClick={() => setExpanded(!expanded)}
                    style={{ paddingLeft: `${12 + depth * 16}px` }}
                >
                    {expanded ? (
                        <IconChevronDown className="text-muted-foreground/40 shrink-0" size={12} />
                    ) : (
                        <IconChevronRight className="text-muted-foreground/40 shrink-0" size={12} />
                    )}
                    {expanded ? (
                        <IconFolderOpen className="text-foreground/50 shrink-0" size={14} />
                    ) : (
                        <IconFolder className="text-foreground/40 shrink-0" size={14} />
                    )}
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

    const isSelected = selectedPath === node.path;

    return (
        <button
            className={`flex w-full items-center gap-2 px-3 py-1.5 text-left transition-colors ${
                isSelected
                    ? 'border-l-2 border-blue-400/50 bg-white/[0.06]'
                    : 'hover:bg-white/[0.03]'
            }`}
            onClick={() => onFileClick(node.path)}
            style={{ paddingLeft: `${12 + depth * 16}px` }}
        >
            <span className="w-3" />
            <IconFile className="text-muted-foreground/30 shrink-0" size={13} />
            <span
                className={`flex-1 font-mono text-[11px] ${isSelected ? 'text-foreground/80' : 'text-foreground/55'}`}
            >
                {node.name}
            </span>
            {node.file && (
                <span className="text-muted-foreground/20 font-mono text-[9px]">
                    {formatModified(node.file.modified)}
                </span>
            )}
        </button>
    );
}

interface KnowledgeBrowserProps {
    compact?: boolean;
    worldId: string;
    /** When provided, the parent owns the search state and this component
     *  hides its own search input (e.g. when a PageHeader action bar
     *  hosts an ExpandingSearch). */
    searchQuery?: string;
    onSearchChange?: (value: string) => void;
}

export function KnowledgeBrowser({
    compact = false,
    worldId,
    searchQuery: externalSearch,
    onSearchChange,
}: KnowledgeBrowserProps) {
    const [files, setFiles] = useState<KnowledgeFile[]>([]);
    const [loading, setLoading] = useState(true);
    const [internalSearch, setInternalSearch] = useState('');
    const searchControlled = externalSearch !== undefined && onSearchChange !== undefined;
    const searchQuery = searchControlled ? externalSearch : internalSearch;
    const setSearchQuery = searchControlled ? onSearchChange : setInternalSearch;
    const [selectedPath, setSelectedPath] = useState<null | string>(null);
    const [fileContent, setFileContent] = useState<null | string>(null);
    const [contentLoading, setContentLoading] = useState(false);

    const knowledgeApiBase = `/api/worlds/${worldId}/knowledge`;

    useEffect(() => {
        const fetchFiles = async () => {
            try {
                const res = await fetch(goApiUrl(knowledgeApiBase));
                if (res.ok) {
                    const data = await res.json();
                    setFiles(data.files ?? []);
                }
            } catch {
                // Ignore
            } finally {
                setLoading(false);
            }
        };
        fetchFiles();
    }, [knowledgeApiBase]);

    const filteredFiles = useMemo(() => {
        if (!searchQuery.trim()) {
            return files;
        }
        const q = searchQuery.toLowerCase();
        return files.filter((f) => f.path.toLowerCase().includes(q));
    }, [files, searchQuery]);

    const tree = useMemo(() => buildTree(filteredFiles), [filteredFiles]);

    const handleFileClick = async (path: string) => {
        if (selectedPath === path) {
            setSelectedPath(null);
            setFileContent(null);
            return;
        }

        setSelectedPath(path);
        setContentLoading(true);
        setFileContent(null);

        try {
            const res = await fetch(goApiUrl(`${knowledgeApiBase}/${path}`));
            if (res.ok) {
                const data = await res.json();
                setFileContent(data.content ?? '');
            } else {
                setFileContent('⚠ Failed to load file');
            }
        } catch {
            setFileContent('⚠ Failed to connect to API');
        } finally {
            setContentLoading(false);
        }
    };

    const height = compact ? '400px' : '600px';

    return (
        <div className="glass-subtle overflow-hidden rounded-xl" style={{ height }}>
            {/* Header */}
            <div className="flex items-center gap-2.5 border-b border-white/[0.06] px-4 py-3">
                <IconBook2 className="text-muted-foreground/40" size={16} />
                <h3 className="font-heading text-foreground/60 flex-1 text-xs tracking-wide">
                    Knowledge
                </h3>
                <span className="text-muted-foreground/25 rounded-full border border-white/[0.05] bg-white/[0.03] px-2 py-0.5 font-mono text-[9px]">
                    managed by architect
                </span>
            </div>

            {/* Search - hidden when a parent controls searchQuery externally
          (e.g. an ExpandingSearch in the page header). */}
            {!searchControlled && (
                <div className="border-b border-white/[0.04] px-3 py-2">
                    <div className="relative">
                        <IconSearch
                            className="text-muted-foreground/25 absolute top-1/2 left-2.5 -translate-y-1/2"
                            size={13}
                        />
                        <input
                            className="text-foreground/70 placeholder:text-muted-foreground/25 w-full rounded-lg border border-white/[0.06] bg-white/[0.03] py-1.5 pr-3 pl-8 text-[11px] focus:border-white/[0.12] focus:outline-none"
                            onChange={(e) => setSearchQuery(e.target.value)}
                            placeholder="Search files..."
                            type="text"
                            value={searchQuery}
                        />
                    </div>
                </div>
            )}

            {/* Content */}
            <div
                className={`flex ${searchControlled ? 'h-[calc(100%-40px)]' : 'h-[calc(100%-88px)]'}`}
            >
                {/* File tree */}
                <div
                    className={`overflow-y-auto border-r border-white/[0.05] ${selectedPath ? 'w-1/3 min-w-[200px]' : 'w-full'}`}
                >
                    {loading && (
                        <div className="space-y-2 p-4">
                            {[1, 2, 3, 4].map((i) => (
                                <div className="flex items-center gap-2 px-3 py-1.5" key={i}>
                                    <div className="h-3 w-3 animate-pulse rounded bg-white/[0.06]" />
                                    <div
                                        className="h-3 animate-pulse rounded bg-white/[0.06]"
                                        style={{ width: `${40 + i * 15}%` }}
                                    />
                                </div>
                            ))}
                        </div>
                    )}
                    {!loading && tree.length === 0 && (
                        <div className="flex h-full flex-col items-center justify-center px-4 text-center">
                            <IconBook2 className="text-muted-foreground/15 mb-3" size={28} />
                            <p className="text-muted-foreground/35 text-xs">
                                No knowledge files yet
                            </p>
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
                    )}
                    {!loading && tree.length > 0 && (
                        <div className="py-1">
                            {tree.map((node) => (
                                <FileTreeNode
                                    depth={0}
                                    key={node.path}
                                    node={node}
                                    onFileClick={handleFileClick}
                                    selectedPath={selectedPath}
                                />
                            ))}
                        </div>
                    )}
                </div>

                {/* File content viewer */}
                {selectedPath && (
                    <div className="flex-1 overflow-y-auto">
                        <div className="flex items-center gap-2 border-b border-white/[0.04] bg-white/[0.02] px-4 py-2">
                            <IconFile className="text-muted-foreground/30" size={13} />
                            <span className="text-foreground/60 flex-1 font-mono text-[11px]">
                                {selectedPath}
                            </span>
                            {files.find((f) => f.path === selectedPath) && (
                                <span className="text-muted-foreground/20 font-mono text-[9px]">
                                    {formatSize(files.find((f) => f.path === selectedPath)!.size)}
                                </span>
                            )}
                        </div>
                        <div className="p-4">
                            {contentLoading ? (
                                <div className="text-muted-foreground/30 flex items-center gap-2 text-xs">
                                    <div className="border-foreground/20 border-t-foreground/50 h-3 w-3 animate-spin rounded-full border-2" />
                                    Loading...
                                </div>
                            ) : (
                                <pre className="text-foreground/55 font-mono text-[11px] leading-relaxed whitespace-pre-wrap">
                                    {fileContent ?? 'No content'}
                                </pre>
                            )}
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}

/** Inline card shown in chat when architect updates a knowledge file. */
export function KnowledgeUpdateCard({ path, description }: { path: string; description?: string }) {
    return (
        <div className="animate-in slide-in-from-bottom-2 fade-in mt-1.5 max-w-[80%] duration-300">
            <div className="overflow-hidden rounded-lg border border-indigo-500/20 bg-indigo-500/[0.06]">
                <div className="flex items-center gap-1.5 border-b border-indigo-500/15 bg-indigo-500/10 px-3 py-1.5">
                    <span className="text-[10px]">📘</span>
                    <span className="font-mono text-[10px] tracking-wider text-indigo-400/70 uppercase">
                        Knowledge Updated
                    </span>
                </div>
                <div className="px-3 py-2">
                    <p className="font-mono text-xs text-indigo-200/90">{path}</p>
                    {description && (
                        <p className="mt-1 text-[10px] text-indigo-400/40">{description}</p>
                    )}
                </div>
            </div>
        </div>
    );
}

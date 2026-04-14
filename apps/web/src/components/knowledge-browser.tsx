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

import { goApiUrl } from '@/lib/api-client';

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

        for (let i = 0; i < parts.length; i++) {
            const part = parts[i];
            const isLast = i === parts.length - 1;
            const existingIdx = current.findIndex((n) => n.name === part);

            if (isLast) {
                if (existingIdx !== -1) {
                    current[existingIdx].file = file;
                } else {
                    current.push({ name: part, path: file.path, isDir: false, children: [], file });
                }
            } else if (existingIdx !== -1) {
                current = current[existingIdx].children;
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
                    className="w-full flex items-center gap-2 px-3 py-1.5 text-left hover:bg-white/[0.03] transition-colors"
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
                    <span className="text-[11px] font-mono text-foreground/60">{node.name}/</span>
                    <span className="text-[9px] font-mono text-muted-foreground/25 ml-auto">
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
            className={`w-full flex items-center gap-2 px-3 py-1.5 text-left transition-colors ${
                isSelected
                    ? 'bg-white/[0.06] border-l-2 border-blue-400/50'
                    : 'hover:bg-white/[0.03]'
            }`}
            onClick={() => onFileClick(node.path)}
            style={{ paddingLeft: `${12 + depth * 16}px` }}
        >
            <span className="w-3" />
            <IconFile className="text-muted-foreground/30 shrink-0" size={13} />
            <span
                className={`text-[11px] font-mono flex-1 ${isSelected ? 'text-foreground/80' : 'text-foreground/55'}`}
            >
                {node.name}
            </span>
            {node.file && (
                <span className="text-[9px] font-mono text-muted-foreground/20">
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
        <div className="glass-subtle rounded-xl overflow-hidden" style={{ height }}>
            {/* Header */}
            <div className="flex items-center gap-2.5 px-4 py-3 border-b border-white/[0.06]">
                <IconBook2 className="text-muted-foreground/40" size={16} />
                <h3 className="text-xs font-heading tracking-wide text-foreground/60 flex-1">
                    Knowledge
                </h3>
                <span className="text-[9px] font-mono text-muted-foreground/25 px-2 py-0.5 rounded-full bg-white/[0.03] border border-white/[0.05]">
                    managed by architect
                </span>
            </div>

            {/* Search - hidden when a parent controls searchQuery externally
          (e.g. an ExpandingSearch in the page header). */}
            {!searchControlled && (
                <div className="px-3 py-2 border-b border-white/[0.04]">
                    <div className="relative">
                        <IconSearch
                            className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground/25"
                            size={13}
                        />
                        <input
                            className="w-full bg-white/[0.03] border border-white/[0.06] rounded-lg text-[11px] text-foreground/70 placeholder:text-muted-foreground/25 pl-8 pr-3 py-1.5 focus:outline-none focus:border-white/[0.12]"
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
                        <div className="p-4 space-y-2">
                            {[1, 2, 3, 4].map((i) => (
                                <div className="flex items-center gap-2 px-3 py-1.5" key={i}>
                                    <div className="w-3 h-3 rounded bg-white/[0.06] animate-pulse" />
                                    <div
                                        className="h-3 rounded bg-white/[0.06] animate-pulse"
                                        style={{ width: `${40 + i * 15}%` }}
                                    />
                                </div>
                            ))}
                        </div>
                    )}
                    {!loading && tree.length === 0 && (
                        <div className="flex flex-col items-center justify-center h-full text-center px-4">
                            <IconBook2 className="text-muted-foreground/15 mb-3" size={28} />
                            <p className="text-xs text-muted-foreground/35">
                                No knowledge files yet
                            </p>
                            <p className="text-[10px] text-muted-foreground/20 mt-1 max-w-[200px]">
                                Talk to the Architect to start building your knowledge base
                            </p>
                            <Link
                                className="mt-3 text-[10px] font-mono text-blue-400/50 hover:text-blue-400/80 transition-colors"
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
                        <div className="flex items-center gap-2 px-4 py-2 border-b border-white/[0.04] bg-white/[0.02]">
                            <IconFile className="text-muted-foreground/30" size={13} />
                            <span className="text-[11px] font-mono text-foreground/60 flex-1">
                                {selectedPath}
                            </span>
                            {files.find((f) => f.path === selectedPath) && (
                                <span className="text-[9px] font-mono text-muted-foreground/20">
                                    {formatSize(files.find((f) => f.path === selectedPath)!.size)}
                                </span>
                            )}
                        </div>
                        <div className="p-4">
                            {contentLoading ? (
                                <div className="flex items-center gap-2 text-muted-foreground/30 text-xs">
                                    <div className="w-3 h-3 border-2 border-foreground/20 border-t-foreground/50 rounded-full animate-spin" />
                                    Loading...
                                </div>
                            ) : (
                                <pre className="text-[11px] font-mono text-foreground/55 whitespace-pre-wrap leading-relaxed">
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
        <div className="max-w-[80%] mt-1.5 animate-in slide-in-from-bottom-2 fade-in duration-300">
            <div className="rounded-lg overflow-hidden border border-indigo-500/20 bg-indigo-500/[0.06]">
                <div className="flex items-center gap-1.5 px-3 py-1.5 bg-indigo-500/10 border-b border-indigo-500/15">
                    <span className="text-[10px]">📘</span>
                    <span className="text-[10px] font-mono uppercase tracking-wider text-indigo-400/70">
                        Knowledge Updated
                    </span>
                </div>
                <div className="px-3 py-2">
                    <p className="text-xs font-mono text-indigo-200/90">{path}</p>
                    {description && (
                        <p className="text-[10px] text-indigo-400/40 mt-1">{description}</p>
                    )}
                </div>
            </div>
        </div>
    );
}

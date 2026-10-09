export interface KnowledgeFile {
    path: string;
    size: number;
    modified: string;
}

export interface TreeNode {
    name: string;
    path: string;
    isDir: boolean;
    children: TreeNode[];
    file?: KnowledgeFile;
}

function byFoldersThenName(a: TreeNode, b: TreeNode): number {
    if (a.isDir !== b.isDir) {
        return a.isDir ? -1 : 1;
    }
    return a.name.localeCompare(b.name);
}

function sorted(nodes: TreeNode[]): TreeNode[] {
    return nodes
        .toSorted(byFoldersThenName)
        .map((node) => (node.isDir ? { ...node, children: sorted(node.children) } : node));
}

function folderIn(level: TreeNode[], name: string, path: string): TreeNode[] {
    const existing = level.find((node) => node.name === name);
    if (existing !== undefined) {
        return existing.children;
    }
    const folder: TreeNode = { children: [], isDir: true, name, path };
    level.push(folder);
    return folder.children;
}

/** The knowledge files as a tree of folders, folders first, each level by name. */
export function buildTree(files: KnowledgeFile[]): TreeNode[] {
    const root: TreeNode[] = [];
    for (const file of files) {
        const parts = file.path.split('/');
        const name = parts.pop() ?? file.path;
        let level = root;
        for (const [index, part] of parts.entries()) {
            level = folderIn(level, part, parts.slice(0, index + 1).join('/'));
        }
        const existing = level.find((node) => node.name === name);
        if (existing === undefined) {
            level.push({ children: [], file, isDir: false, name, path: file.path });
        } else {
            existing.file = file;
        }
    }
    return sorted(root);
}

export function formatSize(bytes: number): string {
    return bytes < 1024 ? `${bytes} B` : `${(bytes / 1024).toFixed(1)} KB`;
}

export function formatModified(stamp: string): string {
    const minutes = Math.floor((Date.now() - new Date(stamp).getTime()) / 60_000);
    if (Number.isNaN(minutes)) {
        return '';
    }
    if (minutes < 1) {
        return 'just now';
    }
    if (minutes < 60) {
        return `${minutes}m ago`;
    }
    const hours = Math.floor(minutes / 60);
    if (hours < 24) {
        return `${hours}h ago`;
    }
    const days = Math.floor(hours / 24);
    return days < 30 ? `${days}d ago` : new Date(stamp).toLocaleDateString();
}

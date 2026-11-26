export type Face = 'card' | 'dot' | 'open' | 'pill' | 'region';

export type CanvasNode = {
    id: string;
    depth: number;
    kind: string;
    parent: null | string;
    kids: readonly string[];
    label: string;
    marks: readonly boolean[];
    band: number;
};

export type CanvasGraph = {
    nodes: ReadonlyMap<string, CanvasNode>;
    roots: readonly string[];
};

export type Rect = { x: number; y: number; w: number; h: number };

export type Placed = Rect & { face: Face };

export function nodeOf(graph: CanvasGraph, id: string): CanvasNode {
    const node = graph.nodes.get(id);
    if (node === undefined) {
        throw new Error(`the graph holds no node ${id}`);
    }
    return node;
}

export function pathTo(graph: CanvasGraph, id: string, leafDepth: number): string[] {
    const path: string[] = [];
    let at: null | string = id;
    while (at !== null) {
        const node = nodeOf(graph, at);
        if (node.depth < leafDepth) {
            path.unshift(node.id);
        }
        at = node.parent;
    }
    return path;
}

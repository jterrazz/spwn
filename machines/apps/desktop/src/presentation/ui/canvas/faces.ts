import type { CanvasNode, Face } from './graph';

export function faceOf(node: CanvasNode, sel: readonly string[]): Face | null {
    const depth = sel.length;
    const level = node.depth;
    if (sel[level - 1] === node.id) {
        return 'open';
    }
    if (depth === 0) {
        if (level === 1) {
            return 'region';
        }
        return level === 2 ? 'card' : null;
    }
    if (level > 1 && sel[level - 2] !== node.parent) {
        return null;
    }
    if (level === depth + 1) {
        return 'card';
    }
    if (level === depth) {
        return 'pill';
    }
    return level < depth ? 'dot' : null;
}

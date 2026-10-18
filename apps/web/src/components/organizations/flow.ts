import type { Edge, Node } from '@xyflow/react';

import type { OrganizationRole } from '../../domain/model';

export interface RoleNodeData extends Record<string, unknown> {
    role: OrganizationRole;
    isFirst: boolean;
    isLast: boolean;
}

export const NODE_WIDTH = 220;
const NODE_GAP_X = 40;
const ROW_HEIGHT = 200;
const EDGE_STYLE = { stroke: 'rgba(255,255,255,0.08)', strokeWidth: 1.5 };

function byLevel(roles: OrganizationRole[]): Map<number, OrganizationRole[]> {
    const levels = new Map<number, OrganizationRole[]>();
    for (const role of roles.toSorted((left, right) => left.level - right.level)) {
        levels.set(role.level, [...(levels.get(role.level) ?? []), role]);
    }
    return levels;
}

function edge(source: string, target: string): Edge {
    return { id: `${source}->${target}`, source, style: EDGE_STYLE, target };
}

function declaredEdges(roles: OrganizationRole[]): Edge[] {
    return roles.flatMap((role) =>
        role.reports_to === undefined || role.reports_to === ''
            ? []
            : [edge(role.reports_to, role.name)],
    );
}

function levelEdges(rows: OrganizationRole[][]): Edge[] {
    return rows
        .slice(1)
        .flatMap((children, index) =>
            (rows[index] ?? []).flatMap((parent) =>
                children.map((child) => edge(parent.name, child.name)),
            ),
        );
}

/** The canvas height that shows every level of an organization. */
export function flowHeight(roles: OrganizationRole[]): number {
    return Math.max(300, new Set(roles.map((role) => role.level)).size * ROW_HEIGHT + 40);
}

/** An organization drawn as a chart: one row per level, centred, linked by `reports_to` or, when no role declares one, each level to the next. */
export function flowOf(roles: OrganizationRole[]): { nodes: Node<RoleNodeData>[]; edges: Edge[] } {
    const rows = [...byLevel(roles).values()];
    const nodes = rows.flatMap((row, rowIndex) => {
        const start = -((row.length - 1) * (NODE_WIDTH + NODE_GAP_X)) / 2;
        return row.map((role, column) => ({
            data: { isFirst: rowIndex === 0, isLast: rowIndex === rows.length - 1, role },
            id: role.name,
            position: { x: start + column * (NODE_WIDTH + NODE_GAP_X), y: rowIndex * ROW_HEIGHT },
            type: 'roleNode',
        }));
    });
    const declared = declaredEdges(rows.flat());
    return { edges: declared.length > 0 ? declared : levelEdges(rows), nodes };
}

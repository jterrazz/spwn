'use client';

import { IconPlus, IconTrash, IconX } from '@tabler/icons-react';
import { Handle, Position, ReactFlow } from '@xyflow/react';
import type { Edge, Node, NodeProps } from '@xyflow/react';
import { useCallback, useEffect, useMemo, useState } from 'react';

import '@xyflow/react/dist/style.css';
import { apiDelete, apiGet, apiPost } from '@/api/client';
import { ActionButton } from '@/components/action-button';
import { KeyValue, SectionHeader, Separator, SubLabel } from '@/components/ds';
import { Page } from '@/components/page';
import { PageHeader } from '@/components/page-header';
import { Skeleton } from '@/components/ui/skeleton';
import type { Organization, OrganizationRole } from '@/domain/model';
import { usePageTitle } from '@/hooks/use-page-title';

// ── Role colors by level ──────────────────────────────────────────
const LEVEL_COLORS: Record<number, { border: string; text: string; bg: string; glow: string }> = {
    0: {
        border: 'border-amber-500/30',
        text: 'text-amber-300',
        bg: 'bg-amber-500/10',
        glow: 'shadow-[0_0_20px_rgba(245,158,11,0.08)]',
    },
    1: {
        border: 'border-purple-500/25',
        text: 'text-purple-300',
        bg: 'bg-purple-500/10',
        glow: 'shadow-[0_0_16px_rgba(168,85,247,0.06)]',
    },
    2: { border: 'border-blue-500/20', text: 'text-blue-300', bg: 'bg-blue-500/10', glow: '' },
};
function roleColor(level: number) {
    return (
        LEVEL_COLORS[level] ?? {
            border: 'border-white/[0.1]',
            text: 'text-foreground/60',
            bg: 'bg-white/[0.04]',
            glow: '',
        }
    );
}

// ── Custom React Flow node ────────────────────────────────────────
type RoleNodeData = {
    role: OrganizationRole;
    isFirst: boolean;
    isLast: boolean;
};

function RoleNodeComponent({ data }: NodeProps<Node<RoleNodeData>>) {
    const { role, isFirst, isLast } = data;
    const c = roleColor(role.level);

    return (
        <div
            className={`border ${c.border} min-w-[220px] rounded-xl bg-[#0a0a0c]/90 px-6 py-4 text-center backdrop-blur-sm ${c.glow} hover:border-opacity-60 transition-shadow`}
        >
            {/* Handles for edges */}
            {!isFirst && (
                <Handle
                    className="!h-1.5 !w-1.5 !border-0 !bg-white/[0.15]"
                    position={Position.Top}
                    type="target"
                />
            )}
            {!isLast && (
                <Handle
                    className="!h-1.5 !w-1.5 !border-0 !bg-white/[0.15]"
                    position={Position.Bottom}
                    type="source"
                />
            )}

            {/* Role name */}
            <p className={`font-mono text-sm font-bold tracking-[0.06em] uppercase ${c.text}`}>
                {role.name}
            </p>

            {/* Level + constraints */}
            <div className="mt-2 flex items-center justify-center gap-1.5">
                <span
                    className={`rounded-full px-2 py-0.5 font-mono text-[9px] ${c.bg} ${c.text} border ${c.border}`}
                >
                    Level {role.level}
                </span>
                {role.max_per_world != null && role.max_per_world > 0 && (
                    <span className="text-muted-foreground/40 rounded-full border border-white/[0.06] bg-white/[0.03] px-2 py-0.5 font-mono text-[9px]">
                        max {role.max_per_world}
                    </span>
                )}
            </div>

            {/* Permissions */}
            {role.permissions && role.permissions.length > 0 && (
                <div className="mt-3 flex flex-wrap items-center justify-center gap-1">
                    {role.permissions.map((p) => (
                        <span
                            className="text-muted-foreground/40 rounded border border-white/[0.05] bg-white/[0.03] px-1.5 py-0.5 font-mono text-[8px]"
                            key={p}
                        >
                            {p}
                        </span>
                    ))}
                </div>
            )}

            {/* Relationships */}
            {(role.reports_to || (role.can_command && role.can_command.length > 0)) && (
                <div className="mt-2.5 space-y-0.5">
                    {role.reports_to && (
                        <p className="text-muted-foreground/25 font-mono text-[8px]">
                            reports to{' '}
                            <span className="text-muted-foreground/40">{role.reports_to}</span>
                        </p>
                    )}
                    {role.can_command && role.can_command.length > 0 && (
                        <p className="text-muted-foreground/25 font-mono text-[8px]">
                            commands{' '}
                            <span className="text-muted-foreground/40">
                                {role.can_command.join(', ')}
                            </span>
                        </p>
                    )}
                </div>
            )}
        </div>
    );
}

const nodeTypes = { roleNode: RoleNodeComponent };

// ── Build React Flow nodes + edges from organization roles ───────────
const NODE_WIDTH = 220;
const NODE_HEIGHT_BASE = 120;
const NODE_GAP_Y = 80;

function buildFlowElements(roles: OrganizationRole[]): {
    nodes: Node<RoleNodeData>[];
    edges: Edge[];
} {
    const sorted = [...roles].sort((a, b) => a.level - b.level);
    const nodes: Node<RoleNodeData>[] = [];
    const edges: Edge[] = [];

    // Group by level for horizontal spreading
    const levels = new Map<number, OrganizationRole[]>();
    for (const r of sorted) {
        const list = levels.get(r.level) ?? [];
        list.push(r);
        levels.set(r.level, list);
    }

    const sortedLevels = [...levels.keys()].sort((a, b) => a - b);
    let y = 0;

    for (const [li, level] of sortedLevels.entries()) {
        const rolesAtLevel = levels.get(level) ?? [];
        const totalWidth = rolesAtLevel.length * NODE_WIDTH + (rolesAtLevel.length - 1) * 40;
        const startX = -totalWidth / 2 + NODE_WIDTH / 2;

        for (const [ri, role] of rolesAtLevel.entries()) {
            nodes.push({
                id: role.name,
                type: 'roleNode',
                position: { x: startX + ri * (NODE_WIDTH + 40), y },
                data: {
                    role,
                    isFirst: li === 0,
                    isLast: li === sortedLevels.length - 1,
                },
            });
        }
        y += NODE_HEIGHT_BASE + NODE_GAP_Y;
    }

    // Edges: connect based on reports_to
    for (const role of sorted) {
        if (role.reports_to) {
            edges.push({
                id: `${role.reports_to}->${role.name}`,
                source: role.reports_to,
                target: role.name,
                style: { stroke: 'rgba(255,255,255,0.08)', strokeWidth: 1.5 },
                animated: false,
            });
        }
    }

    // If no reports_to defined, connect by level order
    if (edges.length === 0 && sortedLevels.length > 1) {
        for (let i = 0; i < sortedLevels.length - 1; i++) {
            const parentLevel = sortedLevels[i];
            const childLevel = sortedLevels[i + 1];
            if (parentLevel === undefined || childLevel === undefined) {
                continue;
            }
            const parents = levels.get(parentLevel) ?? [];
            const children = levels.get(childLevel) ?? [];
            for (const p of parents) {
                for (const c of children) {
                    edges.push({
                        id: `${p.name}->${c.name}`,
                        source: p.name,
                        target: c.name,
                        style: { stroke: 'rgba(255,255,255,0.08)', strokeWidth: 1.5 },
                    });
                }
            }
        }
    }

    return { nodes, edges };
}

// ── Organization Flow Visualization ──────────────────────────────────
function OrganizationFlow({ organization }: { organization: Organization }) {
    const { nodes, edges } = useMemo(
        () => buildFlowElements(organization.roles),
        [organization.roles],
    );

    // Compute height based on number of levels
    const levels = new Set(organization.roles.map((r) => r.level));
    const height = Math.max(300, levels.size * (NODE_HEIGHT_BASE + NODE_GAP_Y) + 40);

    return (
        <div className="overflow-hidden" style={{ height }}>
            <ReactFlow
                edges={edges}
                elementsSelectable={false}
                fitView
                fitViewOptions={{ padding: 0.3 }}
                maxZoom={1.5}
                minZoom={0.5}
                nodes={nodes}
                nodesConnectable={false}
                nodesDraggable={false}
                nodeTypes={nodeTypes}
                panOnDrag={false}
                proOptions={{ hideAttribution: true }}
                style={{ background: 'transparent' }}
                zoomOnDoubleClick={false}
                zoomOnPinch={false}
                zoomOnScroll={false}
            />
        </div>
    );
}

// ── Create Organization Dialog ───────────────────────────────────────

interface RoleDraft {
    id: string;
    name: string;
    level: number;
    reports_to: string;
    can_command: string;
    permissions: string;
}

let roleDraftCounter = 0;
function emptyRoleDraft(): RoleDraft {
    roleDraftCounter += 1;
    return {
        id: `draft-${roleDraftCounter}`,
        name: '',
        level: 0,
        reports_to: '',
        can_command: '',
        permissions: '',
    };
}

function CreateOrganizationDialog({
    onClose,
    onComplete,
}: {
    onClose: () => void;
    onComplete: () => void;
}) {
    const [name, setName] = useState('');
    const [description, setDescription] = useState('');
    const [roles, setRoles] = useState<RoleDraft[]>([emptyRoleDraft()]);
    const [creating, setCreating] = useState(false);
    const [errorMessage, setErrorMessage] = useState('');

    const updateRole = (idx: number, patch: Partial<RoleDraft>) => {
        setRoles((prev) => prev.map((r, i) => (i === idx ? { ...r, ...patch } : r)));
    };

    const removeRole = (idx: number) => {
        setRoles((prev) => prev.filter((_, i) => i !== idx));
    };

    const handleCreate = async () => {
        if (!name.trim()) {
            setErrorMessage('Name is required');
            return;
        }
        if (roles.every((r) => !r.name.trim())) {
            setErrorMessage('At least one role with a name is required');
            return;
        }
        setCreating(true);
        setErrorMessage('');
        const slug = name
            .trim()
            .toLowerCase()
            .replaceAll(/[^a-z0-9]+/g, '-')
            .replaceAll(/^-|-$/g, '');
        const body: Organization = {
            slug,
            name: name.trim(),
            description: description.trim() || undefined,
            roles: roles
                .filter((r) => r.name.trim())
                .map((r) => ({
                    name: r.name.trim(),
                    level: r.level,
                    reports_to: r.reports_to.trim() || undefined,
                    can_command: r.can_command.trim()
                        ? r.can_command
                              .split(',')
                              .map((s) => s.trim())
                              .filter(Boolean)
                        : undefined,
                    permissions: r.permissions.trim()
                        ? r.permissions
                              .split(',')
                              .map((s) => s.trim())
                              .filter(Boolean)
                        : undefined,
                })),
        };
        try {
            await apiPost('/api/organizations', body);
            onComplete();
            onClose();
        } catch (error) {
            setErrorMessage(
                error instanceof Error ? error.message : 'Failed to create organization',
            );
            setCreating(false);
        }
    };

    const roleNames = roles.map((r) => r.name.trim()).filter(Boolean);

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
            <div className="bg-popover/95 relative z-10 mx-4 flex max-h-[85vh] w-full max-w-lg flex-col rounded-2xl border border-white/[0.08] shadow-2xl backdrop-blur-md">
                <div className="flex shrink-0 items-center justify-between px-6 pt-6 pb-4">
                    <div>
                        <h2 className="font-heading text-foreground/90 text-lg">
                            New Organization
                        </h2>
                        <p className="text-muted-foreground/40 mt-0.5 text-[11px]">
                            Define a role structure for organizing agents
                        </p>
                    </div>
                    <button
                        className="text-muted-foreground/40 hover:text-foreground/60 transition-colors"
                        onClick={onClose}
                    >
                        <IconX size={18} />
                    </button>
                </div>

                <div className="min-h-0 space-y-4 overflow-y-auto px-6 pb-6">
                    <div>
                        <label className="text-muted-foreground/40 mb-1.5 block text-[10px] tracking-widest uppercase">
                            Name <span className="text-red-400/60">*</span>
                        </label>
                        <input
                            className="text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.15] focus:outline-none"
                            onChange={(e) => setName(e.target.value)}
                            placeholder="Military Chain"
                            value={name}
                        />
                    </div>
                    <div>
                        <label className="text-muted-foreground/40 mb-1.5 block text-[10px] tracking-widest uppercase">
                            Description
                        </label>
                        <input
                            className="text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded-lg border border-white/[0.08] bg-white/[0.03] px-3 py-2.5 text-sm transition-colors focus:border-white/[0.15] focus:outline-none"
                            onChange={(e) => setDescription(e.target.value)}
                            placeholder="A strict top-down command structure"
                            value={description}
                        />
                    </div>

                    <div>
                        <label className="text-muted-foreground/40 mb-2 block text-[10px] tracking-widest uppercase">
                            Roles
                        </label>
                        <div className="space-y-3">
                            {roles.map((role, idx) => (
                                <div
                                    className="space-y-2 rounded-lg border border-white/[0.06] bg-white/[0.01] p-3"
                                    key={role.id}
                                >
                                    <div className="flex items-center gap-2">
                                        <span className="text-muted-foreground/30 font-mono text-[9px]">
                                            #{idx + 1}
                                        </span>
                                        <div className="flex-1" />
                                        {roles.length > 1 && (
                                            <button
                                                className="text-muted-foreground/30 transition-colors hover:text-red-400/70"
                                                onClick={() => removeRole(idx)}
                                            >
                                                <IconX size={14} />
                                            </button>
                                        )}
                                    </div>
                                    <div className="grid grid-cols-[1fr_60px] gap-2">
                                        <input
                                            className="text-foreground/80 placeholder:text-muted-foreground/25 rounded border border-white/[0.08] bg-white/[0.03] px-2.5 py-1.5 font-mono text-xs transition-colors focus:border-white/[0.15] focus:outline-none"
                                            onChange={(e) =>
                                                updateRole(idx, { name: e.target.value })
                                            }
                                            placeholder="Role name"
                                            value={role.name}
                                        />
                                        <input
                                            className="text-foreground/80 placeholder:text-muted-foreground/25 rounded border border-white/[0.08] bg-white/[0.03] px-2.5 py-1.5 text-center font-mono text-xs transition-colors focus:border-white/[0.15] focus:outline-none"
                                            onChange={(e) =>
                                                updateRole(idx, {
                                                    level: Number.parseInt(e.target.value) || 0,
                                                })
                                            }
                                            placeholder="Lvl"
                                            type="number"
                                            value={role.level}
                                        />
                                    </div>
                                    <div className="grid grid-cols-2 gap-2">
                                        <div>
                                            <span className="text-muted-foreground/25 mb-0.5 block font-mono text-[8px] uppercase">
                                                Reports to
                                            </span>
                                            <select
                                                className="text-foreground/80 w-full rounded border border-white/[0.08] bg-white/[0.03] px-2 py-1.5 font-mono text-xs transition-colors focus:border-white/[0.15] focus:outline-none"
                                                onChange={(e) =>
                                                    updateRole(idx, { reports_to: e.target.value })
                                                }
                                                value={role.reports_to}
                                            >
                                                <option value="">None</option>
                                                {roleNames
                                                    .filter((n) => n !== role.name.trim())
                                                    .map((n) => (
                                                        <option key={n} value={n}>
                                                            {n}
                                                        </option>
                                                    ))}
                                            </select>
                                        </div>
                                        <div>
                                            <span className="text-muted-foreground/25 mb-0.5 block font-mono text-[8px] uppercase">
                                                Can command
                                            </span>
                                            <input
                                                className="text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded border border-white/[0.08] bg-white/[0.03] px-2 py-1.5 font-mono text-xs transition-colors focus:border-white/[0.15] focus:outline-none"
                                                onChange={(e) =>
                                                    updateRole(idx, { can_command: e.target.value })
                                                }
                                                placeholder="role1, role2"
                                                value={role.can_command}
                                            />
                                        </div>
                                    </div>
                                    <div>
                                        <span className="text-muted-foreground/25 mb-0.5 block font-mono text-[8px] uppercase">
                                            Permissions
                                        </span>
                                        <input
                                            className="text-foreground/80 placeholder:text-muted-foreground/25 w-full rounded border border-white/[0.08] bg-white/[0.03] px-2 py-1.5 font-mono text-xs transition-colors focus:border-white/[0.15] focus:outline-none"
                                            onChange={(e) =>
                                                updateRole(idx, { permissions: e.target.value })
                                            }
                                            placeholder="delegate, review, execute"
                                            value={role.permissions}
                                        />
                                    </div>
                                </div>
                            ))}
                        </div>
                        <button
                            className="text-muted-foreground/50 hover:text-foreground/70 mt-2 flex items-center gap-1.5 font-mono text-[10px] transition-colors"
                            onClick={() => setRoles((prev) => [...prev, emptyRoleDraft()])}
                        >
                            <IconPlus size={12} /> Add Role
                        </button>
                    </div>

                    {errorMessage && <p className="text-xs text-red-400/80">{errorMessage}</p>}

                    <div className="flex justify-end gap-3 pt-2">
                        <button
                            className="text-muted-foreground/60 hover:text-foreground/80 rounded-lg px-4 py-2 text-sm transition-colors hover:bg-white/[0.04] disabled:opacity-50"
                            disabled={creating}
                            onClick={onClose}
                        >
                            Cancel
                        </button>
                        <button
                            className="flex items-center gap-2 rounded-lg border border-emerald-500/20 bg-emerald-500/20 px-4 py-2 text-sm text-emerald-300 transition-colors hover:bg-emerald-500/30 disabled:opacity-50"
                            disabled={creating}
                            onClick={handleCreate}
                        >
                            {creating ? (
                                <>
                                    <div className="h-3 w-3 animate-spin rounded-full border-2 border-emerald-300/40 border-t-emerald-300" />
                                    Creating...
                                </>
                            ) : (
                                'Create'
                            )}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
}

// ── Delete Confirmation Dialog ────────────────────────────────────
function DeleteOrganizationDialog({
    organization,
    onClose,
    onComplete,
}: {
    organization: Organization;
    onClose: () => void;
    onComplete: () => void;
}) {
    const [deleting, setDeleting] = useState(false);
    const [errorMessage, setErrorMessage] = useState('');

    const handleDelete = async () => {
        setDeleting(true);
        setErrorMessage('');
        try {
            await apiDelete(`/api/organizations/${organization.slug}`);
            onComplete();
            onClose();
        } catch (error) {
            setErrorMessage(
                error instanceof Error ? error.message : 'Failed to delete organization',
            );
            setDeleting(false);
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
            <div className="bg-popover/95 relative z-10 mx-4 w-full max-w-sm rounded-2xl border border-white/[0.08] p-6 shadow-2xl backdrop-blur-md">
                <h3 className="font-heading text-foreground/90 mb-2 text-lg">
                    Delete Organization
                </h3>
                <p className="text-muted-foreground/50 mb-6 text-sm">
                    Are you sure you want to delete{' '}
                    <span className="text-foreground/70 font-mono">{organization.name}</span>?
                </p>
                {errorMessage && <p className="mb-3 text-xs text-red-400/80">{errorMessage}</p>}
                <div className="flex justify-end gap-3">
                    <button
                        className="text-muted-foreground/60 hover:text-foreground/80 rounded-lg px-4 py-2 text-sm transition-colors hover:bg-white/[0.04] disabled:opacity-50"
                        disabled={deleting}
                        onClick={onClose}
                    >
                        Cancel
                    </button>
                    <button
                        className="flex items-center gap-2 rounded-lg border border-red-500/20 bg-red-500/20 px-4 py-2 text-sm text-red-300 transition-colors hover:bg-red-500/30 disabled:opacity-50"
                        disabled={deleting}
                        onClick={handleDelete}
                    >
                        {deleting ? (
                            <>
                                <div className="h-3 w-3 animate-spin rounded-full border-2 border-red-300/40 border-t-red-300" />
                                Deleting...
                            </>
                        ) : (
                            <>
                                <IconTrash size={14} />
                                Delete
                            </>
                        )}
                    </button>
                </div>
            </div>
        </div>
    );
}

// ── Main Page ─────────────────────────────────────────────────────

export default function OrganizationsPage() {
    usePageTitle('Organizations');

    const [organizations, setOrganizations] = useState<Organization[]>([]);
    const [loading, setLoading] = useState(true);
    const [showCreate, setShowCreate] = useState(false);
    const [deleteTarget, setDeleteTarget] = useState<null | Organization>(null);

    const fetchOrganizations = useCallback(async () => {
        try {
            const data = await apiGet<Organization[]>('/api/organizations').catch(
                () => [] as Organization[],
            );
            setOrganizations(data ?? []);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchOrganizations();
    }, [fetchOrganizations]);

    return (
        <Page>
            <PageHeader
                actions={
                    <ActionButton
                        compact
                        icon={<IconPlus size={18} stroke={2.4} />}
                        label="New Organization"
                        onClick={() => setShowCreate(true)}
                    />
                }
                description="Define team structures and roles for multi-agent worlds."
                title="Organizations"
            />

            {loading && (
                <div className="space-y-6">
                    <Skeleton className="h-[300px] w-full rounded-xl" />
                </div>
            )}
            {!loading && organizations.length === 0 && (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                    <p className="text-muted-foreground/50 text-sm">
                        No organizations defined yet.
                    </p>
                    <button
                        className="text-muted-foreground/40 hover:text-foreground/60 mt-3 font-mono text-xs underline underline-offset-2 transition-colors"
                        onClick={() => setShowCreate(true)}
                    >
                        Create your first organization
                    </button>
                </div>
            )}
            {!loading && organizations.length > 0 && (
                <div className="space-y-10">
                    {organizations.map((h, idx) => (
                        <div key={h.slug}>
                            <div className="mb-6 flex items-center gap-3">
                                <SectionHeader className="mb-0 flex-1">{h.name}</SectionHeader>
                                {h.slug !== 'default' && (
                                    <button
                                        className="text-muted-foreground/20 p-1 transition-colors hover:text-red-400/70"
                                        onClick={() => setDeleteTarget(h)}
                                        title="Delete organization"
                                    >
                                        <IconTrash size={14} />
                                    </button>
                                )}
                            </div>

                            <div className="flex flex-col gap-8 lg:flex-row">
                                {/* React Flow visualization */}
                                <div className="min-w-0 flex-1">
                                    <OrganizationFlow organization={h} />
                                </div>

                                {/* Metadata sidebar */}
                                <div className="shrink-0 space-y-2 lg:w-52">
                                    <SubLabel>Details</SubLabel>
                                    <KeyValue label="Slug" value={h.slug} />
                                    {h.description && (
                                        <KeyValue label="Description" value={h.description} />
                                    )}
                                    <KeyValue label="Roles" value={h.roles.length} />
                                    <KeyValue
                                        label="Max Depth"
                                        value={Math.max(...h.roles.map((r) => r.level))}
                                    />
                                </div>
                            </div>

                            {idx < organizations.length - 1 && <Separator className="mt-10" />}
                        </div>
                    ))}
                </div>
            )}

            {showCreate && (
                <CreateOrganizationDialog
                    onClose={() => setShowCreate(false)}
                    onComplete={fetchOrganizations}
                />
            )}
            {deleteTarget && (
                <DeleteOrganizationDialog
                    onClose={() => setDeleteTarget(null)}
                    onComplete={fetchOrganizations}
                    organization={deleteTarget}
                />
            )}
        </Page>
    );
}

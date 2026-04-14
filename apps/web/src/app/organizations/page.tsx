'use client';

import { IconPlus, IconTrash, IconX } from '@tabler/icons-react';
import { type Edge, Handle, type Node, type NodeProps, Position, ReactFlow } from '@xyflow/react';
import { useCallback, useEffect, useMemo, useState } from 'react';

import '@xyflow/react/dist/style.css';
import { ActionButton } from '@/components/action-button';
import { KeyValue, SectionHeader, Separator, SubLabel } from '@/components/ds';
import { Page } from '@/components/page';
import { PageHeader } from '@/components/page-header';
import { Skeleton } from '@/components/ui/skeleton';
import { usePageTitle } from '@/hooks/use-page-title';
import { apiDelete, apiGet, apiPost } from '@/lib/api-client';
import type { Organization, OrganizationRole } from '@/lib/types';

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
            className={`border ${c.border} bg-[#0a0a0c]/90 backdrop-blur-sm rounded-xl px-6 py-4 min-w-[220px] text-center ${c.glow} transition-shadow hover:border-opacity-60`}
        >
            {/* Handles for edges */}
            {!isFirst && (
                <Handle
                    className="!bg-white/[0.15] !border-0 !w-1.5 !h-1.5"
                    position={Position.Top}
                    type="target"
                />
            )}
            {!isLast && (
                <Handle
                    className="!bg-white/[0.15] !border-0 !w-1.5 !h-1.5"
                    position={Position.Bottom}
                    type="source"
                />
            )}

            {/* Role name */}
            <p className={`text-sm font-mono font-bold uppercase tracking-[0.06em] ${c.text}`}>
                {role.name}
            </p>

            {/* Level + constraints */}
            <div className="flex items-center justify-center gap-1.5 mt-2">
                <span
                    className={`px-2 py-0.5 text-[9px] font-mono rounded-full ${c.bg} ${c.text} border ${c.border}`}
                >
                    Level {role.level}
                </span>
                {role.max_per_world != null && role.max_per_world > 0 && (
                    <span className="px-2 py-0.5 text-[9px] font-mono rounded-full bg-white/[0.03] text-muted-foreground/40 border border-white/[0.06]">
                        max {role.max_per_world}
                    </span>
                )}
            </div>

            {/* Permissions */}
            {role.permissions && role.permissions.length > 0 && (
                <div className="flex flex-wrap items-center justify-center gap-1 mt-3">
                    {role.permissions.map((p) => (
                        <span
                            className="px-1.5 py-0.5 text-[8px] font-mono bg-white/[0.03] border border-white/[0.05] rounded text-muted-foreground/40"
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
                        <p className="text-[8px] font-mono text-muted-foreground/25">
                            reports to{' '}
                            <span className="text-muted-foreground/40">{role.reports_to}</span>
                        </p>
                    )}
                    {role.can_command && role.can_command.length > 0 && (
                        <p className="text-[8px] font-mono text-muted-foreground/25">
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

    for (let li = 0; li < sortedLevels.length; li++) {
        const level = sortedLevels[li];
        const rolesAtLevel = levels.get(level)!;
        const totalWidth = rolesAtLevel.length * NODE_WIDTH + (rolesAtLevel.length - 1) * 40;
        const startX = -totalWidth / 2 + NODE_WIDTH / 2;

        for (let ri = 0; ri < rolesAtLevel.length; ri++) {
            const role = rolesAtLevel[ri];
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
            const parents = levels.get(sortedLevels[i])!;
            const children = levels.get(sortedLevels[i + 1])!;
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
    name: string;
    level: number;
    reports_to: string;
    can_command: string;
    permissions: string;
}

function emptyRoleDraft(): RoleDraft {
    return { name: '', level: 0, reports_to: '', can_command: '', permissions: '' };
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
    const [error, setError] = useState('');

    const updateRole = (idx: number, patch: Partial<RoleDraft>) => {
        setRoles((prev) => prev.map((r, i) => (i === idx ? { ...r, ...patch } : r)));
    };

    const removeRole = (idx: number) => {
        setRoles((prev) => prev.filter((_, i) => i !== idx));
    };

    const handleCreate = async () => {
        if (!name.trim()) {
            setError('Name is required');
            return;
        }
        if (roles.every((r) => !r.name.trim())) {
            setError('At least one role with a name is required');
            return;
        }
        setCreating(true);
        setError('');
        const slug = name
            .trim()
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-|-$/g, '');
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
            setError(error instanceof Error ? error.message : 'Failed to create organization');
            setCreating(false);
        }
    };

    const roleNames = roles.map((r) => r.name.trim()).filter(Boolean);

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
            <div className="relative z-10 w-full max-w-lg mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl max-h-[85vh] flex flex-col">
                <div className="px-6 pt-6 pb-4 flex items-center justify-between shrink-0">
                    <div>
                        <h2 className="text-lg font-heading text-foreground/90">
                            New Organization
                        </h2>
                        <p className="text-[11px] text-muted-foreground/40 mt-0.5">
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

                <div className="px-6 pb-6 space-y-4 overflow-y-auto min-h-0">
                    <div>
                        <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
                            Name <span className="text-red-400/60">*</span>
                        </label>
                        <input
                            className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                            onChange={(e) => setName(e.target.value)}
                            placeholder="Military Chain"
                            value={name}
                        />
                    </div>
                    <div>
                        <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
                            Description
                        </label>
                        <input
                            className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                            onChange={(e) => setDescription(e.target.value)}
                            placeholder="A strict top-down command structure"
                            value={description}
                        />
                    </div>

                    <div>
                        <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-2">
                            Roles
                        </label>
                        <div className="space-y-3">
                            {roles.map((role, idx) => (
                                <div
                                    className="border border-white/[0.06] rounded-lg p-3 bg-white/[0.01] space-y-2"
                                    key={idx}
                                >
                                    <div className="flex items-center gap-2">
                                        <span className="text-[9px] font-mono text-muted-foreground/30">
                                            #{idx + 1}
                                        </span>
                                        <div className="flex-1" />
                                        {roles.length > 1 && (
                                            <button
                                                className="text-muted-foreground/30 hover:text-red-400/70 transition-colors"
                                                onClick={() => removeRole(idx)}
                                            >
                                                <IconX size={14} />
                                            </button>
                                        )}
                                    </div>
                                    <div className="grid grid-cols-[1fr_60px] gap-2">
                                        <input
                                            className="bg-white/[0.03] border border-white/[0.08] rounded px-2.5 py-1.5 text-xs font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                                            onChange={(e) =>
                                                updateRole(idx, { name: e.target.value })
                                            }
                                            placeholder="Role name"
                                            value={role.name}
                                        />
                                        <input
                                            className="bg-white/[0.03] border border-white/[0.08] rounded px-2.5 py-1.5 text-xs font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors text-center"
                                            onChange={(e) =>
                                                updateRole(idx, {
                                                    level: parseInt(e.target.value) || 0,
                                                })
                                            }
                                            placeholder="Lvl"
                                            type="number"
                                            value={role.level}
                                        />
                                    </div>
                                    <div className="grid grid-cols-2 gap-2">
                                        <div>
                                            <span className="text-[8px] font-mono text-muted-foreground/25 uppercase block mb-0.5">
                                                Reports to
                                            </span>
                                            <select
                                                className="w-full bg-white/[0.03] border border-white/[0.08] rounded px-2 py-1.5 text-xs font-mono text-foreground/80 focus:outline-none focus:border-white/[0.15] transition-colors"
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
                                            <span className="text-[8px] font-mono text-muted-foreground/25 uppercase block mb-0.5">
                                                Can command
                                            </span>
                                            <input
                                                className="w-full bg-white/[0.03] border border-white/[0.08] rounded px-2 py-1.5 text-xs font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                                                onChange={(e) =>
                                                    updateRole(idx, { can_command: e.target.value })
                                                }
                                                placeholder="role1, role2"
                                                value={role.can_command}
                                            />
                                        </div>
                                    </div>
                                    <div>
                                        <span className="text-[8px] font-mono text-muted-foreground/25 uppercase block mb-0.5">
                                            Permissions
                                        </span>
                                        <input
                                            className="w-full bg-white/[0.03] border border-white/[0.08] rounded px-2 py-1.5 text-xs font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
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
                            className="mt-2 flex items-center gap-1.5 text-[10px] font-mono text-muted-foreground/50 hover:text-foreground/70 transition-colors"
                            onClick={() => setRoles((prev) => [...prev, emptyRoleDraft()])}
                        >
                            <IconPlus size={12} /> Add Role
                        </button>
                    </div>

                    {error && <p className="text-xs text-red-400/80">{error}</p>}

                    <div className="flex gap-3 justify-end pt-2">
                        <button
                            className="px-4 py-2 rounded-lg text-sm text-muted-foreground/60 hover:text-foreground/80 hover:bg-white/[0.04] transition-colors disabled:opacity-50"
                            disabled={creating}
                            onClick={onClose}
                        >
                            Cancel
                        </button>
                        <button
                            className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30 border border-emerald-500/20 transition-colors disabled:opacity-50"
                            disabled={creating}
                            onClick={handleCreate}
                        >
                            {creating ? (
                                <>
                                    <div className="w-3 h-3 border-2 border-emerald-300/40 border-t-emerald-300 rounded-full animate-spin" />
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
    const [error, setError] = useState('');

    const handleDelete = async () => {
        setDeleting(true);
        setError('');
        try {
            await apiDelete(`/api/organizations/${organization.slug}`);
            onComplete();
            onClose();
        } catch (error) {
            setError(error instanceof Error ? error.message : 'Failed to delete organization');
            setDeleting(false);
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
            <div className="relative z-10 w-full max-w-sm mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl p-6">
                <h3 className="text-lg font-heading text-foreground/90 mb-2">
                    Delete Organization
                </h3>
                <p className="text-sm text-muted-foreground/50 mb-6">
                    Are you sure you want to delete{' '}
                    <span className="font-mono text-foreground/70">{organization.name}</span>?
                </p>
                {error && <p className="text-xs text-red-400/80 mb-3">{error}</p>}
                <div className="flex gap-3 justify-end">
                    <button
                        className="px-4 py-2 rounded-lg text-sm text-muted-foreground/60 hover:text-foreground/80 hover:bg-white/[0.04] transition-colors disabled:opacity-50"
                        disabled={deleting}
                        onClick={onClose}
                    >
                        Cancel
                    </button>
                    <button
                        className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm bg-red-500/20 text-red-300 hover:bg-red-500/30 border border-red-500/20 transition-colors disabled:opacity-50"
                        disabled={deleting}
                        onClick={handleDelete}
                    >
                        {deleting ? (
                            <>
                                <div className="w-3 h-3 border-2 border-red-300/40 border-t-red-300 rounded-full animate-spin" />
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

            {loading ? (
                <div className="space-y-6">
                    <Skeleton className="h-[300px] w-full rounded-xl" />
                </div>
            ) : organizations.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                    <p className="text-sm text-muted-foreground/50">
                        No organizations defined yet.
                    </p>
                    <button
                        className="mt-3 text-xs font-mono text-muted-foreground/40 hover:text-foreground/60 transition-colors underline underline-offset-2"
                        onClick={() => setShowCreate(true)}
                    >
                        Create your first organization
                    </button>
                </div>
            ) : (
                <div className="space-y-10">
                    {organizations.map((h, idx) => (
                        <div key={h.slug}>
                            <div className="flex items-center gap-3 mb-6">
                                <SectionHeader className="flex-1 mb-0">{h.name}</SectionHeader>
                                {h.slug !== 'default' && (
                                    <button
                                        className="text-muted-foreground/20 hover:text-red-400/70 transition-colors p-1"
                                        onClick={() => setDeleteTarget(h)}
                                        title="Delete organization"
                                    >
                                        <IconTrash size={14} />
                                    </button>
                                )}
                            </div>

                            <div className="flex flex-col lg:flex-row gap-8">
                                {/* React Flow visualization */}
                                <div className="flex-1 min-w-0">
                                    <OrganizationFlow organization={h} />
                                </div>

                                {/* Metadata sidebar */}
                                <div className="lg:w-52 shrink-0 space-y-2">
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

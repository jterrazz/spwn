"use client";

import { useCallback, useEffect, useState } from "react";
import { IconPlus, IconX, IconTrash } from "@tabler/icons-react";
import { Page } from "@/components/page";
import { PageHeader } from "@/components/page-header";
import { ActionButton } from "@/components/action-button";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet, apiPost, apiDelete } from "@/lib/api-client";
import { usePageTitle } from "@/hooks/use-page-title";
import { SectionHeader, SubLabel, KeyValue, Separator } from "@/components/ds";
import type { Hierarchy, HierarchyRole } from "@/lib/types";

// ── Role node color by level ───────────────────────────────────────
function roleColorByLevel(level: number): { border: string; text: string; bg: string } {
  if (level === 0) return { border: "border-amber-500/20", text: "text-amber-300", bg: "bg-amber-500/10" };
  if (level === 1) return { border: "border-blue-500/20", text: "text-blue-300", bg: "bg-blue-500/10" };
  return { border: "border-white/[0.12]", text: "text-foreground/70", bg: "bg-white/[0.04]" };
}

// ── Role Node ──────────────────────────────────────────────────────
function RoleNode({ role }: { role: HierarchyRole }) {
  const color = roleColorByLevel(role.level);
  return (
    <div className="flex flex-col items-center">
      <div className={`border ${color.border} bg-white/[0.02] rounded-lg px-5 py-3 min-w-[200px] text-center`}>
        <p className={`text-sm font-mono font-bold uppercase tracking-wide ${color.text}`}>
          {role.name}
        </p>
        <div className="flex items-center justify-center gap-2 mt-1.5">
          <span className={`px-1.5 py-0.5 text-[9px] font-mono rounded ${color.bg} ${color.text} border ${color.border}`}>
            Level {role.level}
          </span>
          {role.max_per_world != null && (
            <span className="px-1.5 py-0.5 text-[9px] font-mono rounded bg-white/[0.04] text-muted-foreground/50 border border-white/[0.06]">
              max {role.max_per_world}/world
            </span>
          )}
        </div>
        {role.permissions && role.permissions.length > 0 && (
          <div className="flex flex-wrap items-center justify-center gap-1 mt-2">
            {role.permissions.map((p) => (
              <span key={p} className="px-1.5 py-0.5 text-[9px] font-mono bg-white/[0.04] border border-white/[0.06] rounded text-muted-foreground/50">
                {p}
              </span>
            ))}
          </div>
        )}
        {role.reports_to && (
          <p className="text-[9px] font-mono text-muted-foreground/30 mt-1.5">
            reports to <span className="text-muted-foreground/50">{role.reports_to}</span>
          </p>
        )}
        {role.can_command && role.can_command.length > 0 && (
          <p className="text-[9px] font-mono text-muted-foreground/30 mt-0.5">
            commands <span className="text-muted-foreground/50">{role.can_command.join(", ")}</span>
          </p>
        )}
      </div>
    </div>
  );
}

// ── Connection line between nodes ──────────────────────────────────
function ConnectionLine() {
  return <div className="w-px h-6 bg-white/[0.1] mx-auto" />;
}

// ── Visual hierarchy tree ──────────────────────────────────────────
function HierarchyTree({ roles }: { roles: HierarchyRole[] }) {
  const sorted = [...roles].sort((a, b) => a.level - b.level);
  return (
    <div className="flex flex-col items-center py-4">
      {sorted.map((role, i) => (
        <div key={role.name} className="flex flex-col items-center">
          {i > 0 && <ConnectionLine />}
          <RoleNode role={role} />
        </div>
      ))}
    </div>
  );
}

// ── Create Hierarchy Dialog ────────────────────────────────────────

interface RoleDraft {
  name: string;
  level: number;
  reports_to: string;
  can_command: string;
  permissions: string;
}

function emptyRoleDraft(): RoleDraft {
  return { name: "", level: 0, reports_to: "", can_command: "", permissions: "" };
}

function CreateHierarchyDialog({ onClose, onComplete }: { onClose: () => void; onComplete: () => void }) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [roles, setRoles] = useState<RoleDraft[]>([emptyRoleDraft()]);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState("");

  const updateRole = (idx: number, patch: Partial<RoleDraft>) => {
    setRoles((prev) => prev.map((r, i) => (i === idx ? { ...r, ...patch } : r)));
  };

  const removeRole = (idx: number) => {
    setRoles((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleCreate = async () => {
    if (!name.trim()) {
      setError("Name is required");
      return;
    }
    if (roles.length === 0 || roles.every((r) => !r.name.trim())) {
      setError("At least one role with a name is required");
      return;
    }
    setCreating(true);
    setError("");
    const slug = name.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
    const body: Hierarchy = {
      slug,
      name: name.trim(),
      description: description.trim() || undefined,
      roles: roles
        .filter((r) => r.name.trim())
        .map((r) => ({
          name: r.name.trim(),
          level: r.level,
          reports_to: r.reports_to.trim() || undefined,
          can_command: r.can_command.trim() ? r.can_command.split(",").map((s) => s.trim()).filter(Boolean) : undefined,
          permissions: r.permissions.trim() ? r.permissions.split(",").map((s) => s.trim()).filter(Boolean) : undefined,
        })),
    };
    try {
      await apiPost("/api/hierarchies", body);
      onComplete();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create hierarchy");
      setCreating(false);
    }
  };

  const roleNames = roles.map((r) => r.name.trim()).filter(Boolean);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />

      {/* Dialog */}
      <div className="relative z-10 w-full max-w-lg mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl max-h-[85vh] flex flex-col">
        {/* Header */}
        <div className="px-6 pt-6 pb-4 flex items-center justify-between shrink-0">
          <div>
            <h2 className="text-lg font-heading text-foreground/90">New Hierarchy</h2>
            <p className="text-[11px] text-muted-foreground/40 mt-0.5">Define a role structure for organizing agents</p>
          </div>
          <button
            onClick={onClose}
            className="text-muted-foreground/40 hover:text-foreground/60 transition-colors"
          >
            <IconX size={18} />
          </button>
        </div>

        {/* Scrollable form */}
        <div className="px-6 pb-6 space-y-4 overflow-y-auto min-h-0">
          {/* Name */}
          <div>
            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
              Name <span className="text-red-400/60">*</span>
            </label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Military Chain"
              className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
            />
          </div>

          {/* Description */}
          <div>
            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-1.5">
              Description <span className="text-muted-foreground/25 normal-case tracking-normal">(optional)</span>
            </label>
            <input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="A strict top-down command structure"
              className="w-full bg-white/[0.03] border border-white/[0.08] rounded-lg px-3 py-2.5 text-sm text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
            />
          </div>

          {/* Roles */}
          <div>
            <label className="text-[10px] uppercase tracking-widest text-muted-foreground/40 block mb-2">
              Roles
            </label>
            <div className="space-y-3">
              {roles.map((role, idx) => (
                <div key={idx} className="border border-white/[0.06] rounded-lg p-3 bg-white/[0.01] space-y-2">
                  <div className="flex items-center gap-2">
                    <span className="text-[9px] font-mono text-muted-foreground/30 shrink-0">#{idx + 1}</span>
                    <div className="flex-1" />
                    {roles.length > 1 && (
                      <button
                        onClick={() => removeRole(idx)}
                        className="text-muted-foreground/30 hover:text-red-400/70 transition-colors"
                      >
                        <IconX size={14} />
                      </button>
                    )}
                  </div>
                  <div className="grid grid-cols-[1fr_60px] gap-2">
                    <input
                      value={role.name}
                      onChange={(e) => updateRole(idx, { name: e.target.value })}
                      placeholder="Role name"
                      className="bg-white/[0.03] border border-white/[0.08] rounded px-2.5 py-1.5 text-xs font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                    />
                    <input
                      type="number"
                      value={role.level}
                      onChange={(e) => updateRole(idx, { level: parseInt(e.target.value) || 0 })}
                      placeholder="Lvl"
                      className="bg-white/[0.03] border border-white/[0.08] rounded px-2.5 py-1.5 text-xs font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors text-center"
                    />
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <span className="text-[8px] font-mono text-muted-foreground/25 uppercase block mb-0.5">Reports to</span>
                      <select
                        value={role.reports_to}
                        onChange={(e) => updateRole(idx, { reports_to: e.target.value })}
                        className="w-full bg-white/[0.03] border border-white/[0.08] rounded px-2 py-1.5 text-xs font-mono text-foreground/80 focus:outline-none focus:border-white/[0.15] transition-colors"
                      >
                        <option value="">None</option>
                        {roleNames
                          .filter((n) => n !== role.name.trim())
                          .map((n) => (
                            <option key={n} value={n}>{n}</option>
                          ))}
                      </select>
                    </div>
                    <div>
                      <span className="text-[8px] font-mono text-muted-foreground/25 uppercase block mb-0.5">Can command</span>
                      <input
                        value={role.can_command}
                        onChange={(e) => updateRole(idx, { can_command: e.target.value })}
                        placeholder="role1, role2"
                        className="w-full bg-white/[0.03] border border-white/[0.08] rounded px-2 py-1.5 text-xs font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                      />
                    </div>
                  </div>
                  <div>
                    <span className="text-[8px] font-mono text-muted-foreground/25 uppercase block mb-0.5">Permissions</span>
                    <input
                      value={role.permissions}
                      onChange={(e) => updateRole(idx, { permissions: e.target.value })}
                      placeholder="delegate, review, execute"
                      className="w-full bg-white/[0.03] border border-white/[0.08] rounded px-2 py-1.5 text-xs font-mono text-foreground/80 placeholder:text-muted-foreground/25 focus:outline-none focus:border-white/[0.15] transition-colors"
                    />
                  </div>
                </div>
              ))}
            </div>
            <button
              onClick={() => setRoles((prev) => [...prev, emptyRoleDraft()])}
              className="mt-2 flex items-center gap-1.5 text-[10px] font-mono text-muted-foreground/50 hover:text-foreground/70 transition-colors"
            >
              <IconPlus size={12} /> Add Role
            </button>
          </div>

          {/* Error */}
          {error && <p className="text-xs text-red-400/80">{error}</p>}

          {/* Actions */}
          <div className="flex gap-3 justify-end pt-2">
            <button
              onClick={onClose}
              disabled={creating}
              className="px-4 py-2 rounded-lg text-sm text-muted-foreground/60 hover:text-foreground/80 hover:bg-white/[0.04] transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              onClick={handleCreate}
              disabled={creating}
              className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30 border border-emerald-500/20 transition-colors disabled:opacity-50"
            >
              {creating ? (
                <>
                  <div className="w-3 h-3 border-2 border-emerald-300/40 border-t-emerald-300 rounded-full animate-spin" />
                  Creating...
                </>
              ) : (
                "Create"
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Delete Confirmation Dialog ─────────────────────────────────────
function DeleteHierarchyDialog({
  hierarchy,
  onClose,
  onComplete,
}: {
  hierarchy: Hierarchy;
  onClose: () => void;
  onComplete: () => void;
}) {
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState("");

  const handleDelete = async () => {
    setDeleting(true);
    setError("");
    try {
      await apiDelete(`/api/hierarchies/${hierarchy.slug}`);
      onComplete();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete hierarchy");
      setDeleting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative z-10 w-full max-w-sm mx-4 rounded-2xl bg-popover/95 backdrop-blur-md border border-white/[0.08] shadow-2xl p-6">
        <h3 className="text-lg font-heading text-foreground/90 mb-2">Delete Hierarchy</h3>
        <p className="text-sm text-muted-foreground/50 mb-6">
          Are you sure you want to delete <span className="font-mono text-foreground/70">{hierarchy.name}</span>? This cannot be undone.
        </p>
        {error && <p className="text-xs text-red-400/80 mb-3">{error}</p>}
        <div className="flex gap-3 justify-end">
          <button
            onClick={onClose}
            disabled={deleting}
            className="px-4 py-2 rounded-lg text-sm text-muted-foreground/60 hover:text-foreground/80 hover:bg-white/[0.04] transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleDelete}
            disabled={deleting}
            className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm bg-red-500/20 text-red-300 hover:bg-red-500/30 border border-red-500/20 transition-colors disabled:opacity-50"
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

// ── Main Page ──────────────────────────────────────────────────────

export default function HierarchiesPage() {
  usePageTitle("Hierarchies");

  const [hierarchies, setHierarchies] = useState<Hierarchy[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Hierarchy | null>(null);

  const fetchHierarchies = useCallback(async () => {
    try {
      const data = await apiGet<Hierarchy[]>("/api/hierarchies").catch(() => [] as Hierarchy[]);
      setHierarchies(data ?? []);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchHierarchies();
  }, [fetchHierarchies]);

  return (
    <Page>
      <PageHeader
        title="Hierarchies"
        description="Role structures for organizing agents within worlds."
        actions={
          <ActionButton
            compact
            onClick={() => setShowCreate(true)}
            label="New Hierarchy"
            icon={<IconPlus size={18} stroke={2.4} />}
          />
        }
      />

      {loading ? (
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-14 w-full rounded-xl" />
          ))}
        </div>
      ) : hierarchies.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <p className="text-sm text-muted-foreground/50">
            No hierarchies defined yet.
          </p>
          <button
            onClick={() => setShowCreate(true)}
            className="mt-3 text-xs font-mono text-muted-foreground/40 hover:text-foreground/60 transition-colors underline underline-offset-2"
          >
            Create your first hierarchy
          </button>
        </div>
      ) : (
        <div className="space-y-8">
          {hierarchies.map((h, idx) => (
            <div key={h.slug}>
              <div className="flex items-center gap-3">
                <SectionHeader className="flex-1 mb-0">{h.name}</SectionHeader>
                {h.slug !== "default" && (
                  <button
                    onClick={() => setDeleteTarget(h)}
                    className="text-muted-foreground/20 hover:text-red-400/70 transition-colors p-1"
                    title="Delete hierarchy"
                  >
                    <IconTrash size={14} />
                  </button>
                )}
              </div>

              {/* Visual tree */}
              <HierarchyTree roles={h.roles} />

              {/* Metadata */}
              <div className="max-w-[300px] mx-auto space-y-1.5 mt-2">
                <KeyValue label="Slug" value={h.slug} />
                {h.description && <KeyValue label="Description" value={h.description} />}
                <KeyValue label="Roles" value={h.roles.length} />
              </div>

              {idx < hierarchies.length - 1 && <Separator className="mt-8" />}
            </div>
          ))}
        </div>
      )}

      {/* Create dialog */}
      {showCreate && (
        <CreateHierarchyDialog
          onClose={() => setShowCreate(false)}
          onComplete={fetchHierarchies}
        />
      )}

      {/* Delete confirmation dialog */}
      {deleteTarget && (
        <DeleteHierarchyDialog
          hierarchy={deleteTarget}
          onClose={() => setDeleteTarget(null)}
          onComplete={fetchHierarchies}
        />
      )}
    </Page>
  );
}

"use client";

import { useCallback, useEffect, useState } from "react";
import { IconPlus } from "@tabler/icons-react";
import { Page } from "@/components/page";
import { PageHeader } from "@/components/page-header";
import { ActionButton } from "@/components/action-button";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet } from "@/lib/api-client";
import { usePageTitle } from "@/hooks/use-page-title";
import { DataTable, SectionHeader, Separator } from "@/components/ds";
import { ROLE_BADGE } from "@/lib/status";
import type { Hierarchy, HierarchyRole } from "@/lib/types";

export default function HierarchiesPage() {
  usePageTitle("Hierarchies");

  const [hierarchies, setHierarchies] = useState<Hierarchy[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedSlug, setExpandedSlug] = useState<string | null>(null);

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

  const selectedHierarchy = expandedSlug
    ? hierarchies.find((h) => h.slug === expandedSlug)
    : null;

  return (
    <Page>
      <PageHeader
        title="Hierarchies"
        description="Role structures for organizing agents within worlds."
        actions={
          <ActionButton
            compact
            onClick={() => {}}
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
        </div>
      ) : (
        <>
          <DataTable<Hierarchy>
            rows={hierarchies}
            rowKey={(h) => h.slug}
            columns={[
              {
                key: "name",
                label: "Name",
                width: "1fr",
                render: (h) => (
                  <button
                    onClick={() => setExpandedSlug(expandedSlug === h.slug ? null : h.slug)}
                    className="text-[13px] font-mono text-foreground/85 truncate hover:underline underline-offset-2 text-left"
                  >
                    {h.name}
                  </button>
                ),
              },
              {
                key: "slug",
                label: "Slug",
                width: "100px",
                render: (h) => (
                  <span className="text-[11px] font-mono text-muted-foreground/50">{h.slug}</span>
                ),
              },
              {
                key: "roles",
                label: "Roles",
                width: "1fr",
                render: (h) => (
                  <span className="text-[11px] font-mono text-muted-foreground/50">
                    {h.roles.map((r) => r.name).join(", ")}
                  </span>
                ),
              },
              {
                key: "description",
                label: "Description",
                width: "1fr",
                render: (h) => (
                  <span className="text-[11px] font-mono text-muted-foreground/40 truncate">
                    {h.description || "—"}
                  </span>
                ),
              },
            ]}
          />

          {selectedHierarchy && (
            <>
              <Separator />
              <SectionHeader>{selectedHierarchy.name} — Roles</SectionHeader>
              <DataTable<HierarchyRole>
                rows={selectedHierarchy.roles}
                rowKey={(r) => r.name}
                columns={[
                  {
                    key: "name",
                    label: "Role",
                    width: "1fr",
                    render: (r) => {
                      const badge = ROLE_BADGE[r.name] ?? ROLE_BADGE.default;
                      return (
                        <span className={`px-1.5 py-0.5 rounded text-[9px] font-mono uppercase tracking-wider border ${badge}`}>
                          {r.name}
                        </span>
                      );
                    },
                  },
                  {
                    key: "level",
                    label: "Level",
                    width: "60px",
                    render: (r) => (
                      <span className="text-[11px] font-mono text-muted-foreground/50">{r.level}</span>
                    ),
                  },
                  {
                    key: "reports_to",
                    label: "Reports To",
                    width: "100px",
                    render: (r) => (
                      <span className="text-[11px] font-mono text-muted-foreground/50">
                        {r.reports_to || "—"}
                      </span>
                    ),
                  },
                  {
                    key: "can_command",
                    label: "Can Command",
                    width: "1fr",
                    render: (r) => (
                      <span className="text-[11px] font-mono text-muted-foreground/50">
                        {r.can_command?.join(", ") || "—"}
                      </span>
                    ),
                  },
                  {
                    key: "permissions",
                    label: "Permissions",
                    width: "1fr",
                    render: (r) => (
                      <span className="text-[11px] font-mono text-muted-foreground/40">
                        {r.permissions?.join(", ") || "—"}
                      </span>
                    ),
                  },
                ]}
              />
            </>
          )}
        </>
      )}
    </Page>
  );
}

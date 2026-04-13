"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { PageHeader } from "@/components/page-header";
import { Page } from "@/components/page";
import { usePageTitle } from "@/hooks/use-page-title";
import {
  IconTerminal2,
  IconBrandNodejs,
  IconBrandPython,
  IconHammer,
  IconBrandGit,
  IconBrandDocker,
  IconBrain,
  IconRocket,
  IconSearch,
  IconClock,
  IconPackage,
  IconChevronRight,
  IconBookFilled,
} from "@tabler/icons-react";
import { TOOLS, KIND_META, toolSlug, type ToolDef, type ToolKind, type ToolStatus } from "@/lib/tools-catalog";

// ── Icon map ────────────────────────────────────────────────────────────

const TOOL_ICONS: Record<string, React.ReactNode> = {
  "@spwn/unix": <IconTerminal2 size={18} />,
  "@spwn/node": <IconBrandNodejs size={18} />,
  "@spwn/python": <IconBrandPython size={18} />,
  "@spwn/build": <IconHammer size={18} />,
  "@spwn/claude-code": <IconBrain size={18} />,
  "@spwn/codex": <IconBrain size={18} />,
  "@spwn/aider": <IconBrain size={18} />,
  "@spwn/git": <IconBrandGit size={18} />,
  "@spwn/docker-cli": <IconBrandDocker size={18} />,
  "@spwn/qmd": <IconSearch size={18} />,
  "@spwn/cli": <IconRocket size={18} />,
  "@spwn/architect": <IconPackage size={18} />,
};

// ── Components ──────────────────────────────────────────────────────────

function KindBadge({ kind }: { kind: ToolKind }) {
  const { label, color } = KIND_META[kind];
  return (
    <span className={`text-[9px] font-mono uppercase tracking-wider px-1.5 py-0.5 rounded-full border ${color}`}>
      {label}
    </span>
  );
}

function StatusBadge({ status }: { status: ToolStatus }) {
  if (status === "available") return null;
  return (
    <span className="flex items-center gap-1 text-[9px] font-mono text-muted-foreground/30">
      <IconClock size={10} />
      Planned
    </span>
  );
}

function ToolCard({ tool, onClick }: { tool: ToolDef; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={`group text-left w-full rounded-xl border px-5 py-4 transition-all duration-200 ${
        tool.status === "planned"
          ? "bg-white/[0.01] border-white/[0.04] opacity-50"
          : "bg-white/[0.03] border-white/[0.07] hover:border-white/[0.12] hover:bg-white/[0.05]"
      }`}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-lg bg-white/[0.05] border border-white/[0.08] flex items-center justify-center text-muted-foreground/40 group-hover:text-foreground/60 transition-colors">
            {TOOL_ICONS[tool.name] ?? <IconPackage size={18} />}
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="text-sm font-mono font-medium text-foreground/80">{tool.name}</span>
              <KindBadge kind={tool.kind} />
            </div>
            <p className="text-[11px] text-muted-foreground/40 mt-0.5">{tool.description}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <StatusBadge status={tool.status} />
          <IconChevronRight size={14} className="text-muted-foreground/20 group-hover:text-muted-foreground/40 transition-colors" />
        </div>
      </div>

      <div className="space-y-1.5 pl-12">
        <div className="flex items-baseline gap-2">
          <span className="text-[10px] text-muted-foreground/25 w-16 shrink-0">provides</span>
          <span className="text-[11px] text-foreground/50 font-mono">{tool.provides}</span>
        </div>
        {tool.deps.length > 0 && (
          <div className="flex items-center gap-2">
            <span className="text-[10px] text-muted-foreground/25 w-16 shrink-0">depends</span>
            <div className="flex gap-1">
              {tool.deps.map((d) => (
                <span key={d} className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-white/[0.04] border border-white/[0.06] text-muted-foreground/35">
                  {d}
                </span>
              ))}
            </div>
          </div>
        )}
        {tool.skills.length > 0 && (
          <div className="flex items-center gap-2">
            <span className="text-[10px] text-muted-foreground/25 w-16 shrink-0">skills</span>
            <div className="flex gap-1">
              {tool.skills.map((s) => (
                <span key={s.name} className="flex items-center gap-1 text-[10px] font-mono px-1.5 py-0.5 rounded bg-purple-500/8 border border-purple-500/15 text-purple-400/50">
                  <IconBookFilled size={8} />
                  {s.name}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>
    </button>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────

type FilterKind = "all" | ToolKind;

export default function ToolsPage() {
  usePageTitle("Tools");
  const router = useRouter();
  const [filter, setFilter] = useState<FilterKind>("all");

  const filtered = filter === "all" ? TOOLS : TOOLS.filter((t) => t.kind === filter);
  const available = TOOLS.filter((t) => t.status === "available").length;

  const filters: { key: FilterKind; label: string }[] = [
    { key: "all", label: "All" },
    { key: "sdk", label: "SDKs" },
    { key: "runtime", label: "Runtimes" },
    { key: "tool", label: "Tools" },
    { key: "platform", label: "Platform" },
  ];

  return (
    <Page>
      <PageHeader
        title="Tools"
        description={`${available} tools you can stack to build the perfect world image.`}
      />

      {/* Filter tabs */}
      <div className="flex items-center gap-1">
        {filters.map(({ key, label }) => {
          const count = key === "all" ? TOOLS.length : TOOLS.filter((t) => t.kind === key).length;
          return (
            <button
              key={key}
              onClick={() => setFilter(key)}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full text-[11px] font-mono transition-all ${
                filter === key
                  ? "bg-white/[0.08] text-foreground/70 border border-white/[0.12]"
                  : "text-muted-foreground/30 hover:text-muted-foreground/50 border border-transparent"
              }`}
            >
              {label}
              <span className="text-[9px] text-muted-foreground/20">{count}</span>
            </button>
          );
        })}
      </div>

      {/* Tool grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
        {filtered.map((tool) => (
          <ToolCard
            key={tool.name}
            tool={tool}
            onClick={() => router.push(`/tools/${toolSlug(tool.name)}`)}
          />
        ))}
      </div>
    </Page>
  );
}

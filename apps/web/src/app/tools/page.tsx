'use client';

import {
    IconBookFilled,
    IconBrain,
    IconBrandDocker,
    IconBrandGit,
    IconBrandNodejs,
    IconBrandPython,
    IconChevronRight,
    IconClock,
    IconHammer,
    IconPackage,
    IconRocket,
    IconSearch,
    IconTerminal2,
} from '@tabler/icons-react';
import { useRouter } from 'next/navigation';

import { Page } from '@/components/page';
import { PageHeader } from '@/components/page-header';
import { usePageTitle } from '@/hooks/use-page-title';
import { type ToolDef, TOOLS, toolSlug, type ToolStatus } from '@/lib/tools-catalog';

// ── Icon map ────────────────────────────────────────────────────────────

const TOOL_ICONS: Record<string, React.ReactNode> = {
    'spwn:unix': <IconTerminal2 size={18} />,
    'spwn:node': <IconBrandNodejs size={18} />,
    'spwn:python': <IconBrandPython size={18} />,
    'spwn:build': <IconHammer size={18} />,
    'spwn:claude-code': <IconBrain size={18} />,
    'spwn:codex': <IconBrain size={18} />,
    'spwn:aider': <IconBrain size={18} />,
    'spwn:git': <IconBrandGit size={18} />,
    'spwn:docker-cli': <IconBrandDocker size={18} />,
    'spwn:qmd': <IconSearch size={18} />,
    'spwn:cli': <IconRocket size={18} />,
    'spwn:architect': <IconPackage size={18} />,
};

// ── Components ──────────────────────────────────────────────────────────

function StatusBadge({ status }: { status: ToolStatus }) {
    if (status === 'available') {
        return null;
    }
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
            className={`group text-left w-full rounded-xl border px-5 py-4 transition-all duration-200 ${
                tool.status === 'planned'
                    ? 'bg-white/[0.01] border-white/[0.04] opacity-50'
                    : 'bg-white/[0.03] border-white/[0.07] hover:border-white/[0.12] hover:bg-white/[0.05]'
            }`}
            onClick={onClick}
        >
            <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3">
                    <div className="w-9 h-9 rounded-lg bg-white/[0.05] border border-white/[0.08] flex items-center justify-center text-muted-foreground/40 group-hover:text-foreground/60 transition-colors">
                        {TOOL_ICONS[tool.name] ?? <IconPackage size={18} />}
                    </div>
                    <div>
                        <div className="flex items-center gap-2">
                            <span className="text-sm font-mono font-medium text-foreground/80">
                                {tool.name}
                            </span>
                        </div>
                        <p className="text-[11px] text-muted-foreground/40 mt-0.5">
                            {tool.description}
                        </p>
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <StatusBadge status={tool.status} />
                    <IconChevronRight
                        className="text-muted-foreground/20 group-hover:text-muted-foreground/40 transition-colors"
                        size={14}
                    />
                </div>
            </div>

            <div className="space-y-1.5 pl-12">
                <div className="flex items-baseline gap-2">
                    <span className="text-[10px] text-muted-foreground/25 w-16 shrink-0">
                        provides
                    </span>
                    <span className="text-[11px] text-foreground/50 font-mono">
                        {tool.provides}
                    </span>
                </div>
                {tool.dependencies.length > 0 && (
                    <div className="flex items-center gap-2">
                        <span className="text-[10px] text-muted-foreground/25 w-16 shrink-0">
                            depends
                        </span>
                        <div className="flex gap-1">
                            {tool.dependencies.map((d) => (
                                <span
                                    className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-white/[0.04] border border-white/[0.06] text-muted-foreground/35"
                                    key={d}
                                >
                                    {d}
                                </span>
                            ))}
                        </div>
                    </div>
                )}
                {tool.skills.length > 0 && (
                    <div className="flex items-center gap-2">
                        <span className="text-[10px] text-muted-foreground/25 w-16 shrink-0">
                            skills
                        </span>
                        <div className="flex gap-1">
                            {tool.skills.map((s) => (
                                <span
                                    className="flex items-center gap-1 text-[10px] font-mono px-1.5 py-0.5 rounded bg-purple-500/8 border border-purple-500/15 text-purple-400/50"
                                    key={s.name}
                                >
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

export default function ToolsPage() {
    usePageTitle('Tools');
    const router = useRouter();
    const available = TOOLS.filter((t) => t.status === 'available').length;

    return (
        <Page>
            <PageHeader
                description={`${available} tools you can stack to build the perfect world image.`}
                title="Tools"
            />

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
                {TOOLS.map((tool) => (
                    <ToolCard
                        key={tool.name}
                        onClick={() => router.push(`/tools/${toolSlug(tool.name)}`)}
                        tool={tool}
                    />
                ))}
            </div>
        </Page>
    );
}

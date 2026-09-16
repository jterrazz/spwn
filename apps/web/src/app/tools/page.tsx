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
import { TOOLS, toolSlug } from '@/domain/tools-catalog';
import type { ToolDef, ToolStatus } from '@/domain/tools-catalog';
import { usePageTitle } from '@/hooks/use-page-title';

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
        <span className="text-muted-foreground/30 flex items-center gap-1 font-mono text-[9px]">
            <IconClock size={10} />
            Planned
        </span>
    );
}

function ToolCard({ tool, onClick }: { tool: ToolDef; onClick: () => void }) {
    return (
        <button
            className={`group w-full rounded-xl border px-5 py-4 text-left transition-all duration-200 ${
                tool.status === 'planned'
                    ? 'border-white/[0.04] bg-white/[0.01] opacity-50'
                    : 'border-white/[0.07] bg-white/[0.03] hover:border-white/[0.12] hover:bg-white/[0.05]'
            }`}
            onClick={onClick}
        >
            <div className="mb-3 flex items-start justify-between">
                <div className="flex items-center gap-3">
                    <div className="text-muted-foreground/40 group-hover:text-foreground/60 flex h-9 w-9 items-center justify-center rounded-lg border border-white/[0.08] bg-white/[0.05] transition-colors">
                        {TOOL_ICONS[tool.name] ?? <IconPackage size={18} />}
                    </div>
                    <div>
                        <div className="flex items-center gap-2">
                            <span className="text-foreground/80 font-mono text-sm font-medium">
                                {tool.name}
                            </span>
                        </div>
                        <p className="text-muted-foreground/40 mt-0.5 text-[11px]">
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
                    <span className="text-muted-foreground/25 w-16 shrink-0 text-[10px]">
                        provides
                    </span>
                    <span className="text-foreground/50 font-mono text-[11px]">
                        {tool.provides}
                    </span>
                </div>
                {tool.dependencies.length > 0 && (
                    <div className="flex items-center gap-2">
                        <span className="text-muted-foreground/25 w-16 shrink-0 text-[10px]">
                            depends
                        </span>
                        <div className="flex gap-1">
                            {tool.dependencies.map((d) => (
                                <span
                                    className="text-muted-foreground/35 rounded border border-white/[0.06] bg-white/[0.04] px-1.5 py-0.5 font-mono text-[10px]"
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
                        <span className="text-muted-foreground/25 w-16 shrink-0 text-[10px]">
                            skills
                        </span>
                        <div className="flex gap-1">
                            {tool.skills.map((s) => (
                                <span
                                    className="flex items-center gap-1 rounded border border-purple-500/15 bg-purple-500/8 px-1.5 py-0.5 font-mono text-[10px] text-purple-400/50"
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

            <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
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

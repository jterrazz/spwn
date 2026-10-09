'use client';

import {
    IconArrowLeft,
    IconBookFilled,
    IconCheck,
    IconClock,
    IconTerminal,
} from '@tabler/icons-react';
import { useParams, useRouter } from 'next/navigation';

import { Page } from '../../../components/page';
import { PageHeader } from '../../../components/page-header';
import { DetailSection, MetaCard } from '../../../components/tools/detail-parts';
import { SkillViewer } from '../../../components/tools/skill-viewer';
import { getToolByName, toolSlug } from '../../../domain/tools-catalog';
import type { ToolDef } from '../../../domain/tools-catalog';
import { usePageTitle } from '../../../hooks/use-page-title';

const VERSIONS: Record<string, string> = { 'spwn:node': '20', 'spwn:python': '3' };

function plural(count: number, noun: string): string {
    return `${count} ${noun}${count === 1 ? '' : 's'}`;
}

function ToolMeta({ tool }: { tool: ToolDef }) {
    const available = tool.status === 'available';
    return (
        <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            <MetaCard
                icon={
                    available ? (
                        <IconCheck className="text-green-400/60" size={12} />
                    ) : (
                        <IconClock className="text-muted-foreground/30" size={12} />
                    )
                }
                label="Status"
                value={available ? 'Available' : 'Planned'}
            />
            <MetaCard label="Version" value={VERSIONS[tool.name] ?? 'latest'} />
            <MetaCard
                icon={
                    tool.skills.length > 0 ? (
                        <IconBookFilled className="text-purple-400/50" size={12} />
                    ) : undefined
                }
                label="Skills"
                value={tool.skills.length > 0 ? plural(tool.skills.length, 'file') : 'None'}
            />
            <MetaCard
                icon={<IconTerminal className="text-muted-foreground/30" size={12} />}
                label="Verify"
                value={plural(tool.verify.length, 'check')}
            />
        </div>
    );
}

function ToolFacts({ tool, onOpen }: { tool: ToolDef; onOpen: (name: string) => void }) {
    return (
        <div className="space-y-4">
            <DetailSection label="Provides">
                <p className="text-foreground/60 font-mono text-sm">{tool.provides}</p>
            </DetailSection>
            <DetailSection label="Use when">
                <p className="text-muted-foreground/50 text-sm">{tool.useWhen}</p>
            </DetailSection>
            {tool.dependencies.length > 0 && (
                <DetailSection label="Dependencies">
                    <div className="flex flex-wrap gap-1.5">
                        {tool.dependencies.map((dependency) => (
                            <button
                                className="text-muted-foreground/50 hover:text-foreground/70 rounded-lg border border-white/[0.08] bg-white/[0.04] px-2 py-1 font-mono text-[11px] transition-colors hover:border-white/[0.15]"
                                key={dependency}
                                onClick={() => {
                                    onOpen(dependency);
                                }}
                                type="button"
                            >
                                {dependency}
                            </button>
                        ))}
                    </div>
                </DetailSection>
            )}
            <DetailSection label="Verification commands">
                <div className="space-y-1">
                    {tool.verify.map((command) => (
                        <div
                            className="text-muted-foreground/40 flex items-center gap-2 font-mono text-[11px]"
                            key={command}
                        >
                            <span className="text-green-400/40">$</span>
                            <span>command -v {command}</span>
                        </div>
                    ))}
                </div>
            </DetailSection>
        </div>
    );
}

function ManifestSnippet({ tool }: { tool: ToolDef }) {
    return (
        <DetailSection label="Add to world manifest">
            <pre className="text-foreground/50 rounded-lg border border-white/[0.06] bg-white/[0.03] px-4 py-3 font-mono text-[12px] leading-relaxed">
                {`tools:\n  - ${tool.name}`}
            </pre>
            {tool.dependencies.length > 0 && (
                <p className="text-muted-foreground/25 mt-2 text-[10px]">
                    {tool.dependencies.join(', ')} will be installed automatically.
                </p>
            )}
        </DetailSection>
    );
}

function Skills({ tool }: { tool: ToolDef }) {
    if (tool.skills.length === 0) {
        return null;
    }
    return (
        <div className="space-y-3">
            <h2 className="font-heading text-foreground/60 text-sm tracking-wide">Skills</h2>
            <p className="text-muted-foreground/30 text-[11px]">
                Skills are markdown guides installed at{' '}
                <code className="rounded bg-white/[0.04] px-1 py-0.5 font-mono text-[10px]">
                    /world/skills/{toolSlug(tool.name)}/
                </code>{' '}
                inside the container. Agents read these to learn how to use the tool.
            </p>
            <SkillViewer skills={tool.skills} />
        </div>
    );
}

function BackButton({ onBack }: { onBack: () => void }) {
    return (
        <button
            aria-label="Back to Tools"
            className="text-muted-foreground/30 hover:text-foreground/60 flex h-8 w-8 items-center justify-center rounded-lg transition-colors hover:bg-white/[0.05]"
            onClick={onBack}
            type="button"
        >
            <IconArrowLeft size={16} />
        </button>
    );
}

export default function ToolDetailPage() {
    const { slug } = useParams<{ slug: string }>();
    const router = useRouter();
    const tool = getToolByName(`spwn:${slug}`);
    const back = () => {
        router.push('/tools');
    };
    usePageTitle(tool === undefined ? 'Tool Not Found' : tool.name);

    if (tool === undefined) {
        return (
            <Page>
                <PageHeader description={`No tool named spwn:${slug}`} title="Tool Not Found" />
                <button
                    className="text-muted-foreground/40 hover:text-foreground/60 text-sm transition-colors"
                    onClick={back}
                    type="button"
                >
                    Back to Tools
                </button>
            </Page>
        );
    }
    return (
        <Page>
            <PageHeader
                description={tool.description}
                leading={<BackButton onBack={back} />}
                title={tool.name}
            />
            <ToolMeta tool={tool} />
            <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                <ToolFacts
                    onOpen={(name) => {
                        if (getToolByName(name) !== undefined) {
                            router.push(`/tools/${toolSlug(name)}`);
                        }
                    }}
                    tool={tool}
                />
                <div>
                    <ManifestSnippet tool={tool} />
                </div>
            </div>
            <Skills tool={tool} />
        </Page>
    );
}

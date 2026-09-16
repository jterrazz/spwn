'use client';

import {
    IconArrowLeft,
    IconBookFilled,
    IconCheck,
    IconClock,
    IconTerminal,
} from '@tabler/icons-react';
import { useParams, useRouter } from 'next/navigation';
import { useState } from 'react';

import { Page } from '@/components/page';
import { PageHeader } from '@/components/page-header';
import { getToolByName, TOOLS } from '@/domain/tools-catalog';
import type { SkillFile } from '@/domain/tools-catalog';
import { usePageTitle } from '@/hooks/use-page-title';

// ── Markdown renderer (simple) ──────────────────────────────────────────

function SkillContent({ content }: { content: string }) {
    // Simple markdown → HTML: headings, code blocks, lists, inline code
    const lines = content.split('\n');
    const elements: React.ReactNode[] = [];
    let i = 0;
    let key = 0;

    while (i < lines.length) {
        const line = lines[i];
        if (line === undefined) {
            i++;
            continue;
        }

        // Code block
        if (line.trimStart().startsWith('```')) {
            const codeLines: string[] = [];
            i++;
            while (i < lines.length) {
                const codeLine = lines[i];
                if (codeLine === undefined || codeLine.trimStart().startsWith('```')) {
                    break;
                }
                codeLines.push(codeLine);
                i++;
            }
            i++; // Skip closing ```
            elements.push(
                <pre
                    className="text-foreground/60 my-3 overflow-x-auto rounded-lg border border-white/[0.06] bg-white/[0.03] px-4 py-3 font-mono text-[11px] leading-relaxed"
                    key={key++}
                >
                    {codeLines.join('\n')}
                </pre>,
            );
            continue;
        }

        // Heading
        if (line.startsWith('### ')) {
            elements.push(
                <h4 className="text-foreground/60 mt-5 mb-2 text-xs font-medium" key={key++}>
                    {line.slice(4)}
                </h4>,
            );
            i++;
            continue;
        }
        if (line.startsWith('## ')) {
            elements.push(
                <h3 className="text-foreground/70 mt-6 mb-2 text-sm font-medium" key={key++}>
                    {line.slice(3)}
                </h3>,
            );
            i++;
            continue;
        }
        if (line.startsWith('# ')) {
            // Skip top-level heading (already shown in page header)
            i++;
            continue;
        }

        // List item
        if (line.trimStart().startsWith('- ')) {
            elements.push(
                <div
                    className="text-muted-foreground/50 my-0.5 flex gap-2 pl-2 text-[12px] leading-relaxed"
                    key={key++}
                >
                    <span className="text-muted-foreground/25 mt-1.5 shrink-0">-</span>
                    <span>{renderInlineCode(line.trimStart().slice(2))}</span>
                </div>,
            );
            i++;
            continue;
        }

        // Numbered list
        const numMatch = /^(?<num>\d+)\.\s+(?<text>.+)/.exec(line.trimStart());
        if (numMatch) {
            elements.push(
                <div
                    className="text-muted-foreground/50 my-0.5 flex gap-2 pl-2 text-[12px] leading-relaxed"
                    key={key++}
                >
                    <span className="text-muted-foreground/25 w-4 shrink-0 text-right">
                        {numMatch.groups!.num}.
                    </span>
                    <span>{renderInlineCode(numMatch.groups!.text!)}</span>
                </div>,
            );
            i++;
            continue;
        }

        // Empty line
        if (line.trim() === '') {
            i++;
            continue;
        }

        // Paragraph
        elements.push(
            <p className="text-muted-foreground/50 my-2 text-[12px] leading-relaxed" key={key++}>
                {renderInlineCode(line)}
            </p>,
        );
        i++;
    }

    return <>{elements}</>;
}

function renderInlineCode(text: string): React.ReactNode {
    const parts = text.split(/(?<code>`[^`]+`)/g);
    let codeSeq = 0;
    return parts.map((part) => {
        if (part.startsWith('`') && part.endsWith('`')) {
            codeSeq += 1;
            return (
                <code
                    className="text-foreground/60 rounded border border-white/[0.08] bg-white/[0.05] px-1 py-0.5 font-mono text-[11px]"
                    key={`code-${codeSeq}-${part}`}
                >
                    {part.slice(1, -1)}
                </code>
            );
        }
        return part;
    });
}

// ── Skill tab viewer ────────────────────────────────────────────────────

function SkillViewer({ skills }: { skills: SkillFile[] }) {
    const [active, setActive] = useState(0);

    const activeSkill = skills[active] ?? skills[0];
    if (activeSkill === undefined) {
        return null;
    }

    return (
        <div className="overflow-hidden rounded-xl border border-white/[0.07]">
            {/* Tab bar */}
            {skills.length > 1 && (
                <div className="flex border-b border-white/[0.06] bg-white/[0.02]">
                    {skills.map((s, i) => (
                        <button
                            className={`-mb-[1px] flex items-center gap-1.5 border-b-2 px-4 py-2.5 font-mono text-[11px] transition-colors ${
                                active === i
                                    ? 'text-foreground/70 border-purple-400/60 bg-purple-500/5'
                                    : 'text-muted-foreground/30 hover:text-muted-foreground/50 border-transparent'
                            }`}
                            key={s.name}
                            onClick={() => setActive(i)}
                        >
                            <IconBookFilled size={10} />
                            {s.name}
                        </button>
                    ))}
                </div>
            )}

            {/* Single skill header (when only one) */}
            {skills.length === 1 && (
                <div className="flex items-center gap-2 border-b border-white/[0.06] bg-white/[0.02] px-5 py-3">
                    <IconBookFilled className="text-purple-400/50" size={12} />
                    <span className="text-muted-foreground/40 font-mono text-[11px]">
                        {activeSkill.name}
                    </span>
                </div>
            )}

            {/* Content */}
            <div className="px-5 py-4">
                <SkillContent content={activeSkill.content} />
            </div>
        </div>
    );
}

// ── Page ─────────────────────────────────────────────────────────────────

export default function ToolDetailPage() {
    const params = useParams();
    const router = useRouter();
    const slug = params.slug as string;
    const tool = getToolByName(`spwn:${slug}`);

    usePageTitle(tool ? tool.name : 'Tool Not Found');

    if (!tool) {
        return (
            <Page>
                <PageHeader description={`No tool named spwn:${slug}`} title="Tool Not Found" />
                <button
                    className="text-muted-foreground/40 hover:text-foreground/60 text-sm transition-colors"
                    onClick={() => router.push('/tools')}
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
                leading={
                    <button
                        className="text-muted-foreground/30 hover:text-foreground/60 flex h-8 w-8 items-center justify-center rounded-lg transition-colors hover:bg-white/[0.05]"
                        onClick={() => router.push('/tools')}
                    >
                        <IconArrowLeft size={16} />
                    </button>
                }
                title={tool.name}
            />

            {/* Meta grid */}
            <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
                <MetaCard
                    icon={
                        tool.status === 'available' ? (
                            <IconCheck className="text-green-400/60" size={12} />
                        ) : (
                            <IconClock className="text-muted-foreground/30" size={12} />
                        )
                    }
                    label="Status"
                    value={tool.status === 'available' ? 'Available' : 'Planned'}
                />
                <MetaCard
                    label="Version"
                    value={(() => {
                        if (tool.name === 'spwn:node') {
                            return '20';
                        }
                        if (tool.name === 'spwn:python') {
                            return '3';
                        }
                        return 'latest';
                    })()}
                />
                <MetaCard
                    icon={
                        tool.skills.length > 0 ? (
                            <IconBookFilled className="text-purple-400/50" size={12} />
                        ) : undefined
                    }
                    label="Skills"
                    value={
                        tool.skills.length > 0
                            ? `${tool.skills.length} file${tool.skills.length > 1 ? 's' : ''}`
                            : 'None'
                    }
                />
                <MetaCard
                    icon={<IconTerminal className="text-muted-foreground/30" size={12} />}
                    label="Verify"
                    value={`${tool.verify.length} check${tool.verify.length > 1 ? 's' : ''}`}
                />
            </div>

            {/* Details */}
            <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                {/* Left: info */}
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
                                {tool.dependencies.map((d) => (
                                    <button
                                        className="text-muted-foreground/50 hover:text-foreground/70 rounded-lg border border-white/[0.08] bg-white/[0.04] px-2 py-1 font-mono text-[11px] transition-colors hover:border-white/[0.15]"
                                        key={d}
                                        onClick={() => {
                                            const depTool = TOOLS.find((t) => t.name === d);
                                            if (depTool) {
                                                router.push(`/tools/${d.replace('spwn:', '')}`);
                                            }
                                        }}
                                    >
                                        {d}
                                    </button>
                                ))}
                            </div>
                        </DetailSection>
                    )}

                    <DetailSection label="Verification commands">
                        <div className="space-y-1">
                            {tool.verify.map((v) => (
                                <div
                                    className="text-muted-foreground/40 flex items-center gap-2 font-mono text-[11px]"
                                    key={v}
                                >
                                    <span className="text-green-400/40">$</span>
                                    <span>command -v {v}</span>
                                </div>
                            ))}
                        </div>
                    </DetailSection>
                </div>

                {/* Right: manifest example */}
                <div>
                    <DetailSection label="Add to world manifest">
                        <pre className="text-foreground/50 rounded-lg border border-white/[0.06] bg-white/[0.03] px-4 py-3 font-mono text-[12px] leading-relaxed">
                            {`tools:
  - ${tool.name}`}
                        </pre>
                        {tool.dependencies.length > 0 && (
                            <p className="text-muted-foreground/25 mt-2 text-[10px]">
                                {tool.dependencies.join(', ')} will be installed automatically.
                            </p>
                        )}
                    </DetailSection>
                </div>
            </div>

            {/* Skills */}
            {tool.skills.length > 0 && (
                <div className="space-y-3">
                    <h2 className="font-heading text-foreground/60 text-sm tracking-wide">
                        Skills
                    </h2>
                    <p className="text-muted-foreground/30 text-[11px]">
                        Skills are markdown guides installed at{' '}
                        <code className="rounded bg-white/[0.04] px-1 py-0.5 font-mono text-[10px]">
                            /world/skills/{tool.name.replace('spwn:', '')}/
                        </code>{' '}
                        inside the container. Agents read these to learn how to use the tool.
                    </p>
                    <SkillViewer skills={tool.skills} />
                </div>
            )}
        </Page>
    );
}

// ── Helpers ──────────────────────────────────────────────────────────────

function MetaCard({
    label,
    value,
    icon,
}: {
    label: string;
    value: string;
    icon?: React.ReactNode;
}) {
    return (
        <div className="rounded-lg border border-white/[0.06] bg-white/[0.02] px-4 py-3">
            <p className="text-muted-foreground/25 mb-1 text-[9px] tracking-widest uppercase">
                {label}
            </p>
            <div className="flex items-center gap-1.5">
                {icon}
                <span className="text-foreground/70 font-mono text-sm">{value}</span>
            </div>
        </div>
    );
}

function DetailSection({ label, children }: { label: string; children: React.ReactNode }) {
    return (
        <div>
            <p className="text-muted-foreground/25 mb-2 text-[10px] tracking-widest uppercase">
                {label}
            </p>
            {children}
        </div>
    );
}

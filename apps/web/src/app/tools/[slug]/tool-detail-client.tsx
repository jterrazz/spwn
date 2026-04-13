"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { PageHeader } from "@/components/page-header";
import { Page } from "@/components/page";
import { usePageTitle } from "@/hooks/use-page-title";
import { ActionButton } from "@/components/action-button";
import {
  IconArrowLeft,
  IconCheck,
  IconClock,
  IconBookFilled,
  IconTerminal,
} from "@tabler/icons-react";
import { TOOLS, KIND_META, getToolByName, type ToolDef, type SkillFile } from "@/lib/tools-catalog";

// ── Markdown renderer (simple) ──────────────────────────────────────────

function SkillContent({ content }: { content: string }) {
  // Simple markdown → HTML: headings, code blocks, lists, inline code
  const lines = content.split("\n");
  const elements: React.ReactNode[] = [];
  let i = 0;
  let key = 0;

  while (i < lines.length) {
    const line = lines[i];

    // Code block
    if (line.trimStart().startsWith("```")) {
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i].trimStart().startsWith("```")) {
        codeLines.push(lines[i]);
        i++;
      }
      i++; // skip closing ```
      elements.push(
        <pre key={key++} className="bg-white/[0.03] border border-white/[0.06] rounded-lg px-4 py-3 text-[11px] font-mono text-foreground/60 leading-relaxed overflow-x-auto my-3">
          {codeLines.join("\n")}
        </pre>
      );
      continue;
    }

    // Heading
    if (line.startsWith("### ")) {
      elements.push(<h4 key={key++} className="text-xs font-medium text-foreground/60 mt-5 mb-2">{line.slice(4)}</h4>);
      i++;
      continue;
    }
    if (line.startsWith("## ")) {
      elements.push(<h3 key={key++} className="text-sm font-medium text-foreground/70 mt-6 mb-2">{line.slice(3)}</h3>);
      i++;
      continue;
    }
    if (line.startsWith("# ")) {
      // Skip top-level heading (already shown in page header)
      i++;
      continue;
    }

    // List item
    if (line.trimStart().startsWith("- ")) {
      elements.push(
        <div key={key++} className="flex gap-2 text-[12px] text-muted-foreground/50 leading-relaxed pl-2 my-0.5">
          <span className="text-muted-foreground/25 shrink-0 mt-1.5">-</span>
          <span>{renderInlineCode(line.trimStart().slice(2))}</span>
        </div>
      );
      i++;
      continue;
    }

    // Numbered list
    const numMatch = line.trimStart().match(/^(\d+)\.\s+(.+)/);
    if (numMatch) {
      elements.push(
        <div key={key++} className="flex gap-2 text-[12px] text-muted-foreground/50 leading-relaxed pl-2 my-0.5">
          <span className="text-muted-foreground/25 shrink-0 w-4 text-right">{numMatch[1]}.</span>
          <span>{renderInlineCode(numMatch[2])}</span>
        </div>
      );
      i++;
      continue;
    }

    // Empty line
    if (line.trim() === "") {
      i++;
      continue;
    }

    // Paragraph
    elements.push(<p key={key++} className="text-[12px] text-muted-foreground/50 leading-relaxed my-2">{renderInlineCode(line)}</p>);
    i++;
  }

  return <>{elements}</>;
}

function renderInlineCode(text: string): React.ReactNode {
  const parts = text.split(/(`[^`]+`)/g);
  return parts.map((part, i) => {
    if (part.startsWith("`") && part.endsWith("`")) {
      return (
        <code key={i} className="text-[11px] font-mono bg-white/[0.05] border border-white/[0.08] rounded px-1 py-0.5 text-foreground/60">
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

  if (skills.length === 0) return null;

  return (
    <div className="rounded-xl border border-white/[0.07] overflow-hidden">
      {/* Tab bar */}
      {skills.length > 1 && (
        <div className="flex border-b border-white/[0.06] bg-white/[0.02]">
          {skills.map((s, i) => (
            <button
              key={s.name}
              onClick={() => setActive(i)}
              className={`flex items-center gap-1.5 px-4 py-2.5 text-[11px] font-mono transition-colors border-b-2 -mb-[1px] ${
                active === i
                  ? "text-foreground/70 border-purple-400/60 bg-purple-500/5"
                  : "text-muted-foreground/30 border-transparent hover:text-muted-foreground/50"
              }`}
            >
              <IconBookFilled size={10} />
              {s.name}
            </button>
          ))}
        </div>
      )}

      {/* Single skill header (when only one) */}
      {skills.length === 1 && (
        <div className="flex items-center gap-2 px-5 py-3 border-b border-white/[0.06] bg-white/[0.02]">
          <IconBookFilled size={12} className="text-purple-400/50" />
          <span className="text-[11px] font-mono text-muted-foreground/40">{skills[0].name}</span>
        </div>
      )}

      {/* Content */}
      <div className="px-5 py-4">
        <SkillContent content={skills[active].content} />
      </div>
    </div>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────

export default function ToolDetailPage() {
  const params = useParams();
  const router = useRouter();
  const slug = params.slug as string;
  const tool = getToolByName(`@spwn/${slug}`);

  usePageTitle(tool ? tool.name : "Tool Not Found");

  if (!tool) {
    return (
      <Page>
        <PageHeader title="Tool Not Found" description={`No tool named @spwn/${slug}`} />
        <button onClick={() => router.push("/tools")} className="text-sm text-muted-foreground/40 hover:text-foreground/60 transition-colors">
          Back to Tools
        </button>
      </Page>
    );
  }

  const { label: kindLabel, color: kindColor } = KIND_META[tool.kind];

  return (
    <Page>
      <PageHeader
        title={tool.name}
        description={tool.description}
        leading={
          <button
            onClick={() => router.push("/tools")}
            className="w-8 h-8 rounded-lg flex items-center justify-center text-muted-foreground/30 hover:text-foreground/60 hover:bg-white/[0.05] transition-colors"
          >
            <IconArrowLeft size={16} />
          </button>
        }
        actions={
          <span className={`text-[10px] font-mono uppercase tracking-wider px-2 py-1 rounded-full border ${kindColor}`}>
            {kindLabel}
          </span>
        }
      />

      {/* Meta grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetaCard label="Status" value={tool.status === "available" ? "Available" : "Planned"} icon={tool.status === "available" ? <IconCheck size={12} className="text-green-400/60" /> : <IconClock size={12} className="text-muted-foreground/30" />} />
        <MetaCard label="Version" value={tool.name === "@spwn/node" ? "20" : tool.name === "@spwn/python" ? "3" : "latest"} />
        <MetaCard label="Skills" value={tool.skills.length > 0 ? `${tool.skills.length} file${tool.skills.length > 1 ? "s" : ""}` : "None"} icon={tool.skills.length > 0 ? <IconBookFilled size={12} className="text-purple-400/50" /> : undefined} />
        <MetaCard label="Verify" value={`${tool.verify.length} check${tool.verify.length > 1 ? "s" : ""}`} icon={<IconTerminal size={12} className="text-muted-foreground/30" />} />
      </div>

      {/* Details */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Left: info */}
        <div className="space-y-4">
          <DetailSection label="Provides">
            <p className="text-sm font-mono text-foreground/60">{tool.provides}</p>
          </DetailSection>

          <DetailSection label="Use when">
            <p className="text-sm text-muted-foreground/50">{tool.useWhen}</p>
          </DetailSection>

          {tool.deps.length > 0 && (
            <DetailSection label="Dependencies">
              <div className="flex flex-wrap gap-1.5">
                {tool.deps.map((d) => (
                  <button
                    key={d}
                    onClick={() => {
                      const depTool = TOOLS.find(t => t.name === d);
                      if (depTool) router.push(`/tools/${d.replace("@spwn/", "")}`);
                    }}
                    className="text-[11px] font-mono px-2 py-1 rounded-lg bg-white/[0.04] border border-white/[0.08] text-muted-foreground/50 hover:text-foreground/70 hover:border-white/[0.15] transition-colors"
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
                <div key={v} className="flex items-center gap-2 text-[11px] font-mono text-muted-foreground/40">
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
            <pre className="bg-white/[0.03] border border-white/[0.06] rounded-lg px-4 py-3 text-[12px] font-mono text-foreground/50 leading-relaxed">
{`tools:
  - ${tool.name}`}
            </pre>
            {tool.deps.length > 0 && (
              <p className="text-[10px] text-muted-foreground/25 mt-2">
                {tool.deps.join(", ")} will be installed automatically.
              </p>
            )}
          </DetailSection>
        </div>
      </div>

      {/* Skills */}
      {tool.skills.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-sm font-heading tracking-wide text-foreground/60">Skills</h2>
          <p className="text-[11px] text-muted-foreground/30">
            Skills are markdown guides installed at <code className="text-[10px] font-mono bg-white/[0.04] px-1 py-0.5 rounded">/world/skills/{tool.name.replace("@spwn/", "")}/</code> inside the container. Agents read these to learn how to use the tool.
          </p>
          <SkillViewer skills={tool.skills} />
        </div>
      )}
    </Page>
  );
}

// ── Helpers ──────────────────────────────────────────────────────────────

function MetaCard({ label, value, icon }: { label: string; value: string; icon?: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-white/[0.06] bg-white/[0.02] px-4 py-3">
      <p className="text-[9px] uppercase tracking-widest text-muted-foreground/25 mb-1">{label}</p>
      <div className="flex items-center gap-1.5">
        {icon}
        <span className="text-sm font-mono text-foreground/70">{value}</span>
      </div>
    </div>
  );
}

function DetailSection({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="text-[10px] uppercase tracking-widest text-muted-foreground/25 mb-2">{label}</p>
      {children}
    </div>
  );
}

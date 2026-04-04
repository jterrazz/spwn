"use client";

import { useParams } from "next/navigation";
import { usePageTitle } from "@/hooks/use-page-title";
import { KnowledgeBrowser } from "@/components/knowledge-browser";
import { IconBook2 } from "@tabler/icons-react";

export default function KnowledgePage() {
  usePageTitle("Knowledge");
  const params = useParams();
  const worldId = params.id as string;

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <IconBook2 size={24} className="text-foreground/50" />
        <div>
          <h1 className="text-2xl font-heading tracking-wide text-foreground/90">Knowledge</h1>
          <p className="text-xs font-mono text-muted-foreground/40 mt-0.5">
            World knowledge base — managed by the Architect
          </p>
        </div>
      </div>

      <KnowledgeBrowser worldId={worldId} />
    </div>
  );
}

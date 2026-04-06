"use client";

import { useParams } from "next/navigation";
import { useState } from "react";
import { IconFolderOpen } from "@tabler/icons-react";
import { usePageTitle } from "@/hooks/use-page-title";
import { KnowledgeBrowser } from "@/components/knowledge-browser";
import { PageHeader } from "@/components/page-header";
import { Page } from "@/components/page";
import { ExpandingSearch } from "@/components/expanding-search";
import { ActionButton } from "@/components/action-button";
import { isTauri } from "@/lib/tauri";

export default function KnowledgePage() {
  usePageTitle("Knowledge");
  const params = useParams();
  const worldId = params.id as string;
  const [searchQuery, setSearchQuery] = useState("");

  const openInFinder = async () => {
    // The knowledge dir is always at ~/.spwn/knowledge/ on the host.
    const path = `${process.env.HOME || "~"}/.spwn/knowledge`;
    if (isTauri()) {
      try {
        // @ts-expect-error Tauri shell plugin global
        await window.__TAURI__.shell.open(path);
      } catch {
        // Fallback: copy path
        await navigator.clipboard.writeText(path);
      }
    } else {
      await navigator.clipboard.writeText("~/.spwn/knowledge");
    }
  };

  return (
    <Page>
      <PageHeader
        title="Knowledge"
        description="World knowledge base — managed by the Architect."
        actions={
          <>
            <ExpandingSearch
              value={searchQuery}
              onChange={setSearchQuery}
              placeholder="Search files…"
            />
            <ActionButton
              compact
              onClick={openInFinder}
              label="Open in Finder"
              icon={<IconFolderOpen size={16} stroke={2.2} />}
            />
          </>
        }
      />
      <KnowledgeBrowser
        worldId={worldId}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
      />
    </Page>
  );
}

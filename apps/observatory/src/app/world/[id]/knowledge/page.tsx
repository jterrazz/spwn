"use client";

import { useParams } from "next/navigation";
import { useState } from "react";
import { usePageTitle } from "@/hooks/use-page-title";
import { KnowledgeBrowser } from "@/components/knowledge-browser";
import { PageHeader } from "@/components/page-header";
import { Page } from "@/components/page";
import { ExpandingSearch } from "@/components/expanding-search";

export default function KnowledgePage() {
  usePageTitle("Knowledge");
  const params = useParams();
  const worldId = params.id as string;
  const [searchQuery, setSearchQuery] = useState("");

  return (
    <Page>
      <PageHeader
        title="Knowledge"
        description="World knowledge base — managed by the Architect."
        actions={
          <ExpandingSearch
            value={searchQuery}
            onChange={setSearchQuery}
            placeholder="Search files…"
          />
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

"use client";

import { useParams } from "next/navigation";
import { usePageTitle } from "@/hooks/use-page-title";
import { KnowledgeBrowser } from "@/components/knowledge-browser";
import { PageHeader } from "@/components/page-header";
import { Page } from "@/components/page";

export default function KnowledgePage() {
  usePageTitle("Knowledge");
  const params = useParams();
  const worldId = params.id as string;

  return (
    <Page>
      <PageHeader
        title="Knowledge"
        description="World knowledge base — managed by the Architect."
      />
      <KnowledgeBrowser worldId={worldId} />
    </Page>
  );
}

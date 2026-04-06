"use client";

import { useParams, useRouter } from "next/navigation";
import { useEffect } from "react";

/**
 * Thin redirect: /world/{id}/{agent} → /agents/{agent}?world={id}
 *
 * The canonical agent page lives at /agents/[name]. When accessed from a
 * world context, the world ID is passed as a query param so the agent
 * page can pin chat + diagnostics to the right container.
 */
export default function WorldAgentRedirect() {
  const params = useParams();
  const router = useRouter();
  const worldId = params.id as string;
  const agentName = params.agent as string;

  useEffect(() => {
    router.replace(`/agents/${agentName}?world=${worldId}`);
  }, [router, agentName, worldId]);

  return null;
}

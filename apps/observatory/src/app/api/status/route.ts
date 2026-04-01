import { NextResponse } from "next/server";
import { getWorlds, getAgents, getLimboAgents } from "@/lib/spwn-data";

export const dynamic = "force-dynamic";

export async function GET() {
  const worlds = await getWorlds();
  const agents = await getAgents();
  const limbo = await getLimboAgents(worlds);

  return NextResponse.json({
    worlds: worlds.length,
    agents: agents.length,
    running: worlds.filter((w) => w.status === "running").length,
    limbo: limbo.length,
  });
}

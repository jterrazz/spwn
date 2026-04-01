import { NextResponse } from "next/server";
import { spwnExec } from "@/lib/spwn-exec";

export const dynamic = "force-dynamic";

export async function POST(request: Request) {
  const body = await request.json();
  const { agent, workspace, config, tier } = body as {
    agent?: string;
    workspace?: string;
    config?: string;
    tier?: string;
  };

  if (!agent) {
    return NextResponse.json({ error: "Agent name is required" }, { status: 400 });
  }

  const args = ["up", "--agent", agent];
  if (workspace) args.push("-w", workspace);
  if (config) args.push("--config", config);
  if (tier) args.push("--tier", tier);

  const result = spwnExec(args, 60000);
  if (result.ok) {
    return NextResponse.json({ ok: true, output: result.stdout });
  }
  return NextResponse.json({ error: result.error }, { status: 500 });
}

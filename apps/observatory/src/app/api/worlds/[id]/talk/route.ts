import { NextResponse } from "next/server";
import { spwnExec } from "@/lib/spwn-exec";

export const dynamic = "force-dynamic";

export async function POST(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id } = await params;
    const body = await request.json();
    const { message, agent } = body as { message?: string; agent?: string };

    if (!message) {
      return NextResponse.json({ error: "Message is required" }, { status: 400 });
    }

    // If agent name is provided directly, use it
    let agentName = agent;

    // Otherwise, find the agent for this world
    if (!agentName) {
      const lsResult = spwnExec(["ls", "--json"]);
      if (lsResult.ok && lsResult.stdout) {
        try {
          const worlds = JSON.parse(lsResult.stdout);
          const world = Array.isArray(worlds)
            ? worlds.find((w: { id?: string }) => w.id === id)
            : null;
          if (world) {
            agentName = world.agent;
          }
        } catch {
          // Failed to parse ls output
        }
      }
    }

    if (!agentName) {
      return NextResponse.json(
        { error: "Could not find agent for world " + id },
        { status: 404 }
      );
    }

    const result = spwnExec(["agent", "talk", agentName, message], 120000);

    if (result.ok && result.stdout) {
      // Strip CLI header from response
      let response = result.stdout.trim();
      if (response.startsWith("Agent:")) {
        const idx = response.indexOf("\n\n");
        if (idx !== -1) {
          response = response.slice(idx).trim();
        }
      }
      return NextResponse.json({ response });
    }

    return NextResponse.json(
      { error: result.error || "Failed to talk to agent" },
      { status: 500 }
    );
  } catch {
    return NextResponse.json(
      { error: "Invalid request body" },
      { status: 400 }
    );
  }
}

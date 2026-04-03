import { spwnExec } from "@/lib/spwn-exec";
import { spwnStream } from "@/lib/spwn-stream";

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
      return Response.json({ error: "Message is required" }, { status: 400 });
    }

    let agentName = agent;

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
          // Failed to parse
        }
      }
    }

    if (!agentName) {
      return Response.json(
        { error: "Could not find agent for world " + id },
        { status: 404 }
      );
    }

    const stream = spwnStream(["agent", "talk", agentName, message], 120000);

    return new Response(stream, {
      headers: {
        "Content-Type": "text/plain; charset=utf-8",
        "Transfer-Encoding": "chunked",
        "Cache-Control": "no-cache",
      },
    });
  } catch {
    return Response.json(
      { error: "Invalid request body" },
      { status: 400 }
    );
  }
}

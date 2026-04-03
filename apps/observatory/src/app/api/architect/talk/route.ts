import { spwnStream } from "@/lib/spwn-stream";

export const dynamic = "force-dynamic";

export async function POST(request: Request) {
  try {
    const body = await request.json();
    const { message } = body as { message?: string };

    if (!message) {
      return Response.json({ error: "Message is required" }, { status: 400 });
    }

    const args = message.trim().split(/\s+/);
    if (args.length === 0) {
      return Response.json({ error: "Empty command" }, { status: 400 });
    }

    const stream = spwnStream(args);

    return new Response(stream, {
      headers: {
        "Content-Type": "text/plain; charset=utf-8",
        "Transfer-Encoding": "chunked",
        "Cache-Control": "no-cache",
      },
    });
  } catch {
    return Response.json({ error: "Invalid request body" }, { status: 400 });
  }
}

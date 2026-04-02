import { NextResponse } from "next/server";
import { spwnExec } from "@/lib/spwn-exec";

export const dynamic = "force-dynamic";

export async function POST(request: Request) {
  const body = await request.json();
  const { message } = body as { message?: string };

  if (!message) {
    return NextResponse.json({ error: "Message is required" }, { status: 400 });
  }

  // Parse message into command args
  const args = message.trim().split(/\s+/);
  if (args.length === 0) {
    return NextResponse.json({ error: "Empty command" }, { status: 400 });
  }

  const result = spwnExec(args);
  if (result.ok) {
    return NextResponse.json({ response: result.stdout });
  }
  return NextResponse.json({ response: result.error, error: result.error }, { status: 200 });
}

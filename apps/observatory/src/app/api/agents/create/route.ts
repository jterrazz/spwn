import { NextResponse } from "next/server";
import { spwnExec } from "@/lib/spwn-exec";

export const dynamic = "force-dynamic";

export async function POST(request: Request) {
  try {
  const body = await request.json();
  const { name } = body as { name?: string };

  if (!name) {
    return NextResponse.json({ error: "Agent name is required" }, { status: 400 });
  }

  const result = spwnExec(["agent", "new", name]);
  if (result.ok) {
    return NextResponse.json({ ok: true, output: result.stdout });
  }
  return NextResponse.json({ error: result.error }, { status: 500 });
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }
}

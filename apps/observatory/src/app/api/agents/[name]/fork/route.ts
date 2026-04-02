import { NextResponse } from "next/server";
import { spwnExec } from "@/lib/spwn-exec";

export const dynamic = "force-dynamic";

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  const { name } = await params;
  try {
  const body = await request.json();
  const { target } = body as { target?: string };

  if (!target) {
    return NextResponse.json({ error: "Target name is required" }, { status: 400 });
  }

  const result = spwnExec(["agent", "fork", name, target]);
  if (result.ok) {
    return NextResponse.json({ ok: true, output: result.stdout });
  }
  return NextResponse.json({ error: result.error }, { status: 500 });
  } catch {
    return NextResponse.json({ error: "Invalid request body" }, { status: 400 });
  }
}

import { NextResponse } from "next/server";
import { spwnExec } from "@/lib/spwn-exec";

export const dynamic = "force-dynamic";

export async function POST(
  _request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params;
  const result = spwnExec(["down", id]);
  if (result.ok) {
    return NextResponse.json({ ok: true, output: result.stdout });
  }
  return NextResponse.json({ error: result.error }, { status: 500 });
}

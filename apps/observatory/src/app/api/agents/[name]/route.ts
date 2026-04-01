import { NextResponse } from "next/server";
import { getAgentProfile } from "@/lib/spwn-data";
import { spwnExec } from "@/lib/spwn-exec";

export const dynamic = "force-dynamic";

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  const { name } = await params;
  const profile = await getAgentProfile(name);
  if (!profile) {
    return NextResponse.json({ error: "Agent not found" }, { status: 404 });
  }
  return NextResponse.json(profile);
}

export async function DELETE(
  _request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  const { name } = await params;
  const result = spwnExec(["agent", "rm", name]);
  if (result.ok) {
    return NextResponse.json({ ok: true, output: result.stdout });
  }
  return NextResponse.json({ error: result.error }, { status: 500 });
}

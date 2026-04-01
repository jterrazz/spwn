import { NextResponse } from "next/server";
import { getAgentMindTree } from "@/lib/spwn-data";

export const dynamic = "force-dynamic";

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ name: string }> }
) {
  const { name } = await params;
  const tree = await getAgentMindTree(name);
  return NextResponse.json(tree);
}

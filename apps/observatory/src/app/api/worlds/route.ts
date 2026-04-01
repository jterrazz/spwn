import { NextResponse } from "next/server";
import { getWorlds } from "@/lib/spwn-data";

export const dynamic = "force-dynamic";

export async function GET() {
  const worlds = await getWorlds();
  return NextResponse.json(worlds);
}

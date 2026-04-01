import { NextResponse } from "next/server";
import { spwnExec } from "@/lib/spwn-exec";

export const dynamic = "force-dynamic";

export async function POST() {
  const result = spwnExec(["architect", "stop"], 30000);

  if (!result.ok) {
    return NextResponse.json(
      { error: result.error ?? "Failed to stop architect" },
      { status: 500 }
    );
  }

  return NextResponse.json({ ok: true, output: result.stdout });
}

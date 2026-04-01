import { NextResponse } from "next/server";
import { spwnExec } from "@/lib/spwn-exec";

export const dynamic = "force-dynamic";

export async function POST() {
  const result = spwnExec(["architect", "start"], 60000);

  if (!result.ok) {
    return NextResponse.json(
      { error: result.error ?? "Failed to start architect" },
      { status: 500 }
    );
  }

  return NextResponse.json({ ok: true, output: result.stdout });
}

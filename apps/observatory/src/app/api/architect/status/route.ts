import { NextResponse } from "next/server";
import { spwnExec } from "@/lib/spwn-exec";

export const dynamic = "force-dynamic";

export async function GET() {
  const result = spwnExec(["architect", "status"]);

  if (!result.ok) {
    return NextResponse.json({
      status: "stopped",
      containerId: null,
      uptime: null,
      error: result.error,
    });
  }

  const stdout = result.stdout ?? "";

  // Parse status output
  const running = /running|up|online/i.test(stdout);
  const containerMatch = stdout.match(/container[:\s]+([a-f0-9]{12,})/i);
  const uptimeMatch = stdout.match(/uptime[:\s]+([\dhms]+)/i);

  return NextResponse.json({
    status: running ? "running" : "stopped",
    containerId: containerMatch?.[1] ?? null,
    uptime: uptimeMatch?.[1] ?? null,
    raw: stdout,
  });
}

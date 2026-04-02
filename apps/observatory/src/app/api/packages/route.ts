import { NextResponse } from "next/server";
import { spwnExec } from "@/lib/spwn-exec";

export const dynamic = "force-dynamic";

export async function GET() {
  const result = spwnExec(["get", "ls"]);

  if (!result.ok) {
    return NextResponse.json({ packages: [], error: result.error });
  }

  const stdout = result.stdout ?? "";
  const lines = stdout
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith("NAME") && !l.startsWith("---"));

  // Filter out CLI messages that aren't actual package entries
  const packageLines = lines.filter((l) =>
    !l.toLowerCase().includes("no packages") &&
    !l.toLowerCase().includes("run '") &&
    !l.toLowerCase().includes("run \"")
  );

  const packages = packageLines.map((line) => {
    const parts = line.split(/\s{2,}|\t/);
    return {
      name: parts[0] ?? line,
      version: parts[1] ?? "",
      description: parts[2] ?? "",
    };
  });

  return NextResponse.json({ packages });
}

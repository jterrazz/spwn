import { NextResponse, NextRequest } from "next/server";
import fs from "node:fs";
import path from "node:path";
import os from "node:os";

export const dynamic = "force-dynamic";

function blueprintDir(): string {
  const home = process.env.SPWN_HOME || path.join(os.homedir(), ".spwn");
  return path.join(home, "blueprint");
}

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  const { path: pathSegments } = await params;
  const relPath = pathSegments.join("/");

  // Prevent directory traversal
  if (relPath.includes("..")) {
    return NextResponse.json({ error: "Invalid path" }, { status: 400 });
  }

  const base = blueprintDir();
  const absPath = path.join(base, relPath);

  // Ensure resolved path is under base
  const cleanPath = path.resolve(absPath);
  const cleanBase = path.resolve(base);
  if (!cleanPath.startsWith(cleanBase)) {
    return NextResponse.json({ error: "Path outside blueprint directory" }, { status: 400 });
  }

  try {
    const content = await fs.promises.readFile(absPath, "utf-8");
    return NextResponse.json({ path: relPath, content });
  } catch {
    return NextResponse.json({ error: "File not found" }, { status: 404 });
  }
}

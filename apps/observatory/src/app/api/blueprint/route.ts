import { NextResponse } from "next/server";
import fs from "node:fs";
import path from "node:path";
import os from "node:os";

export const dynamic = "force-dynamic";

function blueprintDir(): string {
  const home = process.env.SPWN_HOME || path.join(os.homedir(), ".spwn");
  return path.join(home, "blueprint");
}

interface FileEntry {
  path: string;
  size: number;
  modified: string;
}

async function walkDir(dir: string, base: string): Promise<FileEntry[]> {
  const entries: FileEntry[] = [];
  try {
    const items = await fs.promises.readdir(dir, { withFileTypes: true });
    for (const item of items) {
      if (item.name.startsWith(".")) continue;
      const fullPath = path.join(dir, item.name);
      if (item.isDirectory()) {
        const sub = await walkDir(fullPath, base);
        entries.push(...sub);
      } else {
        const stat = await fs.promises.stat(fullPath);
        const relPath = path.relative(base, fullPath);
        entries.push({
          path: relPath,
          size: stat.size,
          modified: stat.mtime.toISOString(),
        });
      }
    }
  } catch {
    // directory doesn't exist
  }
  return entries;
}

export async function GET() {
  const dir = blueprintDir();
  const files = await walkDir(dir, dir);
  return NextResponse.json({ files });
}

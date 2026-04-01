import { existsSync, readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { expect } from "vitest";

export class MindAssertion {
  private agentDir: string;

  constructor(
    private spwnHome: string,
    private agentName: string,
  ) {
    this.agentDir = join(spwnHome, "agents", agentName);
  }

  exists(): this {
    expect(existsSync(this.agentDir)).toBe(true);
    return this;
  }

  hasLayer(layer: string): this {
    expect(existsSync(join(this.agentDir, layer))).toBe(true);
    return this;
  }

  hasFile(relPath: string): this {
    expect(existsSync(join(this.agentDir, relPath))).toBe(true);
    return this;
  }

  hasSession(universeId: string): this {
    const sessionsDir = join(this.agentDir, "sessions");
    if (!existsSync(sessionsDir)) throw new Error("No sessions directory");
    const files = readdirSync(sessionsDir);
    const found = files.some((f) => f.includes(universeId));
    expect(found).toBe(true);
    return this;
  }

  hasJournalEntries(minCount: number): this {
    const journalDir = join(this.agentDir, "memory", "journal");
    if (!existsSync(journalDir)) {
      if (minCount > 0) throw new Error("No journal directory");
      return this;
    }
    const entries = readdirSync(journalDir).filter((f) => f.endsWith(".md"));
    expect(entries.length).toBeGreaterThanOrEqual(minCount);
    return this;
  }

  journalContains(text: string): this {
    const journalDir = join(this.agentDir, "memory", "journal");
    const entries = readdirSync(journalDir).filter((f) => f.endsWith(".md"));
    const allContent = entries
      .map((f) => readFileSync(join(journalDir, f), "utf-8"))
      .join("\n");
    expect(allContent).toContain(text);
    return this;
  }
}

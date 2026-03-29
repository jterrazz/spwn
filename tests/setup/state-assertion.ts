import { readFileSync, existsSync } from "node:fs";
import { join } from "node:path";
import { expect } from "vitest";

export class StateAssertion {
  private statePath: string;

  constructor(private spwnHome: string) {
    this.statePath = join(spwnHome, "state.json");
  }

  private load(): Array<Record<string, unknown>> {
    if (!existsSync(this.statePath)) return [];
    return JSON.parse(readFileSync(this.statePath, "utf-8"));
  }

  exists(): this {
    expect(existsSync(this.statePath)).toBe(true);
    return this;
  }

  universeCount(n: number): this {
    expect(this.load().length).toBe(n);
    return this;
  }

  hasUniverse(universeId: string): this {
    const state = this.load();
    expect(state.some((u) => u.id === universeId)).toBe(true);
    return this;
  }

  universeStatus(universeId: string, status: string): this {
    const state = this.load();
    const u = state.find((u) => u.id === universeId);
    expect(u).toBeDefined();
    expect(u!.status).toBe(status);
    return this;
  }

  hasAgent(universeId: string, agentName: string): this {
    const state = this.load();
    const u = state.find((u) => u.id === universeId);
    expect(u).toBeDefined();
    expect(u!.agent).toBe(agentName);
    return this;
  }

  noUniverse(universeId: string): this {
    const state = this.load();
    expect(state.some((u) => u.id === universeId)).toBe(false);
    return this;
  }
}

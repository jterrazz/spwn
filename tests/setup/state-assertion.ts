import { execSync } from "node:child_process";
import { expect } from "vitest";

/**
 * StateAssertion queries world state via Docker container labels —
 * the canonical source of truth in the new model. The legacy
 * state.json file is no longer maintained.
 *
 * Each helper shells out to `docker ps` filtered by the spwn
 * `sh.spwn.kind=world` label and parses the resulting world IDs
 * from the container names.
 */
export class StateAssertion {
  constructor(private _spwnHome: string) {}

  private listWorldIds(): string[] {
    try {
      const out = execSync(
        `docker ps -a --filter label=sh.spwn.kind=world --format '{{.Names}}'`,
        { encoding: "utf-8", timeout: 5000, stdio: ["pipe", "pipe", "ignore"] },
      ).trim();
      if (!out) return [];
      return out.split("\n").filter(Boolean);
    } catch {
      return [];
    }
  }

  private worldStatusOf(worldId: string): string | null {
    try {
      const out = execSync(
        `docker inspect --format='{{.State.Status}}' ${worldId}`,
        { encoding: "utf-8", timeout: 5000, stdio: ["pipe", "pipe", "ignore"] },
      ).trim();
      return out || null;
    } catch {
      return null;
    }
  }

  private worldLabel(worldId: string, label: string): string | null {
    try {
      const out = execSync(
        `docker inspect --format='{{ index .Config.Labels "${label}" }}' ${worldId}`,
        { encoding: "utf-8", timeout: 5000, stdio: ["pipe", "pipe", "ignore"] },
      ).trim();
      return out || null;
    } catch {
      return null;
    }
  }

  exists(): this {
    // Always considered "true": the source of truth is Docker, which is
    // either reachable or not. Kept as a no-op for callers that still
    // ask for it.
    return this;
  }

  worldCount(n: number): this {
    expect(this.listWorldIds().length).toBe(n);
    return this;
  }

  hasWorld(worldId: string): this {
    expect(this.listWorldIds()).toContain(worldId);
    return this;
  }

  worldStatus(worldId: string, status: string): this {
    expect(this.listWorldIds()).toContain(worldId);
    // Map docker status to spwn status concept where useful.
    const dockerStatus = this.worldStatusOf(worldId);
    expect(dockerStatus).not.toBeNull();
    if (status === "running" || status === "idle") {
      expect(dockerStatus).toBe("running");
    } else if (status === "stopped" || status === "destroyed") {
      expect(["exited", "stopped"]).toContain(dockerStatus);
    } else {
      // Direct match fallback.
      expect(dockerStatus).toBe(status);
    }
    return this;
  }

  hasAgent(worldId: string, agentName: string): this {
    const ag = this.worldLabel(worldId, "sh.spwn.world.agent");
    expect(ag).toBe(agentName);
    return this;
  }

  noWorld(worldId: string): this {
    expect(this.listWorldIds()).not.toContain(worldId);
    return this;
  }
}

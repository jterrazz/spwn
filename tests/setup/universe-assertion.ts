import { execSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { expect } from "vitest";

export class AgentProbeAssertion {
  constructor(private probe: Record<string, unknown>) {}

  sawMind(): this {
    expect(this.probe.mind_exists).toBe(true);
    return this;
  }

  sawPersonas(): this {
    expect(this.probe.mind_personas).toBe(true);
    return this;
  }

  sawPhysics(): this {
    expect(this.probe.physics_exists).toBe(true);
    return this;
  }

  sawFaculties(): this {
    expect(this.probe.faculties_exists).toBe(true);
    return this;
  }

  sawWorkspace(): this {
    expect(this.probe.workspace_exists).toBe(true);
    return this;
  }

  hasSessionId(): this {
    expect(this.probe.session_id).toBeTruthy();
    return this;
  }

  wasResumed(): this {
    expect(this.probe.resume).toBe(true);
    return this;
  }

  wasNotResumed(): this {
    expect(this.probe.resume).toBe(false);
    return this;
  }

  physicsContains(text: string): this {
    expect(this.probe.physics_content).toContain(text);
    return this;
  }

  facultiesContains(text: string): this {
    expect(this.probe.faculties_content).toContain(text);
    return this;
  }
}

export class UniverseAssertion {
  private containerId: string | null;

  constructor(
    private universeId: string,
    private spwnHome: string,
  ) {
    try {
      const stateRaw = readFileSync(`${spwnHome}/state.json`, "utf-8");
      const state: Array<{ id: string; container_id: string }> =
        JSON.parse(stateRaw);
      const universe = state.find((u) => u.id === universeId);
      this.containerId = universe?.container_id ?? null;
    } catch {
      this.containerId = null;
    }
  }

  private requireContainerId(): string {
    if (!this.containerId) {
      throw new Error(
        `Universe ${this.universeId} not found in state — cannot inspect container`,
      );
    }
    return this.containerId;
  }

  /** Assert container is running */
  toBeRunning(): this {
    const cid = this.requireContainerId();
    const result = execSync(
      `docker inspect --format='{{.State.Running}}' ${cid}`,
      { encoding: "utf-8", timeout: 5000 },
    ).trim();
    expect(result).toBe("true");
    return this;
  }

  /** Assert container is gone (works even if universe was removed from state) */
  toNotExist(): this {
    if (!this.containerId) {
      // Universe already removed from state — container is gone
      return this;
    }
    try {
      execSync(`docker inspect ${this.containerId}`, { timeout: 5000 });
      throw new Error("Container still exists");
    } catch (err: unknown) {
      if (err instanceof Error && err.message === "Container still exists")
        throw err;
      // Expected: docker inspect fails = container doesn't exist
    }
    return this;
  }

  /** Read a file inside the container */
  readFile(path: string): string {
    const cid = this.requireContainerId();
    return execSync(`docker exec ${cid} cat ${path}`, {
      encoding: "utf-8",
      timeout: 5000,
    });
  }

  /** Check if file exists inside container */
  fileExists(path: string): boolean {
    const cid = this.requireContainerId();
    try {
      execSync(`docker exec ${cid} test -e ${path}`, {
        timeout: 5000,
      });
      return true;
    } catch {
      return false;
    }
  }

  /** Assert file exists with optional content check */
  toHaveFile(path: string, containing?: string): this {
    expect(this.fileExists(path)).toBe(true);
    if (containing) {
      const content = this.readFile(path);
      expect(content).toContain(containing);
    }
    return this;
  }

  /** Assert file does not exist */
  toNotHaveFile(path: string): this {
    expect(this.fileExists(path)).toBe(false);
    return this;
  }

  /** Assert directory exists */
  toHaveDirectory(path: string): this {
    const cid = this.requireContainerId();
    try {
      execSync(`docker exec ${cid} test -d ${path}`, {
        timeout: 5000,
      });
    } catch {
      throw new Error(`Expected directory ${path} to exist in container`);
    }
    return this;
  }

  /** Get physics.md content */
  physics(): string {
    return this.readFile("/universe/physics.md");
  }

  /** Get faculties.md content */
  faculties(): string {
    return this.readFile("/universe/faculties.md");
  }

  /** Read the mock agent probe output */
  agentProbe(): AgentProbeAssertion {
    const cid = this.requireContainerId();
    try {
      const raw = execSync(
        `docker exec ${cid} cat /tmp/claude-mock.json`,
        { encoding: "utf-8", timeout: 5000 },
      );
      return new AgentProbeAssertion(JSON.parse(raw));
    } catch {
      throw new Error("Mock agent probe not found — agent may not have run");
    }
  }

  /** Execute a command inside the container */
  exec(cmd: string): string {
    const cid = this.requireContainerId();
    return execSync(`docker exec ${cid} sh -c '${cmd}'`, {
      encoding: "utf-8",
      timeout: 10000,
    }).trim();
  }

  /** Get docker inspect data */
  inspect(): Record<string, unknown> {
    const cid = this.requireContainerId();
    const raw = execSync(`docker inspect ${cid}`, {
      encoding: "utf-8",
      timeout: 5000,
    });
    return (JSON.parse(raw) as Record<string, unknown>[])[0];
  }

  /** Assert network mode */
  toHaveNetwork(mode: string): this {
    const data = this.inspect() as { HostConfig?: { NetworkMode?: string } };
    expect(data.HostConfig?.NetworkMode).toContain(mode);
    return this;
  }

  /** Assert memory limit */
  toHaveMemoryLimit(bytes: number): this {
    const data = this.inspect() as { HostConfig?: { Memory?: number } };
    expect(data.HostConfig?.Memory).toBe(bytes);
    return this;
  }
}

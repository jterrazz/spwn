import { execSync, spawnSync } from "node:child_process";
import { expect } from "vitest";

/**
 * Docker inspect JSON structure (subset of fields we care about).
 * Used for type-safe access to container metadata.
 */
export interface DockerInspectData {
  Id?: string;
  State?: {
    Status?: string;
    Running?: boolean;
    Pid?: number;
    ExitCode?: number;
    StartedAt?: string;
    FinishedAt?: string;
  };
  HostConfig?: {
    Memory?: number;
    NanoCpus?: number;
    CpuQuota?: number;
    CpuPeriod?: number;
    PidsLimit?: number;
    NetworkMode?: string;
    Binds?: string[];
  };
  Config?: {
    Image?: string;
    Env?: string[];
    Cmd?: string[];
    WorkingDir?: string;
  };
  Mounts?: Array<{
    Type?: string;
    Source?: string;
    Destination?: string;
    Mode?: string;
    RW?: boolean;
  }>;
}

/**
 * Structured result from the mock agent probe inside a container.
 */
export interface AgentProbeData {
  mind_exists?: boolean;
  mind_identity?: boolean;
  physics_exists?: boolean;
  physics_content?: string;
  faculties_exists?: boolean;
  faculties_content?: string;
  workspace_exists?: boolean;
  session_id?: string;
  resume?: boolean;
}

export class AgentProbeAssertion {
  constructor(private probe: Record<string, unknown>) {}

  sawMind(): this {
    expect(this.probe.mind_exists).toBe(true);
    return this;
  }

  sawIdentity(): this {
    expect(this.probe.mind_identity).toBe(true);
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

export class WorldAssertion {
  private containerId: string | null;

  constructor(
    private worldId: string,
    _spwnHome: string,
  ) {
    // Container labels are the source of truth - the world ID is also
    // the container name (set at create time), so we resolve it via
    // `docker inspect <name>` directly.
    try {
      const id = execSync(
        `docker inspect --format='{{.Id}}' ${worldId}`,
        { encoding: "utf-8", timeout: 5000, stdio: ["pipe", "pipe", "ignore"] },
      ).trim();
      this.containerId = id || null;
    } catch {
      this.containerId = null;
    }
  }

  private requireContainerId(): string {
    if (!this.containerId) {
      throw new Error(
        `World ${this.worldId} not found - no container with that name (looked up via docker inspect)`,
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

  /** Assert container is gone (works even if world was removed from state) */
  toNotExist(): this {
    if (!this.containerId) {
      // World already removed from state - container is gone
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
    return this.readFile("/world/physics.md");
  }

  /** Get faculties.md content */
  faculties(): string {
    return this.readFile("/world/faculties.md");
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
      throw new Error("Mock agent probe not found - agent may not have run");
    }
  }

  /** Execute a command inside the container */
  exec(cmd: string): string {
    const cid = this.requireContainerId();
    // Use the array form of spawnSync so node doesn't shell-interpret
    // the command - single-quoted strings inside `cmd` would otherwise
    // collide with the outer `sh -c '...'` quoting.
    const result = spawnSync("docker", ["exec", cid, "sh", "-c", cmd], {
      encoding: "utf-8",
      timeout: 10000,
    });
    if (result.status !== 0) {
      throw new Error(
        `docker exec failed (exit ${result.status}): ${result.stderr || result.stdout}`,
      );
    }
    return (result.stdout || "").trim();
  }

  /**
   * Get the full Docker inspect data for the container.
   * Returns a typed DockerInspectData object for safe access.
   */
  inspect(): DockerInspectData {
    const cid = this.requireContainerId();
    const raw = execSync(`docker inspect ${cid}`, {
      encoding: "utf-8",
      timeout: 5000,
    });
    const parsed = JSON.parse(raw) as DockerInspectData[];
    return parsed[0];
  }

  /** Assert network mode */
  toHaveNetwork(mode: string): this {
    const data = this.inspect();
    expect(data.HostConfig?.NetworkMode).toContain(mode);
    return this;
  }

  /** Assert memory limit */
  toHaveMemoryLimit(bytes: number): this {
    const data = this.inspect();
    expect(data.HostConfig?.Memory).toBe(bytes);
    return this;
  }

  /** Assert CPU limit (in nanocpus) */
  toHaveCpuLimit(nanoCpus: number): this {
    const data = this.inspect();
    expect(data.HostConfig?.NanoCpus).toBe(nanoCpus);
    return this;
  }

  /** Assert pids limit */
  toHavePidsLimit(limit: number): this {
    const data = this.inspect();
    expect(data.HostConfig?.PidsLimit).toBe(limit);
    return this;
  }
}

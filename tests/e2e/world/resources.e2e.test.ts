import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("world resource limits", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("memory limit is applied to container", () => {
    // GIVEN - a spawned world (default config has memory set)
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;

    // WHEN - inspecting the container
    const inspectData = ctx.world(id).inspect();

    // THEN - memory limit is set (not 0 = unlimited)
    expect(inspectData.HostConfig?.Memory).toBeDefined();
    expect(inspectData.HostConfig!.Memory).toBeGreaterThan(0);
  });

  test("CPU limit is applied to container", () => {
    // GIVEN - a spawned world (default config has cpu set)
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;

    // WHEN - inspecting the container
    const inspectData = ctx.world(id).inspect();

    // THEN - CPU limit is set (via NanoCpus or CpuQuota - not both 0)
    const nanoCpus = inspectData.HostConfig?.NanoCpus ?? 0;
    const cpuQuota = inspectData.HostConfig?.CpuQuota ?? 0;
    expect(nanoCpus + cpuQuota).toBeGreaterThan(0);
  });

  test("default limits are reasonable (not unlimited)", () => {
    // GIVEN - a spawned world with default config
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;

    // WHEN - inspecting the container
    const inspectData = ctx.world(id).inspect();

    // THEN - memory is set and reasonable (between 64MB and 16GB)
    const memory = inspectData.HostConfig?.Memory ?? 0;
    expect(memory).toBeGreaterThanOrEqual(64 * 1024 * 1024); // >= 64MB
    expect(memory).toBeLessThanOrEqual(16 * 1024 * 1024 * 1024); // <= 16GB

    // AND - CPU is constrained
    const nanoCpus = inspectData.HostConfig?.NanoCpus ?? 0;
    const cpuQuota = inspectData.HostConfig?.CpuQuota ?? 0;
    expect(nanoCpus + cpuQuota).toBeGreaterThan(0);
  });

  test("pids limit is set", () => {
    // GIVEN - a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;

    // WHEN - inspecting the container
    const inspectData = ctx.world(id).inspect();

    // THEN - pids limit is set (not 0 or -1 which means unlimited)
    const pidsLimit = inspectData.HostConfig?.PidsLimit ?? 0;
    expect(pidsLimit).toBeGreaterThan(0);
  });
});

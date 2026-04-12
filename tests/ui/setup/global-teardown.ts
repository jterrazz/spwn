import { execSync } from "node:child_process";

export default async function globalTeardown() {
  console.log("\n[global-teardown] Stopping API server...");

  // Kill the API server
  const api = globalThis.__spwnApiProcess;
  if (api && !api.killed) {
    api.kill("SIGTERM");
    // Wait a moment for graceful shutdown
    await new Promise((r) => setTimeout(r, 2000));
    if (!api.killed) {
      api.kill("SIGKILL");
    }
  }

  // Destroy any running test worlds
  console.log("[global-teardown] Cleaning up Docker containers...");
  try {
    execSync(
      'docker ps --filter "label=spwn.kind=world" -q | xargs -r docker rm -f',
      { stdio: "ignore", timeout: 10_000 },
    );
  } catch {
    // best effort
  }

  // Clean up temp home
  const home = globalThis.__spwnHome;
  if (home) {
    try {
      execSync(`rm -rf "${home}"`, { stdio: "ignore", timeout: 5_000 });
    } catch {
      // best effort
    }
  }

  console.log("[global-teardown] Done ✓\n");
}

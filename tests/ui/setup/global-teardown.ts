import { execSync } from "node:child_process";
import { readFileSync } from "node:fs";

export default async function globalTeardown() {
  console.log("\n[global-teardown] Stopping API server...");

  const configPath = process.env.SPWN_TEST_CONFIG;
  let home: string | undefined;

  if (configPath) {
    try {
      const config = JSON.parse(readFileSync(configPath, "utf-8"));
      home = config.spwnHome;

      // Kill the API server via PID file
      if (config.pidFile) {
        try {
          const pid = readFileSync(config.pidFile, "utf-8").trim();
          process.kill(Number(pid), "SIGTERM");
          await new Promise((r) => setTimeout(r, 2000));
          try { process.kill(Number(pid), 0); process.kill(Number(pid), "SIGKILL"); } catch { /* already dead */ }
        } catch { /* best effort */ }
      }
    } catch { /* config not found */ }
  }

  // Also kill by port as fallback
  try {
    execSync("lsof -ti:9877 | xargs kill -9", { stdio: "ignore", timeout: 5_000 });
  } catch { /* nothing on that port */ }

  // Destroy any running test worlds
  console.log("[global-teardown] Cleaning up Docker containers...");
  try {
    execSync(
      'docker ps --filter "label=spwn.kind=world" -q | xargs -r docker rm -f',
      { stdio: "ignore", timeout: 10_000 },
    );
  } catch { /* best effort */ }

  // Clean up temp home
  if (home) {
    try {
      execSync(`rm -rf "${home}"`, { stdio: "ignore", timeout: 5_000 });
    } catch { /* best effort */ }
  }

  console.log("[global-teardown] Done ✓\n");
}

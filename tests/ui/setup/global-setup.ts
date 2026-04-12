import { execSync, spawn, type ChildProcess } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(__dirname, "../../..");
const BIN = resolve(REPO_ROOT, "bin/spwn");
const API_PORT = 9877;

// Stored globally so teardown can read them
declare global {
  var __spwnApiProcess: ChildProcess | undefined;
  var __spwnHome: string | undefined;
}

export default async function globalSetup() {
  console.log("\n[global-setup] Building spwn binary...");
  execSync("make build", { cwd: REPO_ROOT, stdio: "inherit" });

  // Isolated SPWN_HOME so tests don't pollute the user's real data
  const home = mkdtempSync(join(tmpdir(), "spwn-ui-test-"));
  globalThis.__spwnHome = home;

  // Create minimal directory structure
  execSync(`mkdir -p "${home}/worlds" "${home}/agents"`, { stdio: "ignore" });

  // Install all examples into the isolated home
  console.log("[global-setup] Installing examples...");
  execSync(`${BIN} example install startup`, {
    env: { ...process.env, SPWN_HOME: home },
    stdio: "inherit",
  });
  execSync(`${BIN} example install matrix`, {
    env: { ...process.env, SPWN_HOME: home },
    stdio: "inherit",
  });

  // Start API server — detached so it survives the setup process
  console.log(`[global-setup] Starting API server on port ${API_PORT}...`);
  const api = spawn(BIN, ["dash", "start", "--port", String(API_PORT)], {
    env: { ...process.env, SPWN_HOME: home },
    stdio: ["ignore", "ignore", "ignore"],
    detached: true,
  });
  api.unref(); // allow setup process to exit without killing the child

  // Write PID so teardown can find it
  const pidFile = join(home, ".api-pid");
  writeFileSync(pidFile, String(api.pid));

  // Wait for API to be ready
  await new Promise<void>((resolve, reject) => {
    const timeout = setTimeout(() => reject(new Error("API server did not start within 15s")), 15_000);
    const check = () => {
      fetch(`http://localhost:${API_PORT}/api/version`)
        .then((r) => {
          if (r.ok) {
            clearTimeout(timeout);
            resolve();
          } else {
            setTimeout(check, 500);
          }
        })
        .catch(() => setTimeout(check, 500));
    };
    check();
  });

  console.log("[global-setup] API server ready ✓\n");

  // Write config so tests + teardown can find everything
  const configPath = join(home, ".test-config.json");
  writeFileSync(
    configPath,
    JSON.stringify({ apiPort: API_PORT, spwnHome: home, bin: BIN, pidFile }),
  );

  // Playwright passes env vars between setup → tests → teardown
  process.env.SPWN_TEST_CONFIG = configPath;
}

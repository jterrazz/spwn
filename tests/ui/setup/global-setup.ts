import { execSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(__dirname, "../../..");
const BIN = resolve(REPO_ROOT, "bin/spwn");

/**
 * Build the binary and install examples so the gallery works.
 * Servers are managed by playwright.config.ts webServer — not here.
 */
export default async function globalSetup() {
  console.log("\n[global-setup] Building spwn binary...");
  execSync("make build", { cwd: REPO_ROOT, stdio: "inherit" });

  console.log("[global-setup] Installing bundled examples...");
  for (const slug of ["startup", "matrix"]) {
    try {
      execSync(`${BIN} example install ${slug}`, { stdio: "inherit", timeout: 15_000 });
    } catch {
      // idempotent — already installed is fine
    }
  }

  console.log("[global-setup] Ready ✓\n");
}

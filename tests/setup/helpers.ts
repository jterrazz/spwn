import { mkdtempSync, mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

/** Create an isolated SPWN_HOME directory */
export function createSpwnHome(): string {
  const dir = mkdtempSync(join(tmpdir(), "spwn-test-"));
  mkdirSync(join(dir, "universes"), { recursive: true });
  mkdirSync(join(dir, "agents"), { recursive: true });
  return dir;
}

/** Create a minimal agent Mind */
export function createAgent(spwnHome: string, name: string) {
  const agentDir = join(spwnHome, "agents", name);
  const layers = [
    "personas",
    "skills",
    "knowledge",
    "playbooks",
    "journal",
    "sessions",
  ];
  for (const layer of layers) {
    mkdirSync(join(agentDir, layer), { recursive: true });
  }
  writeFileSync(
    join(agentDir, "personas", "default.md"),
    `# ${name}\nYou are a test agent.`,
  );
}

/** Create a minimal universe config */
export function createUniverseConfig(
  spwnHome: string,
  name: string,
  overrides: Record<string, unknown> = {},
) {
  const config = {
    name,
    physics: {
      cpu: 1,
      memory: "512m",
      timeout: "30m",
      "max-processes": 100,
    },
    elements: ["@unix", "@git"],
    ...overrides,
  };
  writeFileSync(
    join(spwnHome, "universes", `${name}.yaml`),
    Object.entries(config)
      .map(([k, v]) => `${k}: ${JSON.stringify(v)}`)
      .join("\n"),
  );
}

/** Create a minimal org.yaml */
export function createOrgManifest(spwnHome: string, name = "test-org") {
  writeFileSync(
    join(spwnHome, "org.yaml"),
    `name: ${name}\nversion: "1.0"\n`,
  );
}

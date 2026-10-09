import { cpSync, mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

const CATALOG = resolve(import.meta.dirname, '../../../catalog');

const MANIFEST = `version: 1
name: website

dependencies:
  - "spwn:unix"
  - "spwn:git"

worlds:
  matrix:
    agents: [neo]
    workspaces: [.]
  startup:
    agents: [ceo, analyst]
    workspaces: [.]
`;

const AGENTS = [
    ['matrix', 'neo'],
    ['startup', 'ceo'],
    ['startup', 'analyst'],
] as const;

function freshProject(): string {
    const root = mkdtempSync(join(tmpdir(), 'spwn-website-'));
    for (const [example, agent] of AGENTS) {
        cpSync(join(CATALOG, example, 'agents', agent), join(root, 'spwn', 'agents', agent), {
            recursive: true,
        });
    }
    writeFileSync(join(root, 'spwn.yaml'), MANIFEST);
    mkdirSync(join(root, 'spwn-home'));
    writeFileSync(join(root, 'spwn-home', '.onboarding-complete'), '');
    return root;
}

process.stdout.write(freshProject());

import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    platform: 'node',
    presets: [test()],
    test: {
        projects: {
            cli: {
                literate: {
                    include: [
                        'cli/**/*.spec.yaml',
                        'adapters/**/*.spec.yaml',
                        'usecases/**/*.spec.yaml',
                    ],
                    runner: './cli/cli.specification.ts',
                },
                reason: 'the documents of one adapter (`adapters/<tech>/`) record its command lines, those of one use case (`usecases/<family>/<verb>/`) tell it over a seeded world; all run on the CLI runner of `cli/`, the one cli project a repository keeps',
            },
            website: {
                include: ['website/**/*.spec.ts'],
                reason: 'this member IS the specs tree, so its specifications sit at its root, where the derived glob (`specs/website/...`) does not look',
            },
        },
    },
    typescript: {
        knip: {
            entry: {
                'cli/cli.specification.ts':
                    'the runner every `<case>.spec.yaml` document is bound to by the literate plugin; no TypeScript spec imports it',
                'cli/__setup__/github-api.ts':
                    'the GitHub API stub a document serves (`serve: [github]`); the runner spawns it with node, nothing imports it',
                'cli/__setup__/tailscale-api.ts':
                    'the Tailscale API stub a document serves (`serve: tailscale`); the runner spawns it with node, nothing imports it',
            },
            ignoreBinaries: {
                go: 'the CLI specification builds the Go binary it drives; the toolchain is a machine requirement, never an npm dependency',
            },
            ignoreDependencies: {
                playwright:
                    'the optional peer @jterrazz/test imports at run time when the website specification visits a page',
            },
        },
        oxlint: {
            overrides: [
                {
                    files: ['website/**/*.spec.ts'],
                    reason: 'each scenario of apps/window/SCENARIOS.md is a test whose title opens on its id (`A4 ...`), and one not proved yet is declared all the same, a `test.todo` naming why it waits, so a journey whose scenarios all wait imports its runner for its side alone until its first body drives it; scenarios_test.go of specs/architecture holds the page and the titles together',
                    rules: {
                        'import/no-unassigned-import': 'off',
                        'vitest/prefer-lowercase-title': 'off',
                        'vitest/warn-todo': 'off',
                    },
                },
            ],
        },
        roots: {
            adapters: 'this member IS the specs tree',
            cli: 'this member IS the specs tree',
            machine: 'this member IS the specs tree',
            usecases: 'this member IS the specs tree',
            website: 'this member IS the specs tree',
        },
    },
});

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
        },
        roots: {
            adapters: 'this member IS the specs tree',
            cli: 'this member IS the specs tree',
            machine: 'this member IS the specs tree',
            usecases: 'this member IS the specs tree',
        },
    },
});

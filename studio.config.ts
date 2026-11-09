import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    kind: 'cli',
    platform: 'node',
    presets: [test()],
    test: {
        projects: {
            cli: {
                literate: {
                    include: [
                        'specs/cli/**/*.spec.yaml',
                        'specs/adapters/**/*.spec.yaml',
                        'specs/usecases/**/*.spec.yaml',
                        'specs/tui/**/*.spec.yaml',
                    ],
                },
                reason: 'the documents of one adapter (`specs/adapters/<tech>/`) record its command lines, those of one use case (`specs/usecases/<family>/<verb>/`) tell it over a seeded world, and those of one screen (`specs/tui/<screen>/`) drive the TUI over one; all run on the CLI runner of `specs/cli/`, the one cli project a repository keeps',
            },
        },
    },
    typescript: {
        knip: {
            entry: {
                'specs/cli/cli.specification.ts':
                    'the runner every `<case>.spec.yaml` document is bound to by the literate plugin; no TypeScript spec imports it',
                'specs/cli/__setup__/github-api.ts':
                    'the GitHub API stub a document serves (`serve: [github]`); the runner spawns it with node, nothing imports it',
                'specs/cli/__setup__/tailscale-api.ts':
                    'the Tailscale API stub a document serves (`serve: tailscale`); the runner spawns it with node, nothing imports it',
            },
            ignoreBinaries: {
                go: 'the CLI specification builds the Go binary it drives; the toolchain is a machine requirement, never an npm dependency',
            },
        },
        oxfmt: {
            ignore: {
                'TODO.md': 'copied verbatim from the design',
                'apps/menubar/stories/STORIES.md':
                    'copied verbatim from the design hand-off (v4), its captures left in the design folder',
                'apps/menubar/stories/INDEX.md':
                    'written by `StoryCoverageTests` from the story files, byte for byte what its drift test compares',
                'apps/menubar/stories/pages/**':
                    'written by `make generate` from the story worlds, byte for byte what the story pages drift test compares',
                'specs/usecases/worlds/stories.json':
                    "written by `make generate` from the stories' scene, byte for byte what the play world drift test compares",
                'src/remote/api/calls.json':
                    'written by `make generate` from the call registry, byte for byte what the calls-json-fresh test compares',
            },
        },
        roots: {
            scripts: 'the curl-piped installer a fresh machine runs before `station` exists on it',
        },
    },
});

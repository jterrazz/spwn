import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    presets: [test()],
    typescript: {
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
                'apps/engine/remote/api/calls.json':
                    'written by `make generate` from the call registry, byte for byte what the calls-json-fresh test compares',
                'packages/api/schemas/**':
                    "written by `make generate` from the catalog's Go types, byte for byte what the catalog schema drift test compares",
            },
        },
        roots: {
            catalog:
                'the built-in packages and machines the engine embeds through the catalog module',
            scripts: 'the curl-piped installer a fresh machine runs before `station` exists on it',
        },
    },
});

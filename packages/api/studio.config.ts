import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    kind: 'library',
    platform: 'node',
    typescript: {
        oxlint: {
            overrides: [
                {
                    files: ['index.d.ts'],
                    reason: "written by `make generate` in the schemas' order, one file a client imports whole, byte for byte what the packages-api drift test compares",
                    rules: {
                        'eslint/max-lines': 'off',
                        'jterrazz-typescript/comment-states-its-kind': 'off',
                        'perfectionist/sort-union-types': 'off',
                    },
                },
                {
                    files: ['words.d.ts'],
                    reason: "written by `make generate` under the generated header its drift test compares, the type of `words.json` a client's import resolves to",
                    rules: {
                        'jterrazz-typescript/comment-states-its-kind': 'off',
                    },
                },
            ],
        },
        roots: {
            schemas:
                "the wire's JSON Schemas `make generate` writes from the engine's Go types, which `index.d.ts` types",
        },
    },
});

import { heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('tool', () => {
    test('a tool page reads its facts and renders its skill file', async () => {
        // Given - the codex runtime, which ships one skill file
        const result = await website.visit('/tools/codex', async (visitor) => {
            await visitor.see(heading('spwn:codex'));
        });

        // Then - the skill's sections, headings and code rendered from its markdown
        expect(result.tree.text).toContain(`  - heading "Usage" [level=3]
  - text: "codex \\"your task here\\" # Interactive mode codex exec \\"your task here\\" # Non-interactive mode codex exec \\"task\\" --full-auto # Full auto with sandboxed writes codex exec \\"task\\" --model gpt-5.4 # Specify model"
  - heading "Configuration" [level=3]
  - paragraph:
    - text: Codex config lives at
    - code: ~/.codex/config.toml`);
    });
});

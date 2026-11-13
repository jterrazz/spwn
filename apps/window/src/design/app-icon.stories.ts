import type { ComponentProps } from 'react';

import type { AppIcon } from './app-icon';
import { SECRET_SAMPLE } from './story';
import type { Story } from './story';

export const APP_ICON_STORIES: readonly Story<ComponentProps<typeof AppIcon>>[] = [
    { name: "the menu bar's mark", props: { name: 'docker' } },
    { name: 'a logo', props: { name: 'github' } },
    { name: 'a logo, through an alias', props: { name: 'gh' } },
    { name: 'a monogram, no mark held', props: { name: 'herdr' } },
    { name: "a setting, its shelf's glyph", props: { kind: 'setting', name: 'git-shortcuts' } },
    { name: "a page's head", props: { name: 'node', size: 'page' } },
    { name: 'long text', props: { name: 'a-package-whose-name-runs-on' } },
    { name: 'empty', props: { name: '' } },
    { name: 'withheld', props: { name: SECRET_SAMPLE } },
];

import type { Story } from '../story';
import { ILLUSTRATIONS } from './illustration';
import type { IllustrationName } from './illustration';

export const ILLUSTRATION_STORIES: readonly Story<{ name: IllustrationName }>[] = ILLUSTRATIONS.map(
    (name) => ({ name, props: { name } }),
);

import type { CopyButtonProps } from './copy-button';
import { SECRET_SAMPLE } from './story';
import type { Story } from './story';

export const COPY_BUTTON_STORIES: readonly Story<CopyButtonProps>[] = [
    { name: 'its own word', props: { name: 'Copy the command', text: 'station engine start' } },
    { name: 'another word', props: { label: 'Copy the address', text: 'http://gate4.local:3000' } },
    { name: 'withheld: no copy at all', props: { name: 'Copy the token', text: SECRET_SAMPLE } },
];

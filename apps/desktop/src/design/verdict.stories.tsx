import { Button } from './button';
import type { Story } from './story';
import type { VerdictProps } from './verdict';

export const VERDICT_STORIES: readonly Story<VerdictProps>[] = [
    {
        name: 'page, ok',
        props: {
            line: 'Every machine answers',
            size: 'page',
            title: 'All is well.',
            tone: 'ok',
            word: 'Up',
        },
    },
    {
        name: 'page, an action',
        props: {
            action: <Button variant="primary">Review</Button>,
            line: '2 things need you',
            size: 'page',
            title: 'Something needs you.',
            tone: 'warn',
            word: 'Needs you',
        },
    },
    {
        name: 'tab, bad',
        props: {
            line: 'relay does not answer',
            size: 'tab',
            title: 'relay is down.',
            tone: 'bad',
            word: 'Down',
        },
    },
    {
        name: 'tab, plain',
        props: {
            line: 'No verdict from the engine.',
            size: 'tab',
            title: 'Stopped.',
            tone: 'plain',
            word: 'Stopped',
        },
    },
    {
        name: 'tab, new',
        props: {
            line: 'Its disk is being made',
            size: 'tab',
            title: 'Creating gate5.',
            tone: 'new',
            word: 'Creating',
        },
    },
    {
        name: 'long text',
        props: {
            line: 'A line long enough to wrap under its title, as an engine sentence sometimes is when it names a path',
            size: 'page',
            title: 'A verdict whose title is long enough to wrap on a narrow window.',
            tone: 'warn',
            word: 'Needs you',
        },
    },
];

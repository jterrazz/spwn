import type { ComponentType, ReactNode } from 'react';

import { Screen } from '../../design/index';
import type { Story } from '../../design/story';

export const LAYERS = [
    'Marks',
    'Measures',
    'Controls',
    'Containers',
    'Statements',
    'Dialogs',
] as const;

export type Layer = (typeof LAYERS)[number];

// value: how an entry's samples are laid out: side by side, one per line, in a column the width of the sidebar,
// value: in a meter's width, or each in a frame that holds a dialog's veil inside it.
export type Frame = 'block' | 'dialog' | 'inline' | 'meter' | 'nav';

export type Sample = { name: string; drawn: ReactNode };

export type Entry = { name: string; layer: Layer; frame: Frame; samples: readonly Sample[] };

type Entered<P> = {
    Component: ComponentType<P>;
    stories: readonly Story<P>[];
    frame?: Frame;
};

export function entry<P extends object>(
    name: string,
    layer: Layer,
    { Component, stories, frame = 'inline' }: Entered<P>,
): Entry {
    return {
        frame,
        layer,
        name,
        samples: stories.map((story) => ({
            drawn: (
                <Screen>
                    <Component {...story.props} />
                </Screen>
            ),
            name: story.name,
        })),
    };
}

export function anchorOf(name: string): string {
    return `entry-${name.toLowerCase()}`;
}

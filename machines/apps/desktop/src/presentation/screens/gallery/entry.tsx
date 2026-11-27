import type { ComponentType, ReactNode } from 'react';

import { Screen } from '../../ui/index';
import type { Story } from '../../ui/story';

export const LAYERS = [
    'Marks',
    'Measures',
    'Controls',
    'Containers',
    'Statements',
    'Dialogs',
] as const;

export type Layer = (typeof LAYERS)[number];

// value: how an entry's samples are laid out: side by side, one per line, in the window's column's width,
// value: in a meter's width, each in a frame that holds a dialog's veil inside it, in one tall enough for a sheet, in
// value: a desk the height of the window's space over its strip, or in the window's whole space under its bar.
export type Frame = 'block' | 'canvas' | 'dialog' | 'inline' | 'meter' | 'nav' | 'sheet' | 'space';

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
                    {story.density === undefined ? (
                        <Component {...story.props} />
                    ) : (
                        <div data-density={story.density}>
                            <Component {...story.props} />
                        </div>
                    )}
                </Screen>
            ),
            name: story.name,
        })),
    };
}

export function anchorOf(name: string): string {
    return `entry-${name.toLowerCase()}`;
}

import type { FieldProps } from './field';
import { Picks } from './picks';
import { nothing } from './story';
import type { Story } from './story';

export const FIELD_STORIES: readonly Story<FieldProps>[] = [
    {
        name: 'a group, its label inside the box',
        props: {
            children: (
                <Picks
                    label="Lifetime"
                    look="cards"
                    onChange={nothing}
                    options={[
                        {
                            id: 'ephemeral',
                            lines: ['Deleted when it powers off'],
                            title: 'ephemeral',
                        },
                        {
                            id: 'persistent',
                            lines: ['Stays until you delete it'],
                            title: 'persistent',
                        },
                    ]}
                    value="persistent"
                />
            ),
            label: 'Lifetime',
        },
    },
    {
        name: 'an input, named by its label',
        props: {
            children: <input id="story-name" readOnly value="gate5" />,
            label: 'Name',
            labelFor: 'story-name',
        },
    },
];

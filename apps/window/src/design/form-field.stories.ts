import type { FormFieldProps } from './form-field';
import { LONG_NAME, nothing } from './story';
import type { Story } from './story';

export const FORM_FIELD_STORIES: readonly Story<FormFieldProps>[] = [
    {
        name: 'empty, its placeholder',
        props: { label: 'Name', onChange: nothing, placeholder: 'gate5', value: '' },
    },
    {
        name: 'filled, with its hint',
        props: {
            hint: 'Lowercase letters, digits and dashes',
            label: 'Name',
            onChange: nothing,
            value: 'gate5',
        },
    },
    {
        name: 'label inside',
        props: { label: 'Type infra to confirm', labelInside: true, onChange: nothing, value: '' },
    },
    { name: 'long text', props: { label: 'Name', onChange: nothing, value: LONG_NAME } },
];

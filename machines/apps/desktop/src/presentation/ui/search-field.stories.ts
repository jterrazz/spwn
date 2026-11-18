import type { SearchFieldProps } from './search-field';
import { LONG_NAME, nothing } from './story';
import type { Story } from './story';

export const SEARCH_FIELD_STORIES: readonly Story<SearchFieldProps>[] = [
    {
        name: 'empty, its placeholder',
        props: { label: 'Filter', onChange: nothing, placeholder: 'Filter the catalog', value: '' },
    },
    {
        name: 'filled',
        props: {
            label: 'Filter',
            onChange: nothing,
            placeholder: 'Filter the catalog',
            value: 'node',
        },
    },
    {
        name: 'long text',
        props: { label: 'Filter', onChange: nothing, placeholder: 'Filter', value: LONG_NAME },
    },
];

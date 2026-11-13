import { Icon } from './icon';

import styles from './search-field.module.css';

export type SearchFieldProps = {
    label: string;
    value: string;
    onChange: (value: string) => void;
    placeholder: string;
};

export function SearchField({ label, value, onChange, placeholder }: SearchFieldProps) {
    return (
        <label className={styles.field}>
            <Icon name="find" />
            <input
                aria-label={label}
                className={styles.input}
                onChange={(event) => {
                    onChange(event.target.value);
                }}
                placeholder={placeholder}
                spellCheck={false}
                type="search"
                value={value}
            />
        </label>
    );
}

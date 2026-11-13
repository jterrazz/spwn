import { useId } from 'react';

import styles from './form-field.module.css';

export type FormFieldProps = {
    label: string;
    value: string;
    onChange: (value: string) => void;
    onEnter?: (() => void) | undefined;
    placeholder?: string;
    hint?: string;
    labelInside?: boolean;
};

export function FormField({
    label,
    value,
    onChange,
    onEnter,
    placeholder,
    hint,
    labelInside = false,
}: FormFieldProps) {
    const id = useId();
    return (
        <div className={styles.form}>
            {labelInside ? null : (
                <label className={styles.label} htmlFor={id}>
                    {label}
                </label>
            )}
            <input
                aria-label={labelInside ? label : undefined}
                autoComplete="off"
                className={styles.input}
                id={id}
                onChange={(event) => {
                    onChange(event.target.value);
                }}
                onKeyDown={(event) => {
                    if (event.key === 'Enter' && !event.nativeEvent.isComposing && onEnter) {
                        event.preventDefault();
                        onEnter();
                    }
                }}
                placeholder={labelInside ? label : placeholder}
                spellCheck={false}
                type="text"
                value={value}
            />
            {(hint ?? '') === '' ? null : <span className={styles.hint}>{hint}</span>}
        </div>
    );
}

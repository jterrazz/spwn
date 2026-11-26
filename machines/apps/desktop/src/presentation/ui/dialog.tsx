import { useEffect, useId, useLayoutEffect, useRef } from 'react';
import type { KeyboardEvent, ReactNode } from 'react';

import { focusablesIn, holdFocus, moveFocus } from './focus';

import styles from './dialog.module.css';

let open = 0;

export function aDialogIsOpen(): boolean {
    return open > 0;
}

export type ModalProps = {
    onClose: () => void;
    children: ReactNode;
    label?: string | undefined;
    labelledBy?: string | undefined;
    className?: string | undefined;
    opensOn?: OpensOn | undefined;
    place?: Place | undefined;
};

type OpensOn = 'dialog' | 'field';

// value: where a modal stands: under the top bar like a sheet of the page, or centred under the bar, which it leaves
// value: uncovered, as the guide of a long act does.
type Place = 'centre' | 'top';

function opening(pop: HTMLElement, on: OpensOn): HTMLElement {
    if (on === 'dialog') {
        return pop;
    }
    return pop.querySelector<HTMLElement>('input:not(:disabled)') ?? focusablesIn(pop)[0] ?? pop;
}

function trapped(event: KeyboardEvent, pop: HTMLElement | null, close: () => void): void {
    if (event.key === 'Escape') {
        event.stopPropagation();
        event.preventDefault();
        close();
        return;
    }
    if (event.key === 'Tab' && pop) {
        event.stopPropagation();
        event.preventDefault();
        moveFocus(focusablesIn(pop), event.shiftKey ? 'previous' : 'next', true);
    }
}

export function Modal({
    onClose,
    children,
    label,
    labelledBy,
    className,
    opensOn = 'field',
    place = 'top',
}: ModalProps) {
    const pop = useRef<HTMLDivElement>(null);
    const close = useRef(onClose);
    const on = useRef(opensOn);
    useEffect(() => {
        close.current = onClose;
    });
    useLayoutEffect(() => {
        open += 1;
        const release = holdFocus(pop.current, (element) => opening(element, on.current));
        return () => {
            open -= 1;
            release();
        };
    }, []);
    return (
        <div
            className={`${styles.veil} ${place === 'centre' ? styles.centre : ''}`}
            onKeyDown={(event) => {
                trapped(event, pop.current, close.current);
            }}
            onMouseDown={(event) => {
                if (event.target === event.currentTarget) {
                    close.current();
                }
            }}
            role="presentation"
        >
            <div
                aria-label={label}
                aria-labelledby={labelledBy}
                aria-modal="true"
                className={`${styles.pop} ${className ?? ''}`}
                ref={pop}
                role="dialog"
                tabIndex={-1}
            >
                {children}
            </div>
        </div>
    );
}

export type DialogProps = {
    title: string;
    lead?: ReactNode;
    onClose: () => void;
    children: ReactNode;
    actions: ReactNode;
    consequence?: string | undefined;
    opensOn?: OpensOn;
};

export function Dialog({
    title,
    lead,
    onClose,
    children,
    actions,
    consequence,
    opensOn,
}: DialogProps) {
    const titleId = useId();
    return (
        <Modal labelledBy={titleId} onClose={onClose} opensOn={opensOn}>
            <div className={styles.sheet}>
                <div className={styles.head}>
                    <h3 className={styles.title} id={titleId}>
                        {lead}
                        {title}
                    </h3>
                    {(consequence ?? '') === '' ? null : (
                        <p className={styles.consequence}>{consequence}</p>
                    )}
                </div>
                {children}
                <div className={styles.actions}>{actions}</div>
            </div>
        </Modal>
    );
}

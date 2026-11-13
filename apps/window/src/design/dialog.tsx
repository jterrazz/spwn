import { useEffect, useId, useLayoutEffect, useRef } from 'react';
import type { KeyboardEvent, ReactNode } from 'react';

import { focusablesIn, moveFocus } from './focus';

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
};

type OpensOn = 'dialog' | 'field';

function opening(pop: HTMLElement | null, on: OpensOn): HTMLElement | undefined {
    if (!pop) {
        return undefined;
    }
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
}: ModalProps) {
    const pop = useRef<HTMLDivElement>(null);
    const close = useRef(onClose);
    const on = useRef(opensOn);
    useEffect(() => {
        close.current = onClose;
    });
    useLayoutEffect(() => {
        const opener = document.activeElement;
        open += 1;
        opening(pop.current, on.current)?.focus();
        return () => {
            open -= 1;
            if (opener instanceof HTMLElement && opener.isConnected) {
                opener.focus();
            }
        };
    }, []);
    return (
        <div
            className={styles.veil}
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
    onClose: () => void;
    children: ReactNode;
    actions: ReactNode;
    consequence?: string | undefined;
    opensOn?: OpensOn;
};

export function Dialog({ title, onClose, children, actions, consequence, opensOn }: DialogProps) {
    const titleId = useId();
    return (
        <Modal labelledBy={titleId} onClose={onClose} opensOn={opensOn}>
            <div className={styles.sheet}>
                <div className={styles.head}>
                    <h3 className={styles.title} id={titleId}>
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

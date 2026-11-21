import type { ReactNode } from 'react';

import { Button, IconButton } from '../ui/index';

import styles from './toolbar.module.css';

// value: `shown` draws the last crumb when a page names its item from the engine's answer.
export type Crumb = { label: string; shown?: ReactNode; open?: (() => void) | undefined };

// value: the one button at the toolbar's right: Home's New machine, the Passwords page's Add a password.
export type ToolbarAction = { label: string; onClick: () => void; leads: boolean };

export type ToolbarProps = {
    crumbs: readonly Crumb[];
    canGoBack: boolean;
    canGoForward: boolean;
    onBack: () => void;
    onForward: () => void;
    onToggleSidebar: () => void;
    action?: ToolbarAction | undefined;
    mark?: ReactNode;
    sidebar?: boolean;
};

export function Toolbar({
    crumbs,
    canGoBack,
    canGoForward,
    onBack,
    onForward,
    onToggleSidebar,
    action,
    mark,
    sidebar = true,
}: ToolbarProps) {
    const last = crumbs.at(-1);
    return (
        <header
            className={`${styles.toolbar} ${sidebar ? '' : styles.bare}`}
            data-tauri-drag-region="deep"
        >
            <IconButton icon="sidebar" label="Show or hide the sidebar" onClick={onToggleSidebar} />
            <IconButton
                disabled={!canGoBack}
                icon="back"
                label="Back"
                onClick={onBack}
                shortcut="⌘["
            />
            <IconButton
                disabled={!canGoForward}
                icon="forward"
                label="Forward"
                onClick={onForward}
                shortcut="⌘]"
            />
            <div className={styles.title}>
                {crumbs.slice(0, -1).map((crumb) => (
                    <span key={crumb.label}>
                        <button className={styles.crumb} onClick={crumb.open} type="button">
                            {crumb.label}
                        </button>
                        <span aria-hidden="true"> › </span>
                    </span>
                ))}
                <b>{last?.shown ?? last?.label}</b>
            </div>
            {mark}
            {action === undefined ? null : (
                <Button onClick={action.onClick} variant={action.leads ? 'primary' : 'plain'}>
                    {action.label}
                </Button>
            )}
        </header>
    );
}

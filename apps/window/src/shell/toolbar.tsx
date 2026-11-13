import { Button, IconButton } from '../design/index';

import styles from './toolbar.module.css';

export type Crumb = { label: string; open?: (() => void) | undefined };

export type ToolbarProps = {
    crumbs: readonly Crumb[];
    canGoBack: boolean;
    canGoForward: boolean;
    onBack: () => void;
    onForward: () => void;
    onToggleSidebar: () => void;
    onNewMachine?: (() => void) | undefined;
    newMachineLeads?: boolean;
    sidebar?: boolean;
};

export function Toolbar({
    crumbs,
    canGoBack,
    canGoForward,
    onBack,
    onForward,
    onToggleSidebar,
    onNewMachine,
    newMachineLeads = true,
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
                <b>{last?.label}</b>
            </div>
            {onNewMachine === undefined ? null : (
                <Button onClick={onNewMachine} variant={newMachineLeads ? 'primary' : 'plain'}>
                    New machine
                </Button>
            )}
        </header>
    );
}

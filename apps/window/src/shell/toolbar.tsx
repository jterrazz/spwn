import { Button, IconButton } from '../design/index';

import styles from './toolbar.module.css';

export type Crumb = { label: string; open?: (() => void) | undefined };

export type ToolbarProps = {
    crumbs: readonly Crumb[];
    canGoBack: boolean;
    onBack: () => void;
    onToggleSidebar: () => void;
    onFind: () => void;
    onNewMachine?: (() => void) | undefined;
    newMachineLeads?: boolean;
    sidebar?: boolean;
};

export function Toolbar({
    crumbs,
    canGoBack,
    onBack,
    onToggleSidebar,
    onFind,
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
            <IconButton icon="find" label="Find" onClick={onFind} shortcut="⌘K" />
            {onNewMachine === undefined ? null : (
                <Button onClick={onNewMachine} variant={newMachineLeads ? 'primary' : 'plain'}>
                    New machine
                </Button>
            )}
        </header>
    );
}

import { useEffect } from 'react';
import type { ComponentType, ReactNode } from 'react';

import { useCore, useReach } from '../dependencies/dependencies';
import { Screen } from '../ui/index';
import { Boundary } from './boundary';
import { TITLES } from './crumbs';
import { Toolbar } from './toolbar';

import styles from './window.module.css';

const NOWHERE = () => undefined;

export function useQuiet(): boolean {
    const { kind } = useReach();
    const { flow } = useCore();
    useEffect(() => {
        if (kind === 'unanswered') {
            flow.enter();
        } else if (kind === 'answering') {
            flow.answered();
        }
    }, [kind, flow]);
    return kind === 'unanswered';
}

type NoEngineProps = {
    Core: ComponentType;
    sidebar: ReactNode;
    onToggleSidebar: () => void;
};

export function NoEngine({ Core, sidebar, onToggleSidebar }: NoEngineProps) {
    return (
        <Screen>
            <div className={`${styles.window} ${sidebar === null ? styles.collapsed : ''}`}>
                {sidebar}
                <main className={styles.main}>
                    <Toolbar
                        canGoBack={false}
                        canGoForward={false}
                        crumbs={[{ label: TITLES.home }]}
                        onBack={NOWHERE}
                        onForward={NOWHERE}
                        onToggleSidebar={onToggleSidebar}
                        sidebar={sidebar !== null}
                    />
                    <div className={styles.scroll}>
                        <Boundary where="Home">
                            <Core />
                        </Boundary>
                    </div>
                </main>
            </div>
        </Screen>
    );
}

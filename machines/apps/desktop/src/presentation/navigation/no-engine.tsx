import { useEffect, useState } from 'react';
import type { ComponentType } from 'react';

import { useCore, useReach } from '../dependencies/dependencies';
import { Boundary } from './boundary';

import styles from './window.module.css';

// invariant: the frame holds the no-engine panel only while the link is declared absent and it never drew the
// invariant: engine's machines: once it has, an absence keeps every page drawn and the sidebar's foot says it.
export function useQuiet(drawn: boolean): boolean {
    const { kind } = useReach();
    const { flow } = useCore();
    const [shown, setShown] = useState(drawn);
    if (drawn && !shown) {
        setShown(true);
    }
    useEffect(() => {
        if (kind === 'unanswered') {
            flow.enter();
        } else if (kind === 'answering') {
            flow.answered();
        }
    }, [kind, flow]);
    return kind === 'unanswered' && !shown && !drawn;
}

export function NoEngine({ Core }: { Core: ComponentType }) {
    return (
        <div className={styles.scroll}>
            <Boundary where="Home">
                <Core />
            </Boundary>
        </div>
    );
}

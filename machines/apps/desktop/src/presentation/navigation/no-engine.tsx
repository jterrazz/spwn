import { useEffect, useState } from 'react';
import type { ComponentType } from 'react';

import { useCore, useReach } from '../dependencies/dependencies';
import { Boundary } from './boundary';

import styles from './window.module.css';

export type Held = 'core' | 'verdict';

// invariant: the frame holds the no-engine panel only while the link is declared absent and it never drew the
// invariant: engine's machines: once it has, an absence keeps every page drawn and the sidebar's foot says it. An
// invariant: engine judged incompatible holds the verdict's panel whatever was drawn: its pages are not this window's.
export function useHeld(drawn: boolean): Held | undefined {
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
    if (kind === 'incompatible') {
        return 'verdict';
    }
    return kind === 'unanswered' && !shown && !drawn ? 'core' : undefined;
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

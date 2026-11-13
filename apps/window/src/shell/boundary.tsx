import { Component, useState } from 'react';
import type { ReactNode } from 'react';

import { Button, Facts, Panel, Value, Verdict } from '../design/index';
import { ownState } from '../engine/shown';

import styles from './window.module.css';

type CatcherProps = { where: string; children: ReactNode; onReload: () => void };

type CatcherState = { error: Error | undefined };

function errorOf(thrown: unknown): Error {
    return thrown instanceof Error ? thrown : new Error(String(thrown));
}

class Catcher extends Component<CatcherProps, CatcherState> {
    constructor(props: CatcherProps) {
        super(props);
        this.state = { error: undefined };
    }

    static getDerivedStateFromError(thrown: unknown): CatcherState {
        return { error: errorOf(thrown) };
    }

    override render(): ReactNode {
        const { error } = this.state;
        const { children, onReload, where } = this.props;
        return error === undefined ? (
            children
        ) : (
            <Broken error={error} onReload={onReload} where={where} />
        );
    }
}

export type BoundaryProps = {
    where: string;
    children: ReactNode;
    onReload?: (() => void) | undefined;
};

export function Boundary({ where, children, onReload }: BoundaryProps) {
    const [round, setRound] = useState(0);
    const reload =
        onReload ??
        (() => {
            setRound((was) => was + 1);
        });
    return (
        <Catcher key={round} onReload={reload} where={where}>
            {children}
        </Catcher>
    );
}

type BrokenProps = { error: Error; where: string; onReload: () => void };

function Broken({ error, where, onReload }: BrokenProps) {
    return (
        <div className={styles.broken}>
            <Verdict
                line="An error of the window, not an answer of the engine"
                size="tab"
                title={`The window could not draw ${where}.`}
                {...ownState('window-error')}
            />
            <Panel action={<Button onClick={onReload}>Reload</Button>} title="What broke">
                <Facts
                    facts={[
                        {
                            label: 'Error',
                            value: (
                                <Value>{error.message === '' ? error.name : error.message}</Value>
                            ),
                        },
                        { label: 'Route', value: <Value>{where}</Value> },
                    ]}
                    layout="column"
                />
            </Panel>
        </div>
    );
}

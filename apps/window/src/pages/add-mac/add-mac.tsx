import type { CallName, MachinesAdded } from '@jterrazz/station-api';
import { useState } from 'react';

import { Button, Dialog, FormField, Screen } from '../../design/index';
import { useGesture } from '../../engine/use-gesture';
import { machineRoute } from '../../shell/route';
import type { Route } from '../../shell/route';
import { GestureButton, GestureOutcome } from '../kit/gestures';
import { addInputOf, EMPTY_MAC } from './model';
import type { MacDraft } from './model';

import styles from './add-mac.module.css';

const ADD_REREADS: readonly CallName[] = ['machines.list'];

export type AddMacProps = { go: (to: Route) => void; onClose: () => void };

type Change = (next: Partial<MacDraft>) => void;

function Login({ draft, change }: { draft: MacDraft; change: Change }) {
    const [open, setOpen] = useState(false);
    return (
        <div className={styles.login}>
            <button
                aria-expanded={open}
                className={styles.toggle}
                onClick={() => {
                    setOpen((before) => !before);
                }}
                type="button"
            >
                <span aria-hidden="true">{open ? '⌄' : '›'}</span> Login and key
            </button>
            {open ? (
                <div className={styles.fields}>
                    <FormField
                        hint="Left empty, yours on this Mac."
                        label="Its login"
                        onChange={(user) => {
                            change({ user });
                        }}
                        value={draft.user}
                    />
                    <FormField
                        hint="Left empty, the key ssh offers by default."
                        label="Its SSH key, a path"
                        onChange={(key) => {
                            change({ key });
                        }}
                        value={draft.key}
                    />
                </div>
            ) : null}
        </div>
    );
}

export function AddMac({ go, onClose }: AddMacProps) {
    const [draft, setDraft] = useState<MacDraft>(EMPTY_MAC);
    const change: Change = (next) => {
        setDraft((before) => ({ ...before, ...next }));
    };
    const name = draft.name.trim();
    const gesture = useGesture('machines.add', {
        input: addInputOf(draft),
        onDone: (out: MachinesAdded) => {
            onClose();
            go(machineRoute(out.Name));
        },
        rereads: ADD_REREADS,
        subject: name,
    });
    return (
        <Screen>
            <Dialog
                actions={
                    <>
                        <Button onClick={onClose}>Cancel</Button>
                        {name === '' ? (
                            <Button disabled variant="primary">
                                Add
                            </Button>
                        ) : (
                            <GestureButton
                                gesture={gesture}
                                label="Add"
                                title={`Add ${name}?`}
                                variant="primary"
                            />
                        )}
                    </>
                }
                consequence="A Mac of yours on Tailscale where station runs."
                onClose={onClose}
                title="Add a Mac"
            >
                <FormField
                    label="Its Tailscale name"
                    onChange={(typed) => {
                        change({ name: typed });
                    }}
                    placeholder="jterrazz-mac-mini"
                    value={draft.name}
                />
                <Login change={change} draft={draft} />
                <div className={styles.outcome}>
                    <GestureOutcome gesture={gesture} />
                </div>
            </Dialog>
        </Screen>
    );
}

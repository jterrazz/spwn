import type { CallName, MachinesCreated } from '@jterrazz/station-api';
import { useState } from 'react';
import type { ReactNode } from 'react';

import { Button, Dialog, FormField, Missing, Screen, Switch } from '../../design/index';
import { LIFETIMES } from '../../engine/provisional';
import type { Lifetime } from '../../engine/provisional';
import { useRead } from '../../engine/react';
import { useGesture } from '../../engine/use-gesture';
import { LIFETIME_WORDS } from '../../engine/words';
import { machineRoute } from '../../shell/route';
import type { Route } from '../../shell/route';
import { GestureButton, GestureOutcome } from '../kit/gestures';
import { EMPTY_DRAFT, imagesOf, serverInputOf, sizeIsWhole, workspaceInputOf } from './model';
import type { Draft, Kind } from './model';

import styles from './new-machine.module.css';

const CREATE_REREADS: readonly CallName[] = ['machines.list', 'overview.read'];
const KINDS: readonly { id: Kind; label: string }[] = [
    { id: 'workspace', label: 'Workspace' },
    { id: 'server', label: 'Server' },
];

const LIFETIME_OPTIONS = LIFETIMES.map((id) => ({ id, label: LIFETIME_WORDS[id] }));

type Change = (next: Partial<Draft>) => void;

function Choice({ label, children }: { label: string; children: ReactNode }) {
    return (
        <div className={styles.field}>
            <span className={styles.label}>{label}</span>
            {children}
        </div>
    );
}

type ImagesProps = { images: readonly string[] | undefined; draft: Draft; change: Change };

function Images({ images, draft, change }: ImagesProps) {
    if (images === undefined) {
        return null;
    }
    if (images.length === 0) {
        return (
            <Choice label="Image">
                <Missing>
                    images.list names no image on this Mac: the engine is asked with none.
                </Missing>
            </Choice>
        );
    }
    return (
        <Choice label="Image">
            <Switch
                label="Image"
                onChange={(image) => {
                    change({ image });
                }}
                options={images.map((id) => ({ id, label: id }))}
                surface="panel"
                value={draft.image ?? images[0] ?? ''}
            />
        </Choice>
    );
}

function Lifetimes({ draft, change }: { draft: Draft; change: Change }) {
    if (draft.kind === 'server') {
        return null;
    }
    return (
        <Choice label="Lifetime">
            <Switch
                all="The engine's default"
                label="Lifetime"
                onChange={(lifetime: Lifetime | null) => {
                    change({ lifetime });
                }}
                options={LIFETIME_OPTIONS}
                surface="panel"
                value={draft.lifetime}
            />
        </Choice>
    );
}

function Size({ draft, change }: { draft: Draft; change: Change }) {
    return (
        <div className={styles.size}>
            <FormField
                label="CPUs"
                onChange={(cpus) => {
                    change({ cpus });
                }}
                placeholder="its default"
                value={draft.cpus}
            />
            <FormField
                label="Memory, GB"
                onChange={(memoryGB) => {
                    change({ memoryGB });
                }}
                placeholder="its default"
                value={draft.memoryGB}
            />
            <FormField
                label="Disk, GB"
                onChange={(diskGB) => {
                    change({ diskGB });
                }}
                placeholder="its default"
                value={draft.diskGB}
            />
        </div>
    );
}

function Fields({ draft, images, change }: ImagesProps) {
    return (
        <>
            <Choice label="Kind">
                <Switch
                    label="Kind"
                    onChange={(kind) => {
                        change({ kind });
                    }}
                    options={KINDS}
                    surface="panel"
                    value={draft.kind}
                />
            </Choice>
            <FormField
                {...(draft.kind === 'workspace'
                    ? { hint: 'Left empty, the engine names it.' }
                    : {})}
                label="Name"
                onChange={(name) => {
                    change({ name });
                }}
                placeholder="gate5"
                value={draft.name}
            />
            <Images change={change} draft={draft} images={images} />
            <Lifetimes change={change} draft={draft} />
            <Size change={change} draft={draft} />
            <Choice label="Its machine file">
                <Missing>
                    Neither create call takes a machine file: which one the new machine follows is
                    not asked here.
                </Missing>
            </Choice>
        </>
    );
}

export type NewMachineProps = { go: (to: Route) => void; onClose: () => void };

export function NewMachine({ go, onClose }: NewMachineProps) {
    const images = useRead('images.list', {});
    const listed = images.kind === 'read' ? imagesOf(images.out) : undefined;
    const [draft, setDraft] = useState<Draft>(EMPTY_DRAFT);
    const asked: Draft = { ...draft, image: draft.image ?? listed?.[0] ?? null };
    const onDone = (out: MachinesCreated) => {
        onClose();
        go(machineRoute(out.VM.Name));
    };
    const shared = { onDone, rereads: CREATE_REREADS, subject: asked.name.trim() };
    const workspace = useGesture('machines.create-workspace', {
        ...shared,
        input: workspaceInputOf(asked),
    });
    const server = useGesture('machines.create-server', { ...shared, input: serverInputOf(asked) });
    const gesture = draft.kind === 'server' ? server : workspace;
    const change: Change = (next) => {
        setDraft((before) => ({ ...before, ...next }));
    };
    const shown = asked.name.trim() === '' ? `a ${draft.kind}` : asked.name.trim();
    return (
        <Screen>
            <Dialog
                actions={
                    <>
                        <Button onClick={onClose}>Cancel</Button>
                        {sizeIsWhole(draft) ? (
                            <GestureButton
                                gesture={gesture}
                                label="Create"
                                title={`Create ${shown}?`}
                                variant="primary"
                            />
                        ) : (
                            <Button disabled variant="primary">
                                Create
                            </Button>
                        )}
                    </>
                }
                onClose={onClose}
                title="New machine on this Mac"
            >
                <Fields change={change} draft={draft} images={listed} />
                <div className={styles.outcome}>
                    <GestureOutcome gesture={gesture} />
                </div>
            </Dialog>
        </Screen>
    );
}

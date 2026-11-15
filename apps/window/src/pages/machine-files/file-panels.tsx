import type { MachinefileFileView } from '@jterrazz/station-api';

import { Icon } from '../../design/icon';
import { CodeBlock, Missing, Panel, Row, Switch, Value } from '../../design/index';
import { countOf } from '../../engine/engine-words';
import { followerState } from '../../engine/shown';
import { machineRoute } from '../../shell/route';
import type { Route } from '../../shell/route';
import type { Shown } from './file-head';
import { extendedBy, linesOf } from './model';

import styles from './machine-files.module.css';

export function Text({
    view,
    shown,
    show,
}: {
    view: MachinefileFileView;
    shown: Shown;
    show: (shown: Shown) => void;
}) {
    const { Local } = view;
    const own = shown === 'local' && Local !== undefined;
    const path = own ? Local.Path : view.Path;
    return (
        <Panel
            action={
                Local === undefined ? undefined : (
                    <Switch
                        label="Which file"
                        onChange={show}
                        options={[
                            { id: 'yaml', label: view.Name },
                            { id: 'local', label: 'This Mac' },
                        ]}
                        surface="panel"
                        value={own ? 'local' : 'yaml'}
                    />
                )
            }
            note="exactly what is on disk, read only here"
            title={path.split('/').at(-1) ?? path}
        >
            <CodeBlock label="File text" lines={linesOf(own ? Local.Text : view.Text)} numbered />
        </Panel>
    );
}

export function Own({ view }: { view: MachinefileFileView }) {
    const { Local } = view;
    if (Local === undefined) {
        return null;
    }
    return (
        <Panel note={Local.Path} title="This Mac's own changes">
            <div className={styles.rows}>
                {(Local.AppsAdded ?? []).map((name) => (
                    <Row
                        key={`added-${name}`}
                        kind="added"
                        sentence="an app added on this Mac only"
                        size="slim"
                        title={name}
                    />
                ))}
                {(Local.AppsRemoved ?? []).map((name) => (
                    <Row
                        key={`removed-${name}`}
                        kind="removed"
                        sentence="an app the files declare, left out here"
                        size="slim"
                        title={name}
                    />
                ))}
                {(Local.SettingsSet ?? []).map((one) => (
                    <Row
                        key={`set-${one.Setting}-${one.Option}`}
                        sentence={`${one.Option} set here`}
                        size="slim"
                        title={one.Setting}
                        value={<Value>{one.Value}</Value>}
                    />
                ))}
                {(Local.SettingsRemoved ?? []).map((name) => (
                    <Row
                        key={`gone-${name}`}
                        kind="removed"
                        sentence="a setting the files declare, left out here"
                        size="slim"
                        title={name}
                    />
                ))}
            </div>
        </Panel>
    );
}

export function Followers({ view, go }: { view: MachinefileFileView; go: (to: Route) => void }) {
    const followers = view.Followers ?? [];
    return (
        <Panel
            {...(followers.length === 0 ? {} : { note: view.Verdict.Sentence })}
            title="Machines that follow it"
        >
            {followers.length === 0 ? (
                <Missing>{view.Verdict.Sentence}</Missing>
            ) : (
                <div className={styles.rows}>
                    {followers.map((follower) => (
                        <Row
                            key={`${follower.Mac}-${follower.Machine}`}
                            onOpen={() => {
                                go(machineRoute(follower.Machine, 'Packages'));
                            }}
                            sentence={
                                follower.Machine === follower.Mac ? undefined : `on ${follower.Mac}`
                            }
                            size="slim"
                            state={{ ...followerState(follower), calm: 'pill' }}
                            title={follower.Machine}
                        />
                    ))}
                </div>
            )}
        </Panel>
    );
}

export function Users({
    files,
    name,
    choose,
}: {
    files: readonly MachinefileFileView[];
    name: string;
    choose: (file: string) => void;
}) {
    const users = extendedBy(files, name);
    return (
        <Panel note={String(users.length)} title="Files that extend it">
            <div className={styles.rows}>
                {users.map((user) => (
                    <Row
                        key={user.Name}
                        lead={<Icon name="file" />}
                        onOpen={() => {
                            choose(user.Name);
                        }}
                        sentence={user.Description}
                        size="slim"
                        title={user.Name}
                        value={countOf('machine', (user.Followers ?? []).length)}
                    />
                ))}
            </div>
        </Panel>
    );
}

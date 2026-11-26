import type { MachinefileFileView } from '@jterrazz/station-api';

import { sentence } from '../../../domain/format';
import { dayAndClock } from '../../../domain/when';
import { machineRoute } from '../../navigation/route';
import type { Route } from '../../navigation/route';
import { Icon } from '../../ui/icon';
import { CodeBlock, Missing, Panel, Row, Switch, Value } from '../../ui/index';
import { countOf } from '../../words/engine-words';
import { followerState } from '../../words/shown';
import { THE_MAC_ITSELF_WORD } from '../../words/words';
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
        <Panel note={Local.Path.split('/').at(-1)} title="This Mac's own changes">
            <div className={styles.rows}>
                {(Local.AppsAdded ?? []).map((name) => (
                    <Row
                        key={`added-${name}`}
                        kind="added"
                        sentence="an app added on this Mac only"
                        title={name}
                    />
                ))}
                {(Local.AppsRemoved ?? []).map((name) => (
                    <Row
                        key={`removed-${name}`}
                        kind="removed"
                        sentence="an app the files declare, left out here"
                        title={name}
                    />
                ))}
                {(Local.SettingsSet ?? []).map((one) => (
                    <Row
                        key={`set-${one.Setting}-${one.Option}`}
                        sentence={`${one.Option} set here`}
                        title={one.Setting}
                        value={<Value>{one.Value}</Value>}
                    />
                ))}
                {(Local.SettingsRemoved ?? []).map((name) => (
                    <Row
                        key={`gone-${name}`}
                        kind="removed"
                        sentence="a setting the files declare, left out here"
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
            {view.Followers === null ? (
                <Missing>{view.Verdict.Sentence}</Missing>
            ) : followers.length === 0 ? (
                <p className={styles.none}>{view.Verdict.Sentence}</p>
            ) : (
                <div className={styles.rows}>
                    {followers.map((follower) => (
                        <Row
                            key={`${follower.Mac}-${follower.Machine}`}
                            onOpen={() => {
                                go(machineRoute(follower.Machine, 'Packages'));
                            }}
                            sentence={sentence([
                                follower.Machine === follower.Mac
                                    ? THE_MAC_ITSELF_WORD
                                    : `on ${follower.Mac}`,
                                appliedOf(follower.Applied),
                            ])}
                            state={followerState(follower)}
                            title={follower.Machine}
                        />
                    ))}
                </div>
            )}
        </Panel>
    );
}

function appliedOf(at: string | undefined): string | undefined {
    const when = dayAndClock(at);
    return when === undefined ? undefined : `applied ${when}`;
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
                        title={user.Name}
                        value={countOf('machine', (user.Followers ?? []).length)}
                    />
                ))}
            </div>
        </Panel>
    );
}

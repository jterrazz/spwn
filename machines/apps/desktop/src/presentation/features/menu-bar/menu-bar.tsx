import { useEffect } from 'react';

import type { MenuBarState } from '../../../application/use-cases/menu-bar-flow';
import { useMenuBar } from '../../dependencies/dependencies';
import { Panel, ProgressLine, Row, Switch } from '../../ui/index';

export const MENU_BAR_WORDS = {
    approval: 'macOS waits for you: allow Spwn in System Settings, General, Login Items.',
    hidden: 'Spwn Menu is off: it is not in the menu bar.',
    off: 'Off',
    on: 'On',
    panel: 'Menu bar',
    sentence: 'Spwn Menu, at every login',
    show: 'Show in menu bar',
    unread: 'The window could not read whether the menu bar is shown.',
    unswitched: 'The menu bar could not be switched.',
} as const;

type Shown = Exclude<
    MenuBarState,
    { kind: 'refused' } | { kind: 'unknown' } | { kind: 'unsupported' }
>;

function Below({ state }: { state: Shown }) {
    if (state.kind === 'failed') {
        return <ProgressLine detail={state.message} failed task={MENU_BAR_WORDS.unswitched} />;
    }
    if (state.kind !== 'held') {
        return null;
    }
    if (state.held === 'approval') {
        return <span>{MENU_BAR_WORDS.approval}</span>;
    }
    return state.held === 'hidden' ? <span>{MENU_BAR_WORDS.hidden}</span> : null;
}

// invariant: drawn only once the shell has read the login item, and never in a window that has no switch.
export function MenuBar() {
    const { flow, state } = useMenuBar();
    useEffect(() => {
        flow.look();
    }, [flow]);
    if (state.kind === 'unknown' || state.kind === 'unsupported') {
        return null;
    }
    if (state.kind === 'refused') {
        return (
            <Panel title={MENU_BAR_WORDS.panel}>
                <Row
                    below={
                        <ProgressLine detail={state.message} failed task={MENU_BAR_WORDS.unread} />
                    }
                    title={MENU_BAR_WORDS.show}
                />
            </Panel>
        );
    }
    const switching = state.kind === 'switching';
    const on = switching ? state.to : state.held !== 'hidden';
    return (
        <Panel title={MENU_BAR_WORDS.panel}>
            <Row
                below={<Below state={state} />}
                sentence={MENU_BAR_WORDS.sentence}
                title={MENU_BAR_WORDS.show}
                trail={
                    <Switch
                        disabled={switching}
                        label={MENU_BAR_WORDS.show}
                        onChange={(picked) => {
                            flow.show(picked === 'on');
                        }}
                        options={[
                            { busy: switching && state.to, id: 'on', label: MENU_BAR_WORDS.on },
                            { busy: switching && !state.to, id: 'off', label: MENU_BAR_WORDS.off },
                        ]}
                        surface="panel"
                        value={on ? 'on' : 'off'}
                    />
                }
            />
        </Panel>
    );
}

import { useRead } from '../../dependencies/dependencies';
import { MenuBar } from '../../features/menu-bar/menu-bar';
import { WindowRelease } from '../../features/window-release/window-release';
import { useEngineLine } from '../../navigation/engine-line';
import type { Route } from '../../navigation/route';
import { Button, Panel, Row, StateDot } from '../../ui/index';

import page from '../../ui/page.module.css';

type Go = (to: Route) => void;

export const SETTINGS_WORDS = {
    engine: 'Engine',
    open: 'Open',
    openTitle: 'Open the Engine page: its release and its update',
} as const;

function EnginePanel({ go }: { go: Go }) {
    const engine = useRead('engine.read', {});
    const line = useEngineLine(engine);
    return (
        <Panel title={SETTINGS_WORDS.engine}>
            <Row
                lead={<StateDot {...line.dot} />}
                leadSize="dot"
                title={line.said}
                trail={
                    line.opens ? (
                        <Button
                            onClick={() => {
                                go({ page: 'engine' });
                            }}
                            title={SETTINGS_WORDS.openTitle}
                        >
                            {SETTINGS_WORDS.open}
                        </Button>
                    ) : undefined
                }
            />
        </Panel>
    );
}

// invariant: the app's own settings: what the window decides for itself, never a machine's, which the catalog's
// invariant: Machine settings shelf holds.
export function SettingsPage({ go }: { go: Go }) {
    return (
        <div className={page.page}>
            <WindowRelease />
            <MenuBar />
            <EnginePanel go={go} />
        </div>
    );
}

import { Panel, Verdict } from '../presentation/ui/index';
import { ownState } from '../presentation/words/shown';
import { Page } from './page';

export function PlaceholderPage({ title }: { title: string }) {
    return (
        <Page>
            <Verdict
                line="It fills in when the window reads the engine's stream"
                size="page"
                title={`${title} is not wired yet.`}
                {...ownState('not-wired')}
            />
            <Panel title={title} />
        </Page>
    );
}

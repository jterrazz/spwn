import { Panel, Verdict } from '../ui/index';
import { Page } from '../ui/page';
import { ownState } from '../words/shown';

export function PlaceholderPage({ title }: { title: string }) {
    return (
        <Page density="comfortable">
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

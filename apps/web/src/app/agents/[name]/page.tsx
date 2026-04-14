import AgentProfilePageWrapper from './agent-detail-client';

export function generateStaticParams() {
    return [{ name: '_' }];
}

export default function Page() {
    return <AgentProfilePageWrapper />;
}

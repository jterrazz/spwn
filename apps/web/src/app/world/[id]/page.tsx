import WorldDashboard from './world-detail-client';

export function generateStaticParams() {
    return [{ id: '_' }];
}

export default function Page() {
    return <WorldDashboard />;
}

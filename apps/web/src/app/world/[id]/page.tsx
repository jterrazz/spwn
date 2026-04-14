import WorldDashboard from './world-detail-client';

// Placeholder for static export - actual data loaded client-side via useParams
export function generateStaticParams() {
    return [{ id: '_' }];
}

export default function Page() {
    return <WorldDashboard />;
}

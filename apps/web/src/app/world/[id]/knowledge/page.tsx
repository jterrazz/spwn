import KnowledgePage from './knowledge-client';

export function generateStaticParams() {
    return [{ id: '_' }];
}

export default function Page() {
    return <KnowledgePage />;
}

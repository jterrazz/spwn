import KnowledgePage from './knowledge-client';

// Inherits [id] from parent layout
export function generateStaticParams() {
    return [{ id: '_' }];
}

export default function Page() {
    return <KnowledgePage />;
}

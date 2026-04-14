import ToolDetailPage from './tool-detail-client';

export function generateStaticParams() {
    return [
        { slug: 'unix' },
        { slug: 'git' },
        { slug: 'node' },
        { slug: 'python' },
        { slug: 'build' },
        { slug: 'claude-code' },
        { slug: 'codex' },
        { slug: 'aider' },
        { slug: 'docker-cli' },
        { slug: 'qmd' },
        { slug: 'cli' },
        { slug: 'architect' },
    ];
}

export default function Page() {
    return <ToolDetailPage />;
}

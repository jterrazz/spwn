'use client';

import { useRouter } from 'next/navigation';

import { Page } from '../../components/page';
import { PageHeader } from '../../components/page-header';
import { ToolCard } from '../../components/tool-catalog/tool-card';
import { toolSlug } from '../../domain/tools-catalog';
import { TOOLS } from '../../domain/tools-catalog.data';
import { usePageTitle } from '../../hooks/use-page-title';

export default function ToolsPage() {
    usePageTitle('Tools');
    const router = useRouter();
    const available = TOOLS.filter((t) => t.status === 'available').length;

    return (
        <Page>
            <PageHeader
                description={`${available} tools you can stack to build the perfect world image.`}
                title="Tools"
            />

            <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
                {TOOLS.map((tool) => (
                    <ToolCard
                        key={tool.name}
                        onClick={() => {
                            router.push(`/tools/${toolSlug(tool.name)}`);
                        }}
                        tool={tool}
                    />
                ))}
            </div>
        </Page>
    );
}

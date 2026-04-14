'use client';

import { useParams } from 'next/navigation';
import { useState } from 'react';

import { ExpandingSearch } from '@/components/expanding-search';
import { KnowledgeBrowser } from '@/components/knowledge-browser';
import { Page } from '@/components/page';
import { PageHeader } from '@/components/page-header';
import { usePageTitle } from '@/hooks/use-page-title';

export default function KnowledgePage() {
    usePageTitle('Knowledge');
    const params = useParams();
    const worldId = params.id as string;
    const [searchQuery, setSearchQuery] = useState('');

    return (
        <Page>
            <PageHeader
                actions={
                    <ExpandingSearch
                        onChange={setSearchQuery}
                        placeholder="Search files…"
                        value={searchQuery}
                    />
                }
                description="Shared knowledge base for this world."
                title="Knowledge"
            />
            <KnowledgeBrowser
                onSearchChange={setSearchQuery}
                searchQuery={searchQuery}
                worldId={worldId}
            />
        </Page>
    );
}

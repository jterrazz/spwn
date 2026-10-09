'use client';

import { IconPlus } from '@tabler/icons-react';
import { useState } from 'react';

import { ActionButton } from '../../components/action-button';
import { ConfirmDialog } from '../../components/confirm-dialog';
import { Separator } from '../../components/ds';
import { CreateOrganizationDialog } from '../../components/organizations/create-organization-dialog';
import { OrganizationSection } from '../../components/organizations/organization-section';
import { useOrganizations } from '../../components/organizations/use-organizations';
import { Page } from '../../components/page';
import { PageHeader } from '../../components/page-header';
import { Skeleton } from '../../components/ui/skeleton';
import type { Organization } from '../../domain/model';
import { inBackground } from '../../events/in-background';
import { usePageTitle } from '../../hooks/use-page-title';

function NoOrganizations({ onCreate }: { onCreate: () => void }) {
    return (
        <div className="flex flex-col items-center justify-center py-16 text-center">
            <p className="text-muted-foreground/50 text-sm">No organizations defined yet.</p>
            <button
                className="text-muted-foreground/40 hover:text-foreground/60 mt-3 font-mono text-xs underline underline-offset-2 transition-colors"
                onClick={onCreate}
                type="button"
            >
                Create your first organization
            </button>
        </div>
    );
}

function Organizations({
    organizations,
    onDelete,
}: {
    organizations: Organization[];
    onDelete: (organization: Organization) => void;
}) {
    return (
        <div className="space-y-10">
            {organizations.map((organization, index) => (
                <div key={organization.slug}>
                    <OrganizationSection
                        onDelete={() => {
                            onDelete(organization);
                        }}
                        organization={organization}
                    />
                    {index < organizations.length - 1 && <Separator className="mt-10" />}
                </div>
            ))}
        </div>
    );
}

function Listing({
    loading,
    organizations,
    onCreate,
    onDelete,
}: {
    loading: boolean;
    organizations: Organization[];
    onCreate: () => void;
    onDelete: (organization: Organization) => void;
}) {
    if (loading) {
        return <Skeleton className="h-[300px] w-full rounded-xl" />;
    }
    if (organizations.length === 0) {
        return <NoOrganizations onCreate={onCreate} />;
    }
    return <Organizations onDelete={onDelete} organizations={organizations} />;
}

function DeletePrompt({
    organization,
    deleting,
    onCancel,
    onDelete,
}: {
    organization: Organization;
    deleting: boolean;
    onCancel: () => void;
    onDelete: () => void;
}) {
    return (
        <ConfirmDialog
            busy={deleting}
            confirm="Delete"
            confirming="Deleting…"
            onCancel={onCancel}
            onConfirm={onDelete}
            title="Delete Organization"
        >
            Are you sure you want to delete{' '}
            <span className="text-foreground/70 font-mono">{organization.name}</span>?
        </ConfirmDialog>
    );
}

export default function OrganizationsPage() {
    usePageTitle('Organizations');
    const { deleting, fetchOrganizations, loading, organizations, remove } = useOrganizations();
    const [creating, setCreating] = useState(false);
    const [target, setTarget] = useState<null | Organization>(null);
    const handleCreate = () => {
        setCreating(true);
    };

    return (
        <Page>
            <PageHeader
                actions={
                    <ActionButton
                        compact
                        icon={<IconPlus size={18} stroke={2.4} />}
                        label="New Organization"
                        onClick={handleCreate}
                    />
                }
                description="Define team structures and roles for multi-agent worlds."
                title="Organizations"
            />
            <Listing
                loading={loading}
                onCreate={handleCreate}
                onDelete={setTarget}
                organizations={organizations}
            />
            {creating && (
                <CreateOrganizationDialog
                    onClose={() => {
                        setCreating(false);
                    }}
                    onCreated={inBackground(async () => {
                        setCreating(false);
                        await fetchOrganizations();
                    })}
                />
            )}
            {target !== null && (
                <DeletePrompt
                    deleting={deleting}
                    onCancel={() => {
                        setTarget(null);
                    }}
                    onDelete={inBackground(async () => {
                        await remove(target);
                        setTarget(null);
                    })}
                    organization={target}
                />
            )}
        </Page>
    );
}

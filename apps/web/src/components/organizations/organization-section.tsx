'use client';

import { IconTrash } from '@tabler/icons-react';

import type { Organization } from '../../domain/model';
import { KeyValue, SectionHeader, SubLabel } from '../ds';
import { OrganizationFlow } from './organization-flow';

// invariant: the organization every project starts with, which cannot be deleted
const DEFAULT_SLUG = 'default';

/** One organization: its chart beside its details, with the button that deletes it. */
export function OrganizationSection({
    organization,
    onDelete,
}: {
    organization: Organization;
    onDelete: () => void;
}) {
    const description = organization.description ?? '';
    return (
        <section>
            <div className="mb-6 flex items-center gap-3">
                <SectionHeader className="mb-0 flex-1">{organization.name}</SectionHeader>
                {organization.slug !== DEFAULT_SLUG && (
                    <button
                        aria-label={`Delete ${organization.name}`}
                        className="text-muted-foreground/20 p-1 transition-colors hover:text-red-400/70"
                        onClick={onDelete}
                        type="button"
                    >
                        <IconTrash size={14} />
                    </button>
                )}
            </div>
            <div className="flex flex-col gap-8 lg:flex-row">
                <div className="min-w-0 flex-1">
                    <OrganizationFlow organization={organization} />
                </div>
                <div className="shrink-0 space-y-2 lg:w-52">
                    <SubLabel>Details</SubLabel>
                    <KeyValue label="Slug" value={organization.slug} />
                    {description !== '' && <KeyValue label="Description" value={description} />}
                    <KeyValue label="Roles" value={organization.roles.length} />
                    <KeyValue
                        label="Max Depth"
                        value={Math.max(...organization.roles.map((role) => role.level))}
                    />
                </div>
            </div>
        </section>
    );
}

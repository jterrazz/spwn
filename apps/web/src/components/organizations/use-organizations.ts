'use client';

import { useCallback, useState } from 'react';

import { apiDelete, apiGet } from '../../api/client';
import { optional } from '../../api/optional';
import type { Organization } from '../../domain/model';
import { useBackgroundTask } from '../../hooks/use-background-task';

/** The organizations declared on this host, and the verb that deletes one. */
export function useOrganizations() {
    const [organizations, setOrganizations] = useState<Organization[]>([]);
    const [loading, setLoading] = useState(true);
    const [deleting, setDeleting] = useState(false);

    const fetchOrganizations = useCallback(async () => {
        const listed = await optional(apiGet<null | Organization[]>('/api/organizations'));
        setOrganizations(listed ?? []);
        setLoading(false);
    }, []);
    useBackgroundTask(fetchOrganizations);

    const remove = async (organization: Organization) => {
        setDeleting(true);
        await optional(apiDelete(`/api/organizations/${organization.slug}`));
        setDeleting(false);
        await fetchOrganizations();
    };

    return { deleting, fetchOrganizations, loading, organizations, remove };
}

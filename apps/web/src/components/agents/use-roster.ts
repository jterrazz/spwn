'use client';

import { useCallback, useState } from 'react';

import { apiGet } from '../../api/client';
import { optional } from '../../api/optional';
import type { Team, World } from '../../domain/model';
import { useBackgroundTask } from '../../hooks/use-background-task';
import { rosterOf } from './roster';
import type { ListedAgent, RosterAgent } from './roster';

/** The project's agents placed in their worlds, and its teams, polled every five seconds. */
export function useRoster() {
    const [agents, setAgents] = useState<RosterAgent[]>([]);
    const [teams, setTeams] = useState<Team[]>([]);
    const [loading, setLoading] = useState(true);

    const fetchAll = useCallback(async () => {
        const [worlds, listed, known] = await Promise.all([
            optional(apiGet<World[]>('/api/worlds')),
            optional(apiGet<ListedAgent[]>('/api/agents')),
            optional(apiGet<Team[]>('/api/teams')),
        ]);
        setTeams(known ?? []);
        setAgents(rosterOf(worlds ?? [], listed ?? []));
        setLoading(false);
    }, []);
    useBackgroundTask(fetchAll, 5000);

    return { agents, fetchAll, loading, teams };
}

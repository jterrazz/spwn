'use client';

import { useCallback, useState } from 'react';

import { apiAction, apiGet, apiPut, encPath } from '../../api/client';
import { optional } from '../../api/optional';
import type { AgentProfile, Organization, Team, World } from '../../domain/model';
import { useBackgroundTask } from '../../hooks/use-background-task';
import { useList } from '../../hooks/use-list';

/** An agent's profile and the tree of its mind's files, reloaded on demand. */
export function useAgentProfile(name: string) {
    const [profile, setProfile] = useState<AgentProfile | null>(null);
    const [mindTree, setMindTree] = useState<Record<string, string[]>>({});
    const [loading, setLoading] = useState(true);

    const fetchProfile = useCallback(async () => {
        const path = `/api/agents/${encPath(name)}`;
        const [found, tree] = await Promise.all([
            optional(apiGet<AgentProfile>(path)),
            optional(apiGet<Record<string, string[]>>(`${path}/mind`)),
        ]);
        setProfile(found ?? null);
        setMindTree(
            Object.fromEntries(
                Object.entries(tree ?? {}).map(([layer, files]) => [
                    layer,
                    Array.isArray(files) ? files : [],
                ]),
            ),
        );
        setLoading(false);
    }, [name]);
    useBackgroundTask(fetchProfile);

    return { fetchProfile, loading, mindTree, profile };
}

/** What surrounds an agent: the teams and organizations it may join, the worlds it may be deployed to, and the world it lives in (the one it is deployed to, else the one the address names). */
export function useAgentSurroundings(name: string, worldId: string | undefined) {
    const teams = useList<Team>('/api/teams');
    const worlds = useList<World>('/api/worlds');
    const organizations = useList<Organization>('/api/organizations');
    const [named, setNamed] = useState<null | World>(null);

    const loadNamed = useCallback(async () => {
        if (worldId !== undefined && worldId !== '') {
            setNamed((await optional(apiGet<World>(`/api/worlds/${worldId}`))) ?? null);
        }
    }, [worldId]);
    useBackgroundTask(loadNamed);

    const deployedIn = worlds.find(
        (world) => world.agent === name || world.agents.some((agent) => agent.name === name),
    );
    return { organizations, teams, world: deployedIn ?? named, worlds };
}

/** The agent's own actions (identity edits and the mind's verbs), each reported through `report`. */
export function useAgentActions(
    name: string,
    report: (message: string) => void,
    reload: () => Promise<void>,
) {
    const [running, setRunning] = useState<null | string>(null);

    const saveIdentity = async (field: string, content: string): Promise<boolean> => {
        try {
            await apiPut(`/api/agents/${encPath(name)}/identity`, { content, field });
            report(`${field} updated`);
            await reload();
            return true;
        } catch {
            report(`Error: failed to update ${field}`);
            return false;
        }
    };

    const act = async (action: string, body?: object): Promise<boolean> => {
        setRunning(action);
        const result = await apiAction(`/api/agents/${encPath(name)}/${action}`, body);
        setRunning(null);
        if (!result.ok) {
            report(`Error: ${result.error ?? 'Unknown error'}`);
        }
        return result.ok;
    };

    return { act, handleIdentity: saveIdentity, running };
}

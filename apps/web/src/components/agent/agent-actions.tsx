'use client';

import { IconDownload, IconGitFork, IconPlanet, IconRefresh, IconTrash } from '@tabler/icons-react';

import type { AgentProfile } from '../../domain/model';
import { inBackground } from '../../events/in-background';
import { ActionButton } from '../action-button';

interface AgentActionsProps {
    profile: AgentProfile;
    busy: boolean;
    act: (action: string, body?: object) => Promise<boolean>;
    report: (message: string) => void;
    reload: () => Promise<void>;
    onDeploy: () => void;
    onDelete: () => void;
}

function useMindVerbs({
    profile,
    act,
    report,
    reload,
}: Pick<AgentActionsProps, 'act' | 'profile' | 'reload' | 'report'>) {
    const dream = async () => {
        if (profile.journal.length === 0) {
            report('Nothing to dream about yet - spawn the agent in a world first');
            return;
        }
        if (await act('dream')) {
            report('Dream cycle complete - check playbooks for promoted patterns');
            await reload();
        }
    };
    const fork = async () => {
        // oxlint-disable-next-line no-alert -- the fork target is one free-text name, and a modal for it is a follow-up the page does not have
        const target = globalThis.prompt('Fork target name:');
        if (target !== null && target !== '' && (await act('fork', { target }))) {
            report(`Forked to "${target}"`);
        }
    };
    const exportMind = async () => {
        if (await act('export')) {
            report('Export complete!');
        }
    };

    return { dream, exportMind, fork };
}

/** The verbs of an agent's page: dream, fork, export, deploy, delete. */
export function AgentActions({
    profile,
    busy,
    act,
    report,
    reload,
    onDeploy,
    onDelete,
}: AgentActionsProps) {
    const { dream, exportMind, fork } = useMindVerbs({ act, profile, reload, report });
    const verbs = [
        {
            icon: <IconRefresh size={16} stroke={2.2} />,
            label: 'Dream',
            handleClick: inBackground(dream),
        },
        {
            icon: <IconGitFork size={16} stroke={2.2} />,
            label: 'Fork',
            handleClick: inBackground(fork),
        },
        {
            icon: <IconDownload size={16} stroke={2.2} />,
            label: 'Export',
            handleClick: inBackground(exportMind),
        },
        { icon: <IconPlanet size={16} stroke={2.2} />, label: 'Deploy', handleClick: onDeploy },
    ];

    return (
        <>
            {verbs.map((verb) => (
                <ActionButton
                    compact
                    disabled={busy}
                    icon={verb.icon}
                    key={verb.label}
                    label={verb.label}
                    onClick={verb.handleClick}
                />
            ))}
            <ActionButton
                compact
                danger
                disabled={busy}
                icon={<IconTrash size={16} stroke={2.2} />}
                label="Delete"
                onClick={onDelete}
            />
        </>
    );
}

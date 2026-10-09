'use client';

import {
    IconAlertTriangleFilled,
    IconBookFilled,
    IconBrandGithubFilled,
} from '@tabler/icons-react';

import { useVersion } from '../../hooks/use-version';
import { DockerStatusPill } from '../docker-status-pill';
import { ThemeToggle } from '../theme-toggle';
import { SidebarFooter } from '../ui/sidebar';
import { UpgradeBanner } from '../upgrade-banner';

const LINKS = [
    { href: 'https://spwn.sh/docs', icon: <IconBookFilled size={15} />, label: 'Docs' },
    {
        href: 'https://github.com/jterrazz/spwn',
        icon: <IconBrandGithubFilled size={15} />,
        label: 'GitHub',
    },
    {
        href: 'https://github.com/jterrazz/spwn/issues/new',
        icon: <IconAlertTriangleFilled size={15} />,
        label: 'Feedback',
    },
];

/** The sidebar's foot: an available upgrade, the Docker daemon's health, the outside links, the theme. */
export function SpwnSidebarFooter() {
    const { version } = useVersion();
    return (
        <SidebarFooter>
            {version?.updateAvailable === true &&
                version.latest !== process.env.NEXT_PUBLIC_APP_VERSION && (
                    <UpgradeBanner version={version} />
                )}
            <div className="px-1.5 pb-1">
                <DockerStatusPill />
            </div>
            <div className="flex items-center gap-1 px-1.5 pb-1">
                {LINKS.map((link) => (
                    <a
                        aria-label={link.label}
                        className="text-muted-foreground/30 hover:text-foreground flex h-8 w-8 items-center justify-center rounded-full transition-colors"
                        href={link.href}
                        key={link.label}
                        rel="noreferrer"
                        target="_blank"
                    >
                        {link.icon}
                    </a>
                ))}
                <div className="flex-1" />
                <ThemeToggle />
            </div>
        </SidebarFooter>
    );
}

import type { ReactElement } from 'react';

const PATHS = {
    sidebar: (
        <>
            <rect height="10" rx="2" width="12" x="2" y="3" />
            <path d="M6 3v10" />
        </>
    ),
    find: (
        <>
            <circle cx="7" cy="7" r="4" />
            <path d="M10 10l3 3" />
        </>
    ),
    back: <path d="M9.5 3.5L5 8l4.5 4.5" />,
    forward: <path d="M6.5 3.5L11 8l-4.5 4.5" />,
    home: <path d="M2.5 7.5L8 3l5.5 4.5V13h-11z" />,
    inbox: (
        <>
            <path d="M2.5 9.5h3l1 1.5h3l1-1.5h3" />
            <rect height="10" rx="2" width="11" x="2.5" y="3" />
        </>
    ),
    changes: (
        <>
            <circle cx="8" cy="8" r="5.5" />
            <path d="M8 5v3l2 1.5" />
        </>
    ),
    file: (
        <>
            <path d="M4 2.5h5l3 3v8H4z" />
            <path d="M9 2.5v3h3" />
        </>
    ),
    box: (
        <>
            <path d="M8 2l5.5 3v6L8 14l-5.5-3V5z" />
            <path d="M2.5 5L8 8l5.5-3M8 8v6" />
        </>
    ),
    disc: (
        <>
            <circle cx="8" cy="8" r="5.5" />
            <circle cx="8" cy="8" r="1.5" />
        </>
    ),
    mac: (
        <>
            <rect height="7" rx="1.2" width="10" x="3" y="3.5" />
            <path d="M1.5 12.5h13" />
        </>
    ),
    workspace: <path d="M4 5.5L7 8l-3 2.5M8.5 11H12" />,
    server: (
        <>
            <rect height="4" rx="1" width="10" x="3" y="3" />
            <rect height="4" rx="1" width="10" x="3" y="9" />
        </>
    ),
    apps: (
        <>
            <rect height="4.5" rx="1.2" width="4.5" x="2.5" y="2.5" />
            <rect height="4.5" rx="1.2" width="4.5" x="9" y="2.5" />
            <rect height="4.5" rx="1.2" width="4.5" x="2.5" y="9" />
            <rect height="4.5" rx="1.2" width="4.5" x="9" y="9" />
        </>
    ),
    skills: <path d="M8 2.2l1.5 4.3 4.3 1.5-4.3 1.5L8 13.8 6.5 9.5 2.2 8l4.3-1.5z" />,
    settings: (
        <>
            <path d="M2.5 5h6M11.5 5h2M2.5 11h2M7.5 11h6" />
            <circle cx="10" cy="5" r="1.5" />
            <circle cx="6" cy="11" r="1.5" />
        </>
    ),
    check: <path d="M3.5 8.5l3 3 6-7" />,
    close: <path d="M4 4l8 8M12 4l-8 8" />,
} satisfies Record<string, ReactElement>;

export type IconName = keyof typeof PATHS;

export function Icon({ name }: { name: IconName }) {
    return (
        <svg
            aria-hidden="true"
            fill="none"
            stroke="currentColor"
            strokeLinecap="round"
            strokeLinejoin="round"
            viewBox="0 0 16 16"
        >
            {PATHS[name]}
        </svg>
    );
}

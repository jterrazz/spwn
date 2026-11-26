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
    key: (
        <>
            <circle cx="5.5" cy="10.5" r="3" />
            <path d="M7.6 8.4L13 3M11 5l1.5 1.5" />
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
    gear: (
        <>
            <circle cx="8" cy="8" r="2" />
            <path d="M8 1.9l1.1 1.7 2-.4.5 2 1.9.8-.9 1.9.9 1.9-1.9.8-.5 2-2-.4L8 14.1l-1.1-1.7-2 .4-.5-2-1.9-.8.9-1.9-.9-1.9 1.9-.8.5-2 2 .4z" />
        </>
    ),
    machine: (
        <>
            <rect height="4.6" rx="1.4" width="11" x="2.5" y="2.5" />
            <rect height="4.6" rx="1.4" width="11" x="2.5" y="8.9" />
            <path d="M5 4.8h.01M5 11.2h.01" />
        </>
    ),
    agent: (
        <>
            <circle cx="8" cy="5.4" r="2.6" />
            <path d="M2.9 13.6c.3-2.7 2.4-4.3 5.1-4.3s4.8 1.6 5.1 4.3" />
        </>
    ),
    os: (
        <>
            <circle cx="3.8" cy="4" r="1.7" />
            <circle cx="3.8" cy="12" r="1.7" />
            <circle cx="12.2" cy="8" r="1.7" />
            <path d="M5.5 4h1.7c1.2 0 1.9.7 2.3 1.7l.5 1.1M5.5 12h1.7c1.2 0 1.9-.7 2.3-1.7l.5-1.1" />
        </>
    ),
    engine: <path d="M9 1.8L3.8 9h3.7l-.8 5.2L12.2 7H8.4z" />,
    plus: <path d="M8 3v10M3 8h10" />,
    check: <path d="M3.5 8.5l3 3 6-7" />,
    close: <path d="M4 4l8 8M12 4l-8 8" />,
    chevron: <path d="M6 3.5L10.5 8 6 12.5" />,
    up: <path d="M4 10l4-4 4 4" />,
    choose: <path d="M5.5 6.5L8 4l2.5 2.5M5.5 9.5L8 12l2.5-2.5" />,
    lock: (
        <>
            <rect height="6" rx="1.5" width="9" x="3.5" y="7" />
            <path d="M5.5 7V5a2.5 2.5 0 015 0v2" />
        </>
    ),
    pin: <path d="M5.5 2.5h5M6.5 2.5v3.8L4.2 9.5h7.6L9.5 6.3V2.5M8 9.5v4" />,
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

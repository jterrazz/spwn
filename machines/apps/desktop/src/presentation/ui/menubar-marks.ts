// value: the menu bar app's marks, each the body of an SVG on a 20 px box drawn in #000, as
// value: apps/menubar/scripts/symbols.swift writes them; written by scripts/menubar-marks.sh, never by hand.
export const MENUBAR_MARKS: Readonly<Record<string, string>> = {
    api: '<path fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" d="M7.5 4.5c-1.6 0-2.2.8-2.2 2.2v1.6c0 .9-.5 1.5-1.5 1.7 1 .2 1.5.8 1.5 1.7v1.6c0 1.4.6 2.2 2.2 2.2M12.5 4.5c1.6 0 2.2.8 2.2 2.2v1.6c0 .9.5 1.5 1.5 1.7-1 .2-1.5.8-1.5 1.7v1.6c0 1.4-.6 2.2-2.2 2.2"/>',
    astro: '<path d="M10 2.8l4.6 11.4c-1.3-.7-2.8-1-4.6-1s-3.3.3-4.6 1z" fill="#000"/><path fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" d="M8.4 15.4c.2 1 .8 1.8 1.6 1.8s1.4-.8 1.6-1.8"/>',
    claude: '<g stroke="#000" stroke-width="1.9" stroke-linecap="round"><line x1="10" y1="2.6" x2="10" y2="17.4" transform="rotate(0 10 10)"/><line x1="10" y1="2.6" x2="10" y2="17.4" transform="rotate(30 10 10)"/><line x1="10" y1="2.6" x2="10" y2="17.4" transform="rotate(60 10 10)"/><line x1="10" y1="2.6" x2="10" y2="17.4" transform="rotate(90 10 10)"/><line x1="10" y1="2.6" x2="10" y2="17.4" transform="rotate(120 10 10)"/><line x1="10" y1="2.6" x2="10" y2="17.4" transform="rotate(150 10 10)"/></g>',
    cloudflare:
        '<path d="M5 15h10.4a3 3 0 0 0 .3-6 4.6 4.6 0 0 0-8.8-.8A3.5 3.5 0 0 0 5 15z" fill="#000"/>',
    docker: '<g fill="#000"><rect x="3" y="9" width="2.6" height="2.4"/><rect x="6" y="9" width="2.6" height="2.4"/><rect x="9" y="9" width="2.6" height="2.4"/><rect x="6" y="6.2" width="2.6" height="2.4"/><rect x="9" y="6.2" width="2.6" height="2.4"/><rect x="12" y="9" width="2.6" height="2.4"/><path d="M2 12h15c.8-.4 1.2-1.2 1.4-2 .2 1.6-.6 3-2 3.6-1.6 2.6-4.6 3.4-7.8 3.4-3.4 0-5.8-1.6-6.6-5z"/></g>',
    github: '<g transform="scale(1.25)"><path fill="#000" d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></g>',
    kubernetes:
        '<g fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><circle cx="10" cy="10" r="2.2"/><path d="M10 3v4.8M10 12.2V17M3.3 7.8l4.6 1.5M12.1 10.7l4.6 1.5M5.6 15.6l2.8-3.8M11.6 8.2l2.8-3.8M14.4 15.6l-2.8-3.8M8.4 8.2L5.6 4.4"/></g>',
    node: '<g fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M10 2.8l6.2 3.6v7.2L10 17.2l-6.2-3.6V6.4z"/><path d="M10 7.2v5.6"/></g>',
    npm: '<path fill="#000" fill-rule="evenodd" d="M1.5 6.5h17v6.4h-8.5v1.3H6.2v-1.3H1.5zM3 8v3.4h1.9V9.3h1v2.1h1V8zm4.9 0v4.6h1.9v-1.2h1.9V8zm1.9 1.2h.9v1h-.9zM12.6 8v3.4h1.9V9.3h.9v2.1h1V9.3h.9v2.1h1V8z"/>',
    postgres:
        '<g fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="10" cy="5.4" rx="5" ry="2"/><path d="M5 5.4v9.2c0 1.1 2.2 2 5 2s5-.9 5-2V5.4M5 10c0 1.1 2.2 2 5 2s5-.9 5-2"/></g>',
    storybook:
        '<path fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" d="M13.4 6.4c-.6-.9-1.8-1.4-3.2-1.4-1.8 0-3 .9-3 2.3 0 3.2 6.4 1.8 6.4 5.1 0 1.5-1.4 2.6-3.4 2.6-1.6 0-2.9-.6-3.6-1.7"/>',
    tailscale:
        '<circle cx="4" cy="4" r="2.1" fill="#000" opacity="0.35"/><circle cx="10" cy="4" r="2.1" fill="#000" opacity="0.35"/><circle cx="16" cy="4" r="2.1" fill="#000" opacity="0.35"/><circle cx="4" cy="10" r="2.1" fill="#000" opacity="1.0"/><circle cx="10" cy="10" r="2.1" fill="#000" opacity="1.0"/><circle cx="16" cy="10" r="2.1" fill="#000" opacity="1.0"/><circle cx="4" cy="16" r="2.1" fill="#000" opacity="0.35"/><circle cx="10" cy="16" r="2.1" fill="#000" opacity="1.0"/><circle cx="16" cy="16" r="2.1" fill="#000" opacity="0.35"/>',
    traefik:
        '<g fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><circle cx="10" cy="10" r="6.6"/><path d="M3.4 10h13.2M10 3.4c1.8 1.8 2.7 4 2.7 6.6s-.9 4.8-2.7 6.6c-1.8-1.8-2.7-4-2.7-6.6s.9-4.8 2.7-6.6z"/></g>',
    vite: '<path d="M11 2.5L4.8 11h4.6l-.9 6.5L15.2 9h-4.6z" fill="#000"/>',
    web: '<g fill="none" stroke="#000" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="14" height="12" rx="2.4"/><path d="M3 7.6h14"/></g><circle cx="5.4" cy="5.8" r=".5" fill="#000"/><circle cx="7.2" cy="5.8" r=".5" fill="#000"/>',
};

# web

The spwn dashboard: the Next.js app that `spwn web` starts beside the Go API
server (`apps/api`), and that [`src-tauri/`](src-tauri/) wraps as the desktop
app. Every page reads and drives spwn through that API, whose address the
client takes from `NEXT_PUBLIC_API_URL`.

- `src/app/` — one route per page; a page composes the parts it shows.
- `src/components/` — those parts, one folder per page subject.
- `src/api/` — the client of the Go API, and the readers of its payloads.

How to run and change it is the repository's [Develop](../../docs/guides/develop.md)
guide; what proves it is [Test](../../docs/guides/test.md) — the website
specification under `specs/website/` drives this app in a headless browser.

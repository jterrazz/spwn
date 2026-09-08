# Contributing to spwn

Everything a contributor needs is a chapter of the corpus, mapped by [`docs/README.md`](docs/README.md). This page routes; it holds no manual of its own.

| To…                                                          | Read                                             |
| ------------------------------------------------------------ | ------------------------------------------------ |
| Set up a clone, run the loop, learn which file a change opens | [`docs/02-developing.md`](docs/02-developing.md) |
| Know what proves a change, and how to run each suite          | [`docs/03-testing.md`](docs/03-testing.md)       |
| Understand the layers before moving code between packages     | [`docs/01-architecture.md`](docs/01-architecture.md) |
| Cut a release                                                 | [`docs/04-operating.md`](docs/04-operating.md)   |

```bash
git clone https://github.com/jterrazz/spwn.git
cd spwn
go work sync && pnpm install --frozen-lockfile
make build && make test
```

Run `make` with no arguments for the annotated target list. [`AGENTS.md`](AGENTS.md) — with `CLAUDE.md` symlinked to it — is the same routing for an agent.

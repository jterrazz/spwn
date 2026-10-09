# Contributing to spwn

Everything a contributor needs is a chapter of the corpus, mapped by [`docs/README.md`](docs/README.md). This page routes; it holds no manual of its own.

| To…                                                           | Read                                          |
| ------------------------------------------------------------- | --------------------------------------------- |
| Set up a clone, run the loop, learn which file a change opens | [Develop](docs/guides/develop.md)             |
| Know what proves a change, and how to run each suite          | [Test](docs/guides/test.md)                   |
| Understand the layers before moving code between packages     | [Architecture](docs/concepts/architecture.md) |
| Cut a release                                                 | [Release](docs/guides/release.md)             |

```bash
git clone https://github.com/jterrazz/spwn.git
cd spwn
make install
make build && make test
```

Run `make` with no arguments for the annotated target list. [`AGENTS.md`](AGENTS.md) — with `CLAUDE.md` symlinked to it — is the same routing for an agent.

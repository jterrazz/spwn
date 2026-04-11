# Research Lab

> Same brain, new soul.

A science-themed world with Curie — a patient, note-taking agent who
treats every task as an experiment. This example exists mostly to show
off the `spwn agent fork` flow.

## What's inside

- **World** `research-lab` — 2 CPU, 2 GB, Python + Node, 4h timeout.
- **Agent** `curie` — careful, keeps a structured lab notebook in her
  memory, writes hypotheses and conclusions.

## Try it

```sh
spwn up -c research-lab --agent curie
spwn agent talk curie "investigate why my test suite is flaky"
```

Curie frames the task as an experiment, records observations in her
journal, and when she figures something out, promotes the insight into
a playbook she can reuse next time.

## Forking — the headline feature

After Curie has accumulated some knowledge (a handful of journal
entries and playbooks), fork her:

```sh
spwn agent fork curie darwin
```

You now have two agents:

- **Curie** keeps her current identity and continues her current work.
- **Darwin** starts with a blank identity but **inherits Curie's
  entire mind** — same knowledge, same playbooks, same journal.

Edit `~/.spwn/agents/darwin/identity/` to give Darwin a different
persona (evolutionary biologist, say), and he'll apply Curie's methods
to a different domain.

## Remove

```sh
rm ~/.spwn/worlds/research-lab.yaml
rm -rf ~/.spwn/agents/curie
# and darwin too if you forked
```

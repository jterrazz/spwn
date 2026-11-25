# Boundaries

What Spwn is not, and what each of its parts refuses to be, for anyone weighing a feature that
sits near an edge. The owner confirmed the five boundaries of Spwn on 2026-11-21
([his answer](../blueprint/owner-words.md#why-spwn-exists)); the record is
[ADR-002](./decisions/002-spwn-is-one-stack-of-six-apps-on-one-ladder.md), accepted.

## What Spwn is not

<Refusal id="not-a-replacement" refuses="a replacement for the person" because="AI augments the person and does not replace them" />

<Refusal id="not-a-cloud-service" refuses="a cloud service" because="it runs on your machines, and it collects no telemetry" />

<Refusal id="not-one-vendor" refuses="tied to one vendor" because="an agent runs in any harness: Claude Code or Codex today" />

## What each app refuses

<Group id="machines-refuses" title="What Machines refuses" intent="guard">

<Refusal
id="no-running-agent"
by={ref("product/stack#machines")}
refuses="to know about a running agent: starting, watching, answering and merging agents is not its job"
because="spwn:machines/docs/decisions/052-station-keeps-no-agents.md"
/>

<Refusal
id="no-ungranted-port"
by={ref("product/stack#machines")}
refuses="to share a port nobody granted: a port outside a machine's rights is flagged, never shared"
because="spwn:machines/docs/decisions/035-a-port-is-a-right-with-an-audience.md"
/>

</Group>

<Table id="machines" of="refusal" in={ref("product/boundaries")} where={query({ by: { is: ref("product/stack#machines") } })} columns={["refuses", "because"]} />

The rows of the four apps being made are derived from the convictions; none of those apps has a
design yet, so each row is a constraint on the design to come, not a description of it.

## Rests on

Two rows are this page's reading rather than the owner's word or a record: Agents refusing to
prepare a computer, and the OS refusing to hold facts of its own. Both follow from the spaces; the
owner has not ruled on either.

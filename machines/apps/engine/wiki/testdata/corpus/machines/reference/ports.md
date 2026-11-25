# Ports

`station ports` lists every TCP port that listens on a Mac or a VM, each in the class of who reaches it. Which ports a VM may open, and to whom, are its [rights](#right): a port that starts listening inside them is shared with their audience within seconds, one outside them is flagged and never shared.

## Rights and audiences

A VM's rights are the `ports` of its Machinefile ([Machinefiles](machinefiles.md)): which ports it may open, and to whom.

<Term id="right" means="what a machine may open and to whom, its Machinefile's `ports`: `open`, `to` (its audience), `except`, `audiences`" />

<Term
id="audience"
means="who a right, or the owner's `ports share --to`, shares a port with: local, private or public"
/>

<Group id="what-is-flagged" title="What is flagged" intent="guard">

<Rule
id="flagged-never-shared"
says="A port that listens outside the rights is flagged, never shared."
force="hard"
guards={ref("machines/reference/ports")}
proof="spwn:machines/specs/usecases/engine/share/flags-outside-rights.spec.yaml"
/>

<Decision
id="port-is-a-right"
adr="spwn:machines/docs/decisions/035-a-port-is-a-right-with-an-audience.md"
decides="A port is a right with an audience."
/>

</Group>

A row of a port names its label from the catalog's packages' `ports` (5173 `Vite`, 3000 `Node or Next`, 8081 `Expo`, 5432 `Postgres`) as a hint of what probably serves it, never a decision.

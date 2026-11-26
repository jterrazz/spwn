# The stack

Spwn's three spaces and what lives in each, for anyone deciding what a part of the app is for or
where it goes: per space, what you let go of, what you keep, and whether it runs today. The
structure is the owner's, decided on 2026-11-22 ([his words](../blueprint/owner-words.md#the-three-spaces-of-the-app));
the record is
[ADR-003](./decisions/003-the-three-spaces-are-the-structure-of-the-app.md). It replaces
the six-rung ladder of [ADR-002](./decisions/002-spwn-is-one-stack-of-six-apps-on-one-ladder.md).

## The three spaces

The layers of the Spwn app are three great concepts, and in the future three distinct spaces in
its GUI. The owner drew each as the running of a company:

| Space   | The company it is the parallel of                                                                                                     | State      |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| Machine | The environment: the fleet of machines, the premises, the access, the security, the documents                                         | shipped    |
| Agent   | The staff: their personalities, roles, focus and responsibilities                                                                     | shipped    |
| OS      | The flows: which project, which vision, which product, which system runs inside; it steers the flows, the system and the abstractions | being made |

Each space is one step of [the method](../blueprint/vision.md#the-method-climb-by-abstraction): a
piece of work handed to the stack, a decision kept by you. What you let go of and keep, space by
space, in the parts that exist:

<Group id="parts" title="The parts and what you keep" intent="define">

<Part id="machines" space="machine" lets_go="preparing a computer" keeps="the file that describes it" state="shipped" />

<Part id="agents" space="agent" lets_go="doing the work" keeps="who does what, in which world" state="shipped" />

<Part id="wiki" space="os" lets_go="repeating your context" keeps="the facts and the decisions" state="making" />

<Part id="specs" space="os" lets_go="checking by hand" keeps="what done means" state="making" />

<Part id="design" space="os" lets_go="drawing each screen" keeps="your taste, as a system" state="making" />

<Part id="os" space="os" lets_go="holding it all together" keeps="the intention, and the last word" state="making" />

</Group>

<Table id="all" of="part" in={ref("product/stack")} columns={["space", "lets_go", "keeps", "state"]} />

The column "you keep" is the part no space may take away; [the boundaries](./boundaries.md) say
what each part refuses for that reason.

## How the spaces stack

A space leans on the ones before it and never needs the ones after. Agents runs in machines
Machines can build; in the OS space, Wiki is what agents read, Specs proves the work against what
you and Wiki asked for, Design turns your taste into rules the work is held to, and the OS is the
one place you steer from.

## Rests on

One claim here is this page's reading, not the owner's word: that a space leans only on the
spaces before it ("How the spaces stack"). The placement of Wiki, Specs and Design in the OS space
is his; the "you let go of" and "you keep" columns come from the brief he confirmed on 2026-11-21.

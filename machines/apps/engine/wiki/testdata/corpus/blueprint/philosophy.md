# Philosophy

The seven convictions Spwn is built on, written as principles a newcomer can apply to a design
choice; [the vision](./vision.md) holds the case they serve, and this page holds the rules. The
owner confirmed the seven on 2026-11-21 ([his answer](./owner-words.md#why-spwn-exists)).

<Table id="all" of="conviction" in={ref("blueprint/philosophy")} columns={["says", "carried_in"]} />

Each conviction is written in the same six parts: the statement, the sensible opposite (what a
reasonable product does instead, which is why the conviction says something), the argument, where
the product carries it, one example, and the decision that would break it. A page that cites a
conviction cites its number and links here.

## 1. You bring the taste. The stack brings the hands

<Conviction
id="taste"
says="You bring the taste. The stack brings the hands."
opposite="The tool decides the style and the context for you: a house prompt, a default voice, a template of how work should be done."
carried_in={[ref("product/stack#wiki")]}
broken_by="Shipping a built-in persona or style that overrides what you wrote, or letting one fact live in two places, so that an agent reads the stale copy."
/>

- **Argument.** What an agent cannot supply is what makes the work yours: your taste, your vision,
  your context from outside the screen. The owner's point is not to write down how he works but to
  bring his taste and his context while working at the high level
  ([the problem](./owner-words.md#the-problem), 2026-11-14).
- **Example.** An agent opening a repository reads your vision and your decisions first, not a
  generic instruction.

## 2. Delegating is a craft, not a prompt

<Conviction
id="delegating"
says="Delegating is a craft, not a prompt."
opposite="One strong agent, given every task in one long conversation."
carried_in={[ref("product/stack#agents"), ref("product/stack#os")]}
broken_by="A surface that asks you about everything, or one that hides the detail with no way to open it."
/>

- **Example.** At the end of a day the chief shows what was done and asks you the two questions
  only you can answer; everything else was settled by the owners of each domain.

## 3. Rules live in the world, not in the agent

<Conviction
id="rules-in-the-world"
says="Rules live in the world, not in the agent."
opposite="Give the agent every access and ask it, in its instructions, to be careful."
carried_in={[ref("product/stack#machines"), ref("product/stack#agents")]}
broken_by="A guardrail in a prompt used as the security boundary, or a default that hands a VM the credentials of the Mac above it."
/>

- **Example.** An agent cannot break your computer: it is not on it.

## 4. Declared, never improvised

<Conviction
id="declared"
says="Declared, never improvised."
opposite="Configure by hand, as you go, and forget what you did."
carried_in={[ref("product/stack#machines"), ref("product/stack#agents")]}
broken_by="A setting reachable only by a click with no file behind it, or a state the stack holds that no file explains."
/>

- **Example.** `qa.machine.yaml` gives `qa-1`, set up before its first command.

## 5. Done means proven

<Conviction
id="proven"
says="Done means proven."
opposite="“It seems to work”, said by whoever did the work."
carried_in={[ref("product/stack#specs")]}
broken_by="A status of done set by the agent that did the work, with no check behind it."
/>

- **Example.** A request comes back with its proof, or it does not come back as done.

## 6. Taste is a system

<Conviction
id="taste-is-a-system"
says="Taste is a system."
opposite="Each screen designed on its own, to the taste of the day."
carried_in={[ref("product/stack#design")]}
broken_by="A component drawn by hand for one page when the kit has the piece."
/>

- **Example.** The site's canvas kit: one box, one arrow, one title, the same everywhere.

## 7. The stack learns. You don't repeat yourself

<Conviction
id="learns"
says="The stack learns. You don't repeat yourself."
opposite="Every session starts from nothing."
carried_in={[ref("product/stack#agents"), ref("product/stack#wiki")]}
broken_by="A memory that lives only in a session's context, or a lesson written where no agent will read it."
/>

- **Example.** A correction given once becomes a rule the next agent reads.

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

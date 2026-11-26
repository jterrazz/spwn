# ADR-102: The spaces are a switch of their own, and the path starts at the section

**Status:** Proposed
**Date:** 2026-11-26

**Amends:** [ADR-093](093-the-window-is-a-path-bar-over-a-cool-dark.md): supersedes its decisions 2 (the space as the path's first segment) and 3 (the sign that opens Home), and amends decision 1 (the path's first segment, and where the rail's spaces and Home went).

The owner answered three boards of the window's redesign on 2026-11-26 (spwn-ui-r1, spwn-nav-r1, spwn-nav-r2); his answers, quoted below, are this record's authority. The calls no board put to him were taken the same day by his delegation, on the navigation plan's recommendations, each for him to overturn. Every decision below says which of the two it is. The record waits for his acceptance.

## Context

[ADR-093](093-the-window-is-a-path-bar-over-a-cool-dark.md) made the window one path bar: its first segment is the space, `Machine`, carrying the fleet's roll-up and opening the list of the three spaces; the sign after the traffic lights opens Home. Built, it leaves two mechanisms to move by. From a machine, the Mac and VM segments reach another machine; the app's other sections (Storage, Apps, Passwords and the rest) are reached by a section's own popover or by the column's Library. And the switch between spaces is one more segment of the same path.

The owner answered the first board, which applied the six recommended UI choices to the window, with this remark:

```text
Remarques : incroyable tout parfait adapte tout. Par contre pas besoin du logo dans l'app je pense, et merge aussi les trucs machines et navigation normale histoire de pas perdre le user non ? si j'ai besoin de changer en haut dans la barre du haut de l'app d'une machine à une autre sections de l'app. Et par contre considèrer Machine/agent/OS comme totalement un autre élément. Propose plusieurs versions, qui permettent très clairement de switch entre les espaces
```

The second board drew four questions, each with a recommendation (★); he took all four, and asked for the selector at the left, in ten designs:

```text
d1 · Comment on change d'espace : A · Le sélecteur au centre (★)
d2 · Une seule barre, d'une machine à une section : B · Une carte, depuis le premier segment (★)
d3 · Le coin sans logo : A · Rien dans la fenêtre (★)
d4 · Une teinte par espace : A · Un seul look (★)

Remarques : mais pour "A · Le sélecteur au centre" tu le bouges à gauche non ? et tu proposes plusieurs designs. Peut être basé sur des logos aussi ? 10 idées stp
```

The third board drew the ten designs and two placements. He chose idea 10 against the board's ★ (idea 3, three marks with the active one named):

```text
d1 · Le sélecteur d'espace : 10 · Les mots soulignés
d2 · Sa place à gauche : A · Juste après les feux (★)
d3 · Le centre de la barre : A · Rien au centre (★)
```

The board stated idea 10's weakness, and it stands as the build's problem, not a reason to revisit his choice: « Ils ressemblent aux onglets d'une machine (Overview, Ports…), un étage plus bas : on confond les deux niveaux. » The navigation plan then answered what the boards left open, as seven recommendations; the owner delegated them, and they were taken on their ★ on 2026-11-26.

## Decision

Each decision carries its source: **the owner's choice**, with the board answer it rests on, or **delegated, to overturn**, taken on the plan's ★ by his delegation of 2026-11-26. The plan's calls outside its seven questions are marked delegated too: they rest on the same delegation, not on a board he read.

### The spaces

1. **The three spaces are a selector of their own, at the bar's left, just after the traffic lights.** The owner's choice: spwn-ui-r1 ("considèrer Machine/agent/OS comme totalement un autre élément"), nav-r1 d1 with his remark that moves it left, nav-r2 d2 A. A thin vertical separator closes it; then come the column's pin, Back, Forward and the path. [ADR-019](../../../docs/decisions/019-the-product-is-three-spaces-machine-agent-and-os.md) holds: the switch comes first, leftmost.
2. **The selector is three bare words, Machine, Agent and OS, the active one underlined.** The owner's choice: nav-r2 d1, idea 10, « Pas de boîte : Machine, Agent et OS en texte, l'actif en blanc avec un trait au pied de la barre ». No box, no fill, no tint, no icon; the active word is the brightest and one step heavier.
    - Delegated, to overturn: the line sits on the bar's bottom edge, as wide as the word. The board's text says the foot of the bar; its drawing put the line just under the word.
3. **The words carry no state.** The owner's choice: idea 10 has « Pas de pastille d'état ». Delegated, to overturn: what waits for the owner in any space reads in the waiting chip ([ADR-093](093-the-window-is-a-path-bar-over-a-cool-dark.md) decision 4) and the column's Needs you alone.
4. **The underlined word follows the route, and is never stored.** Delegated, to overturn: a machine route and the Machine space's sections underline Machine, and so do Home and Inbox, the window's own pages drawn in Machine's frame; Settings, above the spaces, underlines none. Back, Forward and a relaunch move the line with the route.
5. **⌘1, ⌘2 and ⌘3 press Machine, Agent and OS.** The owner's choice: nav-r1 d1 A, whose answer read « le sélecteur se pose [...] avec ⌘1–⌘3 ». Delegated, to overturn: a space opens on the last route this window showed in it, else on its default, kept for the window's life.
6. **Agent and OS, until they exist, open the spaces list and move nothing.** Delegated, to overturn: the word is drawn as any inactive word; a press, or ⌘2 and ⌘3, opens the list ADR-093 drew (Agent and OS greyed with "Soon" and what each will hold), anchored on the word. The route, the history and the line stay where they were. OS opens a page of its own when the wiki lands.

### One bar, from a machine to a section

7. **The path's first segment is the section, and it opens the card.** The owner's choice: spwn-ui-r1 ("merge aussi les trucs machines et navigation normale"), nav-r1 d2 B. A machine route reads `Fleet / macbook-pro / linux-1`, a section `Storage`, an item `Apps / ghostty`; Home, Inbox and Settings read their own word. The card holds the space's sections and the fleet side by side, under one filter for both. The Mac and VM segments keep their popovers.
    - Delegated, to overturn: the first segment opens the card on every page, Home, Inbox and Settings included; the window with no engine is the one exception.
    - Delegated, to overturn: the card's Fleet row reopens the last machine this window showed, else the first Mac. There is no Fleet page, and the Machine space page once planned is dropped.
    - Delegated, to overturn: `Fleet` carries no mark, so the fleet's roll-up leaves the bar for the column, Home, the chip and the card.
8. **The window holds no logo; Home is ⌘0 and the card's first row.** The owner's choice: spwn-ui-r1 ("pas besoin du logo dans l'app"), nav-r1 d3 A, whose answer read « le signe quitte la barre, Home garde ⌘0 et la première ligne de la carte ». The bar holds no `spwn` word; the app icon keeps the mark; Find still lists Home.
9. **The three spaces share one look.** The owner's choice: nav-r1 d4 A. No space has a tint; the bar, the column and the content are the same grounds in every space.
10. **The bar's centre is empty.** The owner's choice: nav-r2 d3 A. The waiting chip stays at the bar's right, as ADR-093 decision 4 has it.

### The two levels read apart

11. **The space words never read as a page's tabs.** Delegated, to overturn: five differences hold them apart, none of them a box or a tint.
    - The space's line lies on the bar's bottom edge, part of the window's chrome; a tab's line sits over the content's rule, inset from its cell.
    - The space words are one weight heavier than the tabs, at rest and active.
    - The words sit against the traffic lights, closed by the separator; tabs sit under a page's head, never in the bar.
    - The words are bare, with air between them; tabs are padded cells.
    - The selector is a navigation of three toggle buttons: the arrows move the focus and Enter presses. A page's tabs are a tablist, whose arrows select.
    - A component test measures the line's place and the weights on a VM's Ports page; the owner's eye on a capture and on a dev build judges the rest.

### The keys and the widths

12. **The window's keys for its navigation** are these; a key does nothing while a dialog is open.

| Key        | Does                     | Source                                                  |
| ---------- | ------------------------ | ------------------------------------------------------- |
| ⌘1, ⌘2, ⌘3 | press Machine, Agent, OS | the owner's choice, nav-r1 d1 A                         |
| ⌘0         | opens Home               | the owner's choice, nav-r1 d3 A; already bound          |
| ⌘L         | opens the card           | delegated, to overturn; the nav-r1 board's text named F |
| ⌘K         | opens Find               | unchanged                                               |

13. **The window stays at least 1100 px wide, and the bar is laid out for it.** Delegated, to overturn: the third board's 900 px rule has no width to apply to. Below 1280 px Find becomes its magnifier alone, ⌘K unchanged. The three words stay words at every width the window takes; the path's middle folds into `…` by measure, as today.

### Not decided here

- **The column's Library.** The card holds every section, so the Library could leave the column. Delegated: it waits for the owner's word, after he has tried the card.
- **Agent's sections in the card.** None until the space is built; OS lists Home and Wiki when the wiki lands.

## Alternatives

- Ideas 1 to 9 of nav-r2, each refused by the owner's pick of idea 10. The segmented track with icons (1) is the widest, and loses its words below 1200 px. The family's three marks (2), the active mark named (3, the board's ★), the monograms (4), the sign's three dots (5), the one word over three dots (6), the raised tiles (7), the stacked floors (8) and the mark that unfolds on hover (9) each cost a mark to learn, a hidden space, a second gesture, or the logo back in the window.
- nav-r1 d1 B, a rail of three at the window's left edge; C, the bar tinted by its space with a panel to switch; D, a space badge before the path opening three tiles.
- nav-r1 d2 A, each segment opening its neighbours, which makes the owner pick the right menu; C, the bar turned into a field, which hides the path and repeats Find.
- nav-r1 d3 B, a grey sign at the column's foot; d4 B, a tint per space in the tokens.
- nav-r2 d2 B, just before the path; C, at the head of the column. d3 B, the waiting chip at the bar's centre.

## Consequences

- ADR-093's body stays as written and its header names this record. Its sentences this record makes false: decision 1's `Machine / macbook-pro / linux-1`, its list of siblings "(the spaces; ...)", "Home to the sign" and "the spaces to the first segment"; decisions 2 and 3 whole; "beside the sign" for Back and Forward; and its consequence "as the path's first segment" for ADR-019.
- This record changes no code. The window's README ("The path bar and the spaces", "The keyboard") and its scenarios are rewritten by each lot whose code makes them false.
- Every frame that draws the bar moves its ARIA golden: the bar gains the Spaces navigation, the path loses the Machine segment and gains the section, the Home button leaves.
- ⌘2 and ⌘3, dead keys today, act; ⌘L is taken. HYPOTHESIS (2026-11-25): Tauri's default menu binds none of ⌘1 to ⌘3 nor ⌘L; reading `Menu::default` in the tauri crate settles it.
- The OS wiki's branch rebases onto the selector: its space opens on the wiki, and its card lists Home and Wiki.
- The fleet's roll-up no longer shows in the bar; with the column hidden on a section, a Mac that is down shows through the chip and the card. A mark on `Fleet` is the owner's to ask back.

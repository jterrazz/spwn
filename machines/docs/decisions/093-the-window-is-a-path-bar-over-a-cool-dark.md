# ADR-093: The window is a path bar over a cool dark

**Status:** Proposed
**Date:** 2026-11-25

**Amended by:** [ADR-102](102-the-spaces-are-a-switch-of-their-own-and-the-path-starts-at-the-section.md) (Proposed): decisions 2 and 3 superseded, decision 1 amended: the spaces are a selector of their own at the bar's left, the path's first segment is the section, and the window holds no sign.

The owner chose the navigation in round 4 of the window's redesign boards (2026-11-25); his answers, quoted below, are this record's authority. The record itself waits for his acceptance, and five smaller calls it carries are recommended and still his to answer.

## Context

The window is navigated today by two columns, as Slack draws it: a 56 px rail for the app and its spaces, a 224 px sidebar for the Machine space ([the window's README](../../apps/desktop/README.md), "The rail and the spaces"). The product has three spaces, Machine, Agent and OS ([the repository's ADR-019](../../../docs/decisions/019-the-product-is-three-spaces-machine-agent-and-os.md)), and each will grow pages and tabs. No record says where the window's greys come from: `src/presentation/ui/tokens.css` holds a near-black (`--canvas #0c0c0c`) that no measure chose.

The owner opened the redesign with three remarks, answering round 1 (2026-11-25): « j'aimais bcp la proposition, juste c'était pas future proof pour le pleins de tabs qui arriveront dans les sous familles »; « pour la fenetre en dark c'était aussi revoir les couleurs pour match le dark »; « pas forcément abandonner une navbar à la slack ». Round 2 measured his reference, two screenshots of a dark app he chose, pixel by pixel. Round 3 drew five structures, and he answered « j'aime beaucoup le rendu de 05 · STRUCTURE 4 ... prendre un peu du meilleur de tout le monde ». Structure 4 is the path: the top bar says where I stand, and each level opens its neighbours. Round 4 drew it with the best of the four others grafted on, as five questions with a recommended answer (★) and one alternative each. He took every ★:

```text
q1 · État de la flotte : une colonne qui sort et s'épingle (★)
q2 · Changer d'espace : le premier segment, et ⌘1 à ⌘5 (★) — incroyable ça magnifique le concept de popup au hiver
q2 · Marque dans la fenêtre : le signe seul, en haut à gauche (★)
q3 · Ce qui m'attend : une pastille avec le premier besoin (★)
q4 · Corps de Home : la carte, puis Needs you (★) — incroyable la notification c'est magnifique, par contre faudra revoir le layout, histoire d'avoir un truc plus joli pour cette View, on peut faire bien mieux je trouve, prévoir le cas de plusieurs machines etc etc etc. Comment ça va scale, avoir un visuel qui parle du sens. Hesites pas à challenger notre concept de bento présent partout peut être, je sais pas trop. Regarde les best practices de ceux qui ont des designs comme ça et trop parfaits
q5 · Trop d'onglets : un menu More (★)
```

Two records already speak of greys. [ADR-036](036-the-menu-bar-owns-its-glass-panel.md) and [ADR-053](053-the-panel-is-untinted-glass-over-a-grey-ground.md) make the menu bar panel an untinted glass over a hue-free grey. The round-2 reference is not hue-free: its grey leans blue, hue 228° at 8 % saturation. Whether their rule binds the window had to be settled before a token moves.

## Decision

### The navigation

1. **The window is navigated by one path bar, with no permanent navigation column.** One bar across the window's top holds the path: one segment per level, `Machine / macbook-pro / linux-1`, and each segment opens a popover of its siblings (the spaces; the Macs; a Mac's VMs), a filter at its head. A tab is never a segment. The rail is removed and its places move: Home to the sign, Inbox to the chip, the spaces to the first segment, Passwords into the Machine space, Settings to the bar's right end. This is structure 4 of round 3, the owner's pick; the rail kept was the alternative, and it costs every page 56 px.
2. **The space is the path's first segment** (q2): its popover lists Machine, Agent and OS, each with its mark, Agent and OS greyed with "Soon" until they run. The segment carries one mark, the space's standing as the engine reports it ([ADR-054](054-a-machines-standing-is-one-rule-at-each-level.md)); the window computes no worst state of its own. The alternative was a thin rail of space icons, 64 px on every page.
3. **The sign alone, at the top-left, opens Home** (q2, "Marque dans la fenêtre"). The sign is drawn just after the traffic lights, never the word `spwn`, and its geometry has one owner, the app icon's drawing. The alternative was the word.
4. **What waits for the owner is always visible, as a chip naming its first item** (q3): the Inbox icon, the first need's sentence and `+N`, at the bar's right. A press opens the needs in a popover, then "Open Inbox". No need: the chip is the icon alone. The alternative was the icon with a count, which says there is something but not what.
5. **The fleet's state is a column that peeks on hover and pins** (q1). Pinned, it sits under the bar, which stays full width, so nothing moves when it pins. Hidden, it peeks over the content when the pointer reaches the window's left edge, and a button in its foot pins it. It is pinned by default when the window is 1440 px wide or more, hidden below; a choice the owner makes wins over the width and is kept across relaunches. It holds "Needs you", then the machines, each Mac folding with its count and its standing, then the Library. The alternative was one mark per Mac in the Machine segment, with no column: it says where something is wrong, never what.
6. **Tabs that do not fit go into `More`** (q5): the tabs under the object's head, then `More N`, a popover with the hidden ones; the open tab is never hidden. The alternative, scrolling tabs, hides tabs behind arrows.

### The dark palette

7. **The window's dark is the cool grey measured on the owner's reference.** Its greys carry the reference's cast, hue about 228° at about 8 % saturation, and the column and the title bar sit one step above the content, as in the reference. The steps are round 2's measure: `--canvas #17181c`, a new `--side #1b1c20` for the column and the bar, `--panel #26272b`, `--inset #121316`, `--raise #303136`, `--raise-2 #3b3c42`; text `--t1 #f7f7f8` to `--t4 #6f7176`; tones `--ok #4ad68a`, `--warn #f6b24a`, `--bad #ff6d63`, `--new #4cc2f5`; a pill drawn at 8 % fill and 24 % line. Round 3 kept the cool as its ★ and round 4 drew it unchanged. The token list and its order of landing belong to the lot that applies them.
8. **The palette holds its contrast thresholds, by WCAG 2's luminance ratio.**
    - `--t1`, `--t2` and `--t3` hold 4.5:1 on `--canvas`, `--side` and `--panel`; the lowest measured is `--t3` on `--panel`, 5.4.
    - Each tone holds 4.5:1 on its own pill, the tone at 8 % over each ground; the lowest is `--bad` on `--panel`, 4.8.
    - `--on-primary` holds 4.5:1 on `--primary` (17.7). A state mark on the open row's white pill takes its dark value (`--*-on-primary`) and holds 3:1, as any mark or dot does; the lowest is `ok`, 4.0.
    - `--t4` is decoration, never text to read, and holds 3:1; the lowest is 3.1 on `--panel`.
    - A test computes these from `tokens.css` and fails a token below its bar. No golden of the window is a picture, so no golden can hold them.
9. **ADR-036 and ADR-053 rule the menu bar panel alone, so this record supersedes nothing.** Their hue-free rule is scoped to `StatusPanel`: ADR-053's Decision 1 is "`StatusPanel`'s one background is an `NSGlassEffectView`, `.regular`, with no tint" over "a hue-free grey at 0.6" ([053:20](053-the-panel-is-untinted-glass-over-a-grey-ground.md)), and ADR-036 is titled "The menu bar owns its panel" ([036:1](036-the-menu-bar-owns-its-glass-panel.md)), its decision `StatusPanel` and its `NSPanel` ([036:28](036-the-menu-bar-owns-its-glass-panel.md)). The rule answers a complaint about that panel, « washed white-bluish » ([053:8](053-the-panel-is-untinted-glass-over-a-grey-ground.md)), measured against the Apple menu beside it on the menu bar, which the panel must read as. The window has no system neighbour to match: its reference is the one the owner asked it to match. So the menu bar stays hue-free, the window carries the measured cast, and neither rule reaches the other's surface.

### What waits for the owner

These five calls follow from the decisions above, and each is **recommended, awaiting the owner**:

- **Spaces on ⌘1 to ⌘3, tabs moved to ⌃1 to ⌃9.** The owner asked ⌘1 to ⌘5 for spaces; there are three spaces, so ⌘1 to ⌘3, the table growing with the spaces. ⌘ with a digit is a machine's tab today (`src/presentation/navigation/shortcut.ts`), so the tabs move to ⌃, in their bar's order, hidden ones included.
- **⌃⌘S pins the column**, macOS's own "Show Sidebar" key. Round 4 drew ⌘\, which on an AZERTY keyboard needs ⇧ and is dropped by the window's shortcut reader. HYPOTHESIS (2026-11-25): the owner types AZERTY.
- **Back and Forward stay in the bar**, beside the sign: round 4's mock left them out, and the window's history and ⌘[ ⌘] keep them.
- **A Settings gear at the bar's right end, not an avatar**: the window holds no account to show.
- **The cool cast is judged by the owner's eye on a dev build on his Mac**, not on a capture: the vibrant material (`--material-canvas` 86 %) exists only in the Tauri window on a Mac, and no capture shows it. If the cast reads blue there, as the menu bar panel did, this decision is reopened.

### Not decided here

- **Home's layout.** The owner kept the map, then Needs you, and asked for better: « faudra revoir le layout [...] prévoir le cas de plusieurs machines ». That is round 5. Until it lands, Home keeps today's body inside the new bar, its path the word `Home`.
- **The Agent and OS spaces' own pages and their column.** Both stay greyed, as ADR-019 has them.
- **`runs on <machine>` and an agent count per machine.** No call carries them; they wait for the engine to list agent sessions per machine, and are drawn against that field, never a provisional one.

## Consequences

- ADR-019 holds: the switch between spaces comes first, as the path's first segment, with Agent and OS greyed until they run.
- The window's README states the rail, the keys and the chrome as they are today. Each lot that builds a part of this record rewrites the sections its code makes false: "The rail and the spaces", "The keyboard", "The window's chrome on a Mac" (the bar's row height moves the traffic lights' `y`) and "The design rules" (where the greys come from).
- The navigation lots regenerate many ARIA goldens, as every frame names the rail; the palette moves none, so its proof is the contrast test and a before/after capture the owner looks at.
- A popover becomes a kit primitive, the one pattern every segment, the chip and `More` reuse.
- Two surfaces of one product now carry two greys: the menu bar's hue-free, the window's cool. That is the cost of matching each to its neighbour, and the owner's look on his Mac is its check.

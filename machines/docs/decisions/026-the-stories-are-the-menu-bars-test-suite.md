# ADR-026: The stories are the menu bar's test suite

**Status:** Accepted
**Date:** 2026-11-07

Written 2026-11-10 from DECISIONS K3 (round 14), `v4/PLAN.md` §0 and the suite as lots L01 to C1 built it.

## Context

The menu bar's views were proved by per-page snapshot files, one golden each, written as each round built a page: about 66 MB of panel goldens with no list of which states of the design they covered. The v4 target is the accepted prototype, which draws 168 states of the panel. The owner asked for "un genre de storybook pour apps, pour parcourir chaque state, et transcrire toute la vision dans ces tests" (K3, 2026-11-07): one place to browse every state, and the same states as the tests.

## Decision

- A **story** is one state of the panel: an id of `apps/menubar/stories/STORIES.md`, and its `<id>.story.json` (`station.story/v1`) beside it, naming the pages it draws (the page-language JSON the engine sends, generated from the real page functions by `make generate`) and the panel's UI state over them (a segment, a sheet, a menu, a query, a focus, a frozen movement).
- One loader, `Stories.load` in StationKit's `catalog/`, serves three readers: the Station Catalog (`apps/menubar/sources/station-catalog`, a dev-only window with knobs for scheme, contrast, transparency, motion and width, never bundled into `Station.app`), the tests, and the gallery (`StationCatalog --export <dir> --gallery`, an HTML page of every story's images to browse on a phone). Each story also has its Xcode `#Preview`, generated.
- The stories are the tests: `StoryTests` draws every story offscreen at 2×, still, over a flat desk, dark and light, against its two goldens; `StoryCoverageTests` fails on an id of STORIES.md with no story or goldens, on an orphan file, and on drift of the generated `INDEX.md`. A story is also held to the prototype's height (`heights.json`, 4 pt, or a `heightNote` that says why not), stays glass (under 40 % opaque), and each group keeps one story proved at Increase Contrast and Reduce Transparency.
- The ratchet that let 167 stories land lot by lot, `stories/PENDING`, is gone (C1): a missing story now fails. The ids STORIES.md holds that no lot built (the A2 additions the coverage test names `unbuilt`, and the gap ids of SPEC C6) are its pending output, and that list only shrinks.
- The goldens have a budget: 120 MB for `testdata/stories/`, and the kit's other goldens ratcheted at what they weigh.

## Consequences

- A change to a view is judged by the stories that draw it; a new state of the design is a section of STORIES.md first, then its story file, its world's pages and its goldens ([Test](../guides/test.md#the-menu-bars-stories)).
- The per-page snapshot files, the vision rounds' tests and the generic page renderer's goldens retired with the states the stories took over (C1).
- What no golden can show is proved apart: a status item's count by its accessibility label, a movement by its frozen frame and the Catalog's replay at that frame, the real menu bar's behaviour in the macOS VM ([ADR-021](021-the-menu-bar-keeps-its-system-window.md)).
- No check and no agent opens the Catalog's window: `make catalog-open` is the owner's gesture; checks use `--export`.

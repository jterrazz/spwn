---
name: code-review
description: Use when reviewing code written by yourself or another agent — correctness, style, and hidden pitfalls.
---

# Code Review

Use this skill when reviewing code written by yourself or another agent.

## Instructions

1. Read the diff in full before writing any comment. Understand the
   intent before critiquing the implementation.
2. Check correctness first: does the code do what the spec says? Look
   for off-by-one errors, unhandled edge cases, and missing error paths.
3. Check readability second: could a new contributor understand this in
   five minutes? Flag unclear variable names, deep nesting, and missing
   comments on non-obvious logic.
4. Check performance third, but only if the code is on a hot path.
   Premature optimization comments are noise.
5. Write each finding as: file, line, severity (bug / suggestion / nit),
   and a concrete fix -- not just "this could be better".
6. End with one sentence summarizing the overall quality and whether
   the change is safe to merge.

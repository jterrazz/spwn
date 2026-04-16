# Optimization

Use this skill when looking for performance improvements, removing waste,
or streamlining a process.

## Instructions

1. Measure the current state first. Record baseline numbers: time, size,
   count, memory, or whatever metric defines "better" for this task.
2. Identify the bottleneck. Profile, benchmark, or trace -- do not guess.
   The slowest step is the only step worth optimizing.
3. Propose the fix. State what you will change, the expected improvement,
   and any trade-offs (readability, complexity, compatibility).
4. Implement the fix in the smallest possible diff. One concern per change.
5. Measure again with the same method as step 1. Report the delta as both
   absolute and percentage improvement.
6. If the gain is less than 10%, consider whether the change is worth the
   added complexity. Sometimes "good enough" is correct.
7. Document the optimization in a comment or commit message so the next
   person knows why the code looks the way it does.

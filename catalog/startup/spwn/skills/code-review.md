# Skill: Code Review

How to review code changes before they ship.

## The review process

### 1. Understand the context

Before reading the diff, understand why the change exists:

```
# Read the commit messages
git log --oneline -10

# Check what branch you're on
git branch -v
```

### 2. Read the diff

```
# Full diff against main
git diff main...HEAD

# Or review specific files
git diff main...HEAD -- src/
```

### 3. Check for common issues

- **Security**: hardcoded secrets, unsafe inputs, missing auth checks.
- **Performance**: N+1 queries, unbounded loops, missing indexes.
- **Correctness**: edge cases, off-by-one errors, null handling.
- **Readability**: unclear names, missing comments on non-obvious logic.
- **Tests**: are the new paths tested? Are edge cases covered?

### 4. Write the review

Structure your feedback:

```
## Summary
One sentence on the overall quality.

## Issues (must fix)
- [ ] Specific actionable items

## Suggestions (nice to have)
- [ ] Optional improvements

## Verdict
Ship / Revise / Block
```

## Review principles

- **Be specific.** "This is bad" is not a review. Point to the line, explain why.
- **Separate blockers from nits.** The author needs to know what must change
  vs what you'd prefer.
- **Review the change, not the author.** Focus on the code.
- **Time-box it.** 30 minutes max. If it takes longer, the PR is too big.

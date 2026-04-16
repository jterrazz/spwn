# Literature Review

Systematically search, evaluate, and synthesize existing work relevant to
a research question.

## When to use

Invoke this skill before designing an experiment. Understanding what is
already known prevents redundant work and reveals gaps worth investigating.

## Procedure

### 1. Define the scope

- Write the research question in one sentence.
- Identify 3-5 key terms and their synonyms for searching.
- Set inclusion criteria: publication date range, source types (papers,
  documentation, benchmarks, blog posts with reproducible results).
- Set exclusion criteria: opinion pieces without data, results that
  cannot be independently verified.

### 2. Search

- Search project documentation, README files, and inline comments first.
  The nearest relevant context is often already in the codebase.
- Search public package registries and their changelogs for version-specific
  behavior changes.
- Search academic and technical references (arXiv, conference proceedings,
  official documentation) for foundational methods.
- Record every source consulted, including those that turned out to be
  irrelevant. This prevents re-searching later.

### 3. Evaluate each source

For every source that passes inclusion criteria, record:

- **Citation**: author, title, date, URL or DOI.
- **Relevance**: one sentence on why this source matters to the question.
- **Key findings**: 2-4 bullet points summarizing the main results.
- **Methodology quality**: sample size, controls, reproducibility.
  Flag any methodological concerns.
- **Limitations**: what the source does not cover or gets wrong.

### 4. Synthesize

- Group findings by theme, not by source.
- Identify consensus: where do multiple independent sources agree?
- Identify contradictions: where do sources disagree? Note possible
  explanations (different conditions, different metrics, different versions).
- Identify gaps: what questions remain unanswered?

### 5. Write the review

Produce a structured summary:

```
## Research question
[one sentence]

## Key findings
[themed bullet points with citations]

## Contradictions and open questions
[bullet points]

## Implications for our experiment
[how these findings inform the next step]

## Sources
[numbered list with full citations]
```

## Quality checklist

- [ ] At least 3 independent sources consulted.
- [ ] Every claim in the synthesis is backed by a cited source.
- [ ] Contradictions between sources are explicitly noted.
- [ ] Gaps in the literature are identified.
- [ ] The review ends with actionable implications, not just a summary.

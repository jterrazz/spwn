---
name: experiment-design
description: Use when designing controlled, reproducible experiments that isolate the variable under study and produce statistically meaningful results.
---

# Experiment Design

Design controlled, reproducible experiments that isolate the variable
under study and produce statistically meaningful results.

## When to use

Invoke this skill after completing a literature review and before writing
any experimental code. A well-designed experiment saves hours of wasted
computation and ambiguous results.

## Procedure

### 1. State the hypothesis

Write a falsifiable hypothesis in this format:

> **H0 (null):** [variable X] has no effect on [metric Y] under [conditions Z].
> **H1 (alternative):** [variable X] [increases/decreases/changes] [metric Y] by [expected magnitude] under [conditions Z].

If you cannot write a falsifiable hypothesis, the question is not yet
specific enough. Refine it before proceeding.

### 2. Identify variables

- **Independent variable**: the single factor you will manipulate.
- **Dependent variable**: the metric you will measure.
- **Controlled variables**: factors you will hold constant across all
  conditions to prevent confounding.
- **Confounding risks**: factors you cannot fully control. Document them
  and describe your mitigation strategy.

### 3. Design conditions

- **Treatment group**: the condition where the independent variable is
  applied.
- **Control group (baseline)**: the condition without the independent
  variable, but otherwise identical.
- **Replication**: define the number of independent trials. Justify the
  count (power analysis if feasible, minimum 5 trials for computational
  experiments, minimum 30 for noisy measurements).

### 4. Define the measurement protocol

For each dependent variable, specify:

- **Metric**: exact definition (e.g., "p95 latency in milliseconds",
  "F1 score on the held-out test set").
- **Measurement method**: the tool or code that produces the number.
- **Precision**: significant figures or decimal places to report.
- **Collection timing**: when during the experiment each measurement
  is taken.

### 5. Write the execution plan

Produce a numbered step-by-step protocol that another agent could follow
without any additional context:

```
1. Set up environment: [exact versions, dependencies, seeds]
2. Prepare data: [source, preprocessing steps, train/test split]
3. Run baseline: [exact command, expected output format]
4. Run treatment: [exact command, what changes from baseline]
5. Collect measurements: [where results are stored, format]
6. Repeat steps 3-5 for N trials with seeds [list]
```

### 6. Pre-register the analysis plan

Before running anything, write down:

- The statistical test you will use to compare groups (e.g., paired t-test,
  Wilcoxon signed-rank, bootstrap confidence interval).
- The significance threshold (e.g., alpha = 0.05).
- What result would cause you to reject the null hypothesis.
- What result would be inconclusive.

This prevents post-hoc rationalization.

### 7. Document in the protocol file

Save the complete design as a protocol file before execution:

```
## Experiment: [descriptive title]
## Date: [ISO 8601]
## Investigator: curie

### Hypothesis
[H0 and H1]

### Variables
[independent, dependent, controlled, confounders]

### Conditions
[treatment, control, replication count]

### Measurement protocol
[metrics, methods, precision]

### Execution plan
[numbered steps]

### Analysis plan
[statistical test, threshold, decision criteria]
```

## Quality checklist

- [ ] Hypothesis is falsifiable.
- [ ] Exactly one independent variable is being tested.
- [ ] A proper baseline/control condition exists.
- [ ] Number of trials is justified and sufficient.
- [ ] Metrics are precisely defined with units.
- [ ] Analysis plan is pre-registered before execution.
- [ ] The protocol is complete enough for another agent to reproduce.

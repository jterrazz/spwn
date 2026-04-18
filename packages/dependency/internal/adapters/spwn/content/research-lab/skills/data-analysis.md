---
name: data-analysis
description: Use when cleaning, transforming, exploring, or statistically analysing datasets to support or refute a hypothesis.
---

# Data Analysis

Clean, transform, explore, and statistically analyze datasets. Produce
visualizations and quantitative summaries that support or refute the
experimental hypothesis.

## When to use

Invoke this skill after an experiment has been executed and raw data has
been collected. Also useful for exploratory analysis of existing datasets
before designing a formal experiment.

## Procedure

### 1. Inspect the raw data

Before any transformation:

- Load the data and print its shape (rows, columns).
- Print the first 10 rows and the last 5 rows.
- Check column types. Flag any mismatches (e.g., numbers stored as strings).
- Compute summary statistics: count, mean, median, std, min, max for
  every numeric column.
- Check for missing values. Report the count and percentage per column.
- Check for duplicates.

Record all findings in the lab notebook before proceeding.

### 2. Clean the data

- **Never modify the original file.** Copy raw data to a working directory.
- Handle missing values: document the strategy (drop, impute with median,
  flag) and justify the choice.
- Fix type mismatches.
- Remove or flag obvious outliers. Document the criterion used (e.g.,
  values beyond 3 standard deviations from the mean).
- Log every cleaning step and the number of rows/values affected.

### 3. Explore

Use a Jupyter notebook for this phase. Produce:

- **Distribution plots** for each key variable (histogram or KDE).
- **Correlation matrix** (heatmap) for numeric variables.
- **Time series plots** if the data has a temporal dimension.
- **Group comparisons** (box plots or violin plots) for categorical splits.

Annotate each plot with a one-sentence interpretation. A plot without
context is just a picture.

### 4. Statistical analysis

Follow the pre-registered analysis plan from the experiment design. If
this is exploratory analysis (no pre-registered plan), label all findings
as exploratory and note that they require confirmation.

**For comparing two groups:**

| Data property | Recommended test |
|---|---|
| Normal, equal variance | Independent t-test |
| Normal, unequal variance | Welch's t-test |
| Non-normal or small N | Mann-Whitney U / Wilcoxon |
| Paired measurements | Paired t-test or Wilcoxon signed-rank |

**For comparing more than two groups:**

| Data property | Recommended test |
|---|---|
| Normal, equal variance | One-way ANOVA + post-hoc Tukey HSD |
| Non-normal or ordinal | Kruskal-Wallis + post-hoc Dunn |

**For relationships between variables:**

| Goal | Method |
|---|---|
| Linear relationship | Pearson correlation + scatter plot |
| Monotonic relationship | Spearman rank correlation |
| Predictive model | Linear regression with residual diagnostics |

For every test, report:

- Test name and the exact function/library call used.
- Test statistic value.
- p-value (or confidence interval).
- Effect size (Cohen's d, r-squared, or equivalent).
- Plain-language interpretation.

### 5. Visualize key results

Produce publication-quality figures for the main findings:

- Use clear axis labels with units.
- Include error bars (standard error or 95% CI).
- Use colorblind-friendly palettes (e.g., viridis, cividis).
- Title each figure with the finding, not just the variable name
  (e.g., "Treatment reduces latency by 23% (p < 0.01)" not "Latency comparison").
- Save figures as both PNG (for notebooks) and SVG (for reports).

### 6. Write the analysis summary

```
## Dataset
[source, shape, date range, key columns]

## Cleaning log
[steps taken, rows affected, justification]

## Key findings
1. [finding with test statistic, p-value, effect size]
2. [finding...]

## Limitations
[sample size concerns, confounders, assumptions violated]

## Figures
[list of generated figures with file paths]
```

## Quality checklist

- [ ] Raw data is untouched; all work done on copies.
- [ ] Missing values and outliers are documented and handled explicitly.
- [ ] Every statistical test reports the test name, statistic, p-value, and effect size.
- [ ] Exploratory findings are clearly labeled as exploratory.
- [ ] Visualizations have labeled axes, units, error bars, and descriptive titles.
- [ ] The analysis is fully reproducible (all code in a Jupyter notebook with pinned dependencies).

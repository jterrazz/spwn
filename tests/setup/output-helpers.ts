/**
 * Strip ANSI escape codes from output for clean assertions.
 */
export function stripAnsi(str: string): string {
  return str.replace(/\x1B\[[0-9;]*[a-zA-Z]/g, "");
}

/**
 * Normalize output: strip ANSI, trim, collapse whitespace.
 */
export function normalize(str: string): string {
  return stripAnsi(str).trim().replace(/\s+/g, " ");
}

/**
 * Extract lines from output, stripping ANSI and empty lines.
 */
export function lines(str: string): string[] {
  return stripAnsi(str)
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l.length > 0);
}

/**
 * Assert output contains a line matching a pattern.
 */
export function expectLine(output: string, pattern: RegExp): void {
  const outputLines = lines(output);
  const found = outputLines.some((l) => pattern.test(l));
  if (!found) {
    throw new Error(
      `Expected output to contain a line matching ${pattern}\nGot:\n${outputLines.join("\n")}`,
    );
  }
}

/**
 * Assert output does NOT contain a line matching a pattern.
 */
export function expectNoLine(output: string, pattern: RegExp): void {
  const outputLines = lines(output);
  const found = outputLines.some((l) => pattern.test(l));
  if (found) {
    throw new Error(
      `Expected output NOT to contain a line matching ${pattern}\nGot:\n${outputLines.join("\n")}`,
    );
  }
}

/**
 * Assert output contains a table header with specific columns.
 */
export function expectTableHeader(
  output: string,
  columns: string[],
): void {
  const outputLines = lines(output);
  const headerLine = outputLines.find((l) =>
    columns.every((c) => l.includes(c)),
  );
  if (!headerLine) {
    throw new Error(
      `Expected table header with columns [${columns.join(", ")}]\nGot:\n${outputLines.join("\n")}`,
    );
  }
}

/**
 * Assert output contains a table row with specific values.
 */
export function expectTableRow(
  output: string,
  values: string[],
): void {
  const outputLines = lines(output);
  const found = outputLines.some((l) => values.every((v) => l.includes(v)));
  if (!found) {
    throw new Error(
      `Expected table row containing [${values.join(", ")}]\nGot:\n${outputLines.join("\n")}`,
    );
  }
}

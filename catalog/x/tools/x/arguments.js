const UNIT_MS = { d: 86_400_000, h: 3_600_000, m: 60_000 };

/** An integer argument, or `fallback` when the call names none. */
export function integerOf(args, key, fallback) {
  return args[key] === undefined || args[key] === null ? fallback : Number.parseInt(String(args[key]), 10);
}

/** The handle a call names without its @, or X_HANDLE's; refused when neither is set. */
export function handleOf(args) {
  const handle = String(args.handle ?? process.env.X_HANDLE ?? '').replace(/^@/u, '');
  if (handle === '') {
    throw new Error('handle is required (pass --handle or set X_HANDLE)');
  }
  return handle;
}

/** A tweet's text, refused when empty or past X's 280 characters. */
export function tweetTextOf(args) {
  const text = String(args.text ?? '');
  if (text === '') {
    throw new Error('text is empty');
  }
  if (text.length > 280) {
    throw new Error(`text is ${text.length} chars (max 280)`);
  }
  return text;
}

/** `--since` in milliseconds: a date (start of day UTC), a datetime, or a duration back from now ("26h", "3d", "120m"); null when absent or unreadable. */
export function sinceOf(raw) {
  const text = String(raw ?? '').trim();
  if (text === '') {
    return null;
  }
  const relative = /^(?<amount>\d+)\s*(?<unit>[dhm])$/iu.exec(text)?.groups;
  if (relative !== undefined) {
    return Date.now() - Number.parseInt(relative.amount, 10) * UNIT_MS[relative.unit.toLowerCase()];
  }
  const ms = Date.parse(/^\d{4}-\d{2}-\d{2}$/u.test(text) ? `${text}T00:00:00Z` : text);
  return Number.isNaN(ms) ? null : ms;
}

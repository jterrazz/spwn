// outside: X silently drops the data XHRs of a session that scrolls by identical deltas, on uniform timing, with no
// idle reading pause, so every scroll and wait here is jittered.

import { createRequire } from 'node:module';

import { createdMs, harvest } from './tweets.js';

// outside: the gate resolves @spwn/gate-tool through NODE_PATH, which only a CommonJS require reads
const { openSession } = createRequire(import.meta.url)('@spwn/gate-tool');

const PROVIDER = 'x';
export const NAV_TIMEOUT = 25_000;
const MAX_SCROLLS = 60;
const STALE_SCROLLS_FOR_END = 3;

const COOKIE_REFUSAL = `(() => {
  const labels = ['refuse non-essential', 'refuser les cookies non essentiels', 'reject non-essential'];
  for (const button of document.querySelectorAll('button, [role="button"]')) {
    const label = (button.textContent || '').trim().toLowerCase();
    if (labels.some((refusal) => label.startsWith(refusal))) {
      button.click();
      return 'refused';
    }
  }
  return 'not-shown';
})();`;

export const jitter = (min, max) => min + Math.floor(Math.random() * (max - min));

export async function sleep(ms) {
  await new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

/** Run `body` with a fresh session, closed whatever happens. */
export async function withSession(body) {
  const session = await openSession(PROVIDER);
  try {
    return await body(session);
  } finally {
    await session.end();
  }
}

async function humanScroll(session, index) {
  await session.scroll({ count: 1, delta_y: jitter(2800, 5400), wait_ms: jitter(1400, 2900) });
  if (index > 0 && index % jitter(3, 5) === 0) {
    await sleep(jitter(2500, 5500));
  }
}

// outside: X's EU consent banner blocks the timeline from rendering (with it up, /<handle>/likes reads "account doesn't exist"), so it is refused wherever it shows
async function dismissCookieBanner(session) {
  try {
    await session.eval(COOKIE_REFUSAL);
    await sleep(jitter(800, 1500));
  } catch {
    // value: the banner may be gone or its DOM changed; the timeline is read either way
  }
}

// value: a session landing cold on /<handle>/likes reads as a bot more than one that pokes around /home first
async function warmUp(session) {
  try {
    await session.goto('https://x.com/home', { timeout_ms: NAV_TIMEOUT, wait_until: 'domcontentloaded' });
    await sleep(jitter(1500, 3000));
    await dismissCookieBanner(session);
    await session.scroll({ count: 1, delta_y: jitter(800, 1600), wait_ms: jitter(1200, 2200) });
  } catch {
    // value: a warm-up that fails still leaves the real target to try
  }
}

async function open(session, url, warm) {
  if (warm) {
    await warmUp(session);
  }
  await session.goto(url, { timeout_ms: NAV_TIMEOUT, wait_until: 'domcontentloaded' });
  await sleep(jitter(2200, 3800));
  await dismissCookieBanner(session);
}

// outside: with no operation name the harvest reads every GraphQL response, for when X renames one or masks it from a session it suspects
async function harvestCaptured(session, { operation, seen, sink, typenames }) {
  const urlPattern = operation === null ? '/i/api/graphql/' : `/i/api/graphql/.*${operation}`;
  const captured = await session.capturedResponses({ url_pattern: urlPattern });
  for (const response of captured.responses) {
    if (response.body !== null && typeof response.body === 'object') {
      harvest(response.body, typenames, sink, seen);
    }
  }
}

const oldestOf = (tweets) => Math.min(...tweets.map((tweet) => createdMs(tweet) ?? Number.POSITIVE_INFINITY));

/** Navigate, then scroll until `limit` nodes of `typenames` are read, the feed stops growing, or it reaches past `sinceMs`. */
export async function capture(session, { limit, operation = null, prepare, sinceMs = null, typenames, url, warm = false }) {
  const sink = [];
  const seen = new Set();
  await open(session, url, warm);
  await prepare?.(session);
  await harvestCaptured(session, { operation, seen, sink, typenames });
  let stale = 0;
  for (let index = 0; index < MAX_SCROLLS && sink.length < limit && stale < STALE_SCROLLS_FOR_END; index += 1) {
    if (sinceMs !== null && oldestOf(sink) < sinceMs) {
      break;
    }
    const before = sink.length;
    await humanScroll(session, index);
    await harvestCaptured(session, { operation, seen, sink, typenames });
    stale = sink.length === before ? stale + 1 : 0;
  }
  const kept = sinceMs === null ? sink : sink.filter((tweet) => (createdMs(tweet) ?? sinceMs) >= sinceMs);
  return kept.slice(0, limit);
}

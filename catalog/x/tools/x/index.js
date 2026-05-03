#!/usr/bin/env node
// spwn:x — catalog tool. Exposes X (Twitter) reads + writes as MCP
// tools served via the gate. All scrape logic talks to the gate's
// browser sidecar through @spwn/gate-tool's openSession() — we
// never touch Playwright directly here.
//
// Same wire format as the previous x-browse helper: agents (and
// publish.sh, via CLI) can keep calling fetch-favorites, post-tweet,
// etc. The migration is invisible at the tool boundary.

const { Tool, openSession } = require('@spwn/gate-tool');

const PROVIDER = 'x';
const NAV_TIMEOUT = 25000;
const POST_NAV_WAIT = 2500;
const SCROLL_WAIT = 1800;
const MAX_SCROLLS = 60;
const STALE_SCROLLS_FOR_END = 3;

// ── Stealth helpers ─────────────────────────────────────────────
// X has been getting more aggressive about silently dropping data
// XHRs when it detects automation patterns: identical scroll deltas,
// uniform timing, no mouse movement, no idle "reading" pauses. The
// page shell loads fine but the GraphQL data fetch never fires.
// These helpers introduce human-like jitter.

function jitter(min, max) { return min + Math.floor(Math.random() * (max - min)); }

// Replacement for the robotic `s.scroll({delta_y: 4000, wait_ms: 1800})`
// loop. Each step picks a different delta + post-scroll wait, and
// every few scrolls inserts a longer "reading" pause. Returns when
// the iteration is done — drop-in for a single scroll call.
async function humanScroll(s, scrollIndex) {
  const delta = jitter(2800, 5400);
  const wait = jitter(1400, 2900);
  await s.scroll({ delta_y: delta, count: 1, wait_ms: wait });
  // Every ~4 scrolls, pretend to read.
  if (scrollIndex > 0 && scrollIndex % jitter(3, 5) === 0) {
    await sleep(jitter(2500, 5500));
  }
}

// Warm up the session before hitting a content-heavy URL. A "cold"
// session that lands directly on /<handle>/likes is a stronger bot
// signal than one that pokes around /home first. ALSO dismisses the
// EU cookie-consent banner — when shown, it blocks content render
// (we found content stuck behind it on /likes and /following).
async function warmUp(s) {
  try {
    await s.goto('https://x.com/home', { wait_until: 'domcontentloaded', timeout_ms: NAV_TIMEOUT });
    await sleep(jitter(1500, 3000));
    await dismissCookieBanner(s);
    await s.scroll({ delta_y: jitter(800, 1600), count: 1, wait_ms: jitter(1200, 2200) });
  } catch (_) { /* best-effort — if /home fails, push on to the real target */ }
}

// X's EU cookie banner blocks rendering of timeline content (we
// confirmed: with banner up, /jterrazz/likes silently shows "account
// doesn't exist"). Click "Refuse non-essential" if visible — preserves
// privacy AND unblocks content. Idempotent / silent if not shown.
async function dismissCookieBanner(s) {
  try {
    await s.eval(`
      (function() {
        const all = Array.from(document.querySelectorAll('button, [role="button"]'));
        for (const b of all) {
          const t = (b.textContent || '').trim().toLowerCase();
          if (t.startsWith('refuse non-essential') ||
              t.startsWith('refuser les cookies non essentiels') ||
              t.startsWith('reject non-essential')) {
            b.click();
            return 'refused';
          }
        }
        return 'not-shown';
      })();
    `);
    await sleep(jitter(800, 1500));
  } catch (_) { /* swallow — banner gone or DOM changed */ }
}

// ── Shared helpers ──────────────────────────────────────────────

// Robust author extraction. The GraphQL shape varies — promoted
// tweets, ads, "X ranked" filler, and tombstones can lack the
// canonical user_results.result.core.screen_name. We try multiple
// fallbacks before giving up; with screen_name still missing we
// surface a numeric placeholder so the row isn't lost to nullity.
function findAuthor(tw) {
  let found = {};
  (function walk(n) {
    if (!n || typeof n !== 'object' || found.screen_name) return;
    // Canonical: user_results.result with core.screen_name
    const r = n.user_results?.result;
    if (r) {
      const sn = r.core?.screen_name || r.legacy?.screen_name;
      const nm = r.core?.name || r.legacy?.name || null;
      if (sn) { found = { screen_name: sn, name: nm }; return; }
    }
    // Fallback 1: flat user object (sometimes promoted-content shape)
    if (typeof n.screen_name === 'string') {
      found = { screen_name: n.screen_name, name: n.name || null };
      return;
    }
    // Fallback 2: result has author wrapped under different keys
    if (n.user?.legacy?.screen_name) {
      found = { screen_name: n.user.legacy.screen_name, name: n.user.legacy.name || null };
      return;
    }
    for (const v of Array.isArray(n) ? n : Object.values(n)) walk(v);
  })(tw);
  return found;
}

function harvestTweets(node, sink, seen) {
  if (!node || typeof node !== 'object') return;
  const t = node.__typename;
  if ((t === 'Tweet' || t === 'TweetWithVisibilityResults') && node.rest_id && !seen.has(node.rest_id)) {
    seen.add(node.rest_id);
    sink.push(t === 'TweetWithVisibilityResults' ? node.tweet : node);
  }
  if (Array.isArray(node)) for (const v of node) harvestTweets(v, sink, seen);
  else for (const k of Object.keys(node)) harvestTweets(node[k], sink, seen);
}

function stripSourceHtml(src) {
  const m = (src || '').match(/>([^<]+)</);
  return m ? m[1] : null;
}

function extractMedia(legacy) {
  const items = legacy?.extended_entities?.media || legacy?.entities?.media || [];
  if (!items.length) return null;
  return items.map((m) => {
    const out = { type: m.type, url: m.media_url_https || null };
    if (m.video_info?.variants?.length) {
      const mp4 = m.video_info.variants
        .filter((v) => v.content_type === 'video/mp4')
        .sort((a, b) => (b.bitrate || 0) - (a.bitrate || 0))[0];
      if (mp4) out.video_url = mp4.url;
      if (m.video_info.duration_millis) out.duration_ms = m.video_info.duration_millis;
    }
    return out;
  });
}

// Pull external link entities so the dashboard can render article
// preview cards instead of opaque t.co shorturls. Twitter exposes
// each URL as { url (t.co), expanded_url (real), display_url (clean),
// indices } in legacy.entities.urls. Self-references (status links
// to the same tweet) and Twitter-internal URLs are filtered.
function extractUrls(legacy) {
  const urls = legacy?.entities?.urls || [];
  if (!urls.length) return null;
  const out = urls
    .filter((u) => u?.expanded_url)
    .map((u) => ({
      url: u.url || null,                  // the t.co shortlink
      expanded_url: u.expanded_url,        // the real article URL
      display_url: u.display_url || null,  // human-readable form
    }))
    .filter((u) => {
      try {
        const host = new URL(u.expanded_url).hostname;
        // Drop x.com / twitter.com self-links — those are quote-tweet
        // markers, surfaced separately via .quoted
        return host !== 'x.com' && host !== 'twitter.com';
      } catch { return false; }
    });
  return out.length ? out : null;
}

function tweetToDict(tw, depth = 0) {
  const legacy = tw.legacy || tw.tweet?.legacy || {};
  const author = findAuthor(tw);
  const id = tw.rest_id || legacy.id_str || '';
  const screen = author.screen_name || null;

  const retweetedResult = legacy.retweeted_status_result?.result;
  const quotedResult = tw.quoted_status_result?.result || legacy.quoted_status_result?.result;
  const isRetweet = !!retweetedResult || /^RT @\w+:/.test(legacy.full_text || '');
  const isQuote = !!quotedResult || legacy.is_quote_status === true;
  const isReply = !!legacy.in_reply_to_status_id_str;
  const views = tw.views?.count != null ? parseInt(tw.views.count, 10) : null;

  return {
    id: String(id),
    url: id ? `https://x.com/${screen || 'i/web'}/status/${id}` : null,
    user: screen,
    user_name: author.name || null,
    text: legacy.full_text || legacy.text || '',
    created_at: legacy.created_at || null,
    likes: legacy.favorite_count || 0,
    retweets: legacy.retweet_count || 0,
    replies: legacy.reply_count || 0,
    quotes: legacy.quote_count || 0,
    views,
    is_retweet: isRetweet,
    is_quote: isQuote,
    is_reply: isReply,
    in_reply_to: legacy.in_reply_to_status_id_str || null,
    in_reply_to_user: legacy.in_reply_to_screen_name || null,
    source: stripSourceHtml(tw.source),
    media: extractMedia(legacy),
    urls: extractUrls(legacy),
    quoted: depth === 0 && quotedResult ? tweetToDict(quotedResult, 1) : null,
    retweeted: depth === 0 && retweetedResult ? tweetToDict(retweetedResult, 1) : null,
    lang: legacy.lang || null,
  };
}

// Walks a GraphQL response and yields user_results objects (the
// shape used by /<handle>/following and /<handle>/followers).
function harvestUsers(node, sink, seen) {
  if (!node || typeof node !== 'object') return;
  if (node.__typename === 'User' && node.rest_id && !seen.has(node.rest_id)) {
    seen.add(node.rest_id);
    sink.push(node);
  }
  if (Array.isArray(node)) for (const v of node) harvestUsers(v, sink, seen);
  else for (const k of Object.keys(node)) harvestUsers(node[k], sink, seen);
}

function userToDict(u) {
  const legacy = u.legacy || {};
  const core = u.core || {};
  return {
    id: String(u.rest_id || legacy.id_str || ''),
    handle: core.screen_name || legacy.screen_name || null,
    name: core.name || legacy.name || null,
    bio: legacy.description || null,
    followers: legacy.followers_count || 0,
    following: legacy.friends_count || 0,
    tweets: legacy.statuses_count || 0,
    verified: !!(u.is_blue_verified || legacy.verified),
    location: legacy.location || null,
    url: legacy.url || null,
    profile_image: legacy.profile_image_url_https || null,
    created_at: core.created_at || legacy.created_at || null,
  };
}

function tweetCreatedFromResponse(json) {
  const result = json?.data?.create_tweet?.tweet_results?.result;
  if (!result || !result.rest_id) return null;
  const dict = tweetToDict(result);
  return { external_id: dict.id, url: dict.url, user: dict.user };
}

// Captures GraphQL XHRs matching the regex. `withSession` opens a
// session, runs the body with it, and always closes — even on throw.
async function withSession(fn) {
  const s = await openSession(PROVIDER);
  try { return await fn(s); } finally { await s.end(); }
}

// Navigate + scroll until we have `limit` tweets or hit end-of-feed.
// Sidecar's captured-responses gives us all matching XHRs since
// session start, body included.
//
// `opNamePattern` is now optional — when omitted, we harvest from
// ALL graphql responses on the page. Useful when X renames an
// operation (e.g. Likes → ProfileLikesTimeline) or when bot-detection
// is strict enough that we can't rely on a known op name.
async function captureFeed(s, url, opNamePattern, limit, postNavHook = null, opts = {}) {
  const { warmup = false, sinceMs = null } = opts;
  const tweets = [];
  const seen = new Set();
  if (warmup) await warmUp(s);
  await s.goto(url, { wait_until: 'domcontentloaded', timeout_ms: NAV_TIMEOUT });
  await sleep(jitter(2200, 3800));
  // Banner can re-appear after navigation if first dismiss didn't take.
  await dismissCookieBanner(s);
  if (postNavHook) await postNavHook(s);

  // Initial harvest from any responses already received during goto.
  await harvestFromCaptured(s, opNamePattern, tweets, seen);

  // When a sinceMs boundary is set, we early-stop the scroll once the
  // OLDEST captured tweet falls below it — the timeline is reverse-chrono
  // so anything further down is older still. Saves 30-50 scrolls on
  // cron-friendly "give me the last 24h" calls.
  const oldestMs = () => {
    let min = Infinity;
    for (const tw of tweets) {
      const t = tweetCreatedMs(tw);
      if (t != null && t < min) min = t;
    }
    return Number.isFinite(min) ? min : null;
  };

  let stale = 0;
  let last = tweets.length;
  for (let i = 0; i < MAX_SCROLLS && tweets.length < limit && stale < STALE_SCROLLS_FOR_END; i++) {
    if (sinceMs != null) {
      const om = oldestMs();
      if (om != null && om < sinceMs) break; // crossed the boundary
    }
    await humanScroll(s, i);
    await harvestFromCaptured(s, opNamePattern, tweets, seen);
    if (tweets.length === last) stale++; else { stale = 0; last = tweets.length; }
  }

  // Final filter — drop anything older than sinceMs (the boundary may
  // have been hit mid-scroll, so the last batch can include stragglers).
  let out = tweets;
  if (sinceMs != null) out = out.filter((tw) => {
    const t = tweetCreatedMs(tw);
    return t == null || t >= sinceMs;
  });
  return out.slice(0, limit);
}

async function harvestFromCaptured(s, opNamePattern, sink, seen) {
  // When opNamePattern is empty/null we match all graphql responses
  // — useful when X has renamed an op or bot-detection is masking the
  // real op name.
  const url_pattern = opNamePattern ? '/i/api/graphql/.*' + opNamePattern : '/i/api/graphql/';
  const r = await s.capturedResponses({ url_pattern });
  for (const resp of r.responses) {
    if (resp.body && typeof resp.body === 'object') harvestTweets(resp.body, sink, seen);
  }
}

// Same scroll loop as captureFeed, but yields User objects instead
// of Tweet objects. Used by fetch-following. Same op-name fallback
// behaviour.
async function captureUsers(s, url, opNamePattern, limit, opts = {}) {
  const { warmup = false } = opts;
  const users = [];
  const seen = new Set();
  if (warmup) await warmUp(s);
  await s.goto(url, { wait_until: 'domcontentloaded', timeout_ms: NAV_TIMEOUT });
  await sleep(jitter(2200, 3800));
  await dismissCookieBanner(s);

  const harvest = async () => {
    const url_pattern = opNamePattern ? '/i/api/graphql/.*' + opNamePattern : '/i/api/graphql/';
    const r = await s.capturedResponses({ url_pattern });
    for (const resp of r.responses) {
      if (resp.body && typeof resp.body === 'object') harvestUsers(resp.body, users, seen);
    }
  };

  await harvest();

  let stale = 0;
  let last = users.length;
  for (let i = 0; i < MAX_SCROLLS && users.length < limit && stale < STALE_SCROLLS_FOR_END; i++) {
    await humanScroll(s, i);
    await harvest();
    if (users.length === last) stale++; else { stale = 0; last = users.length; }
  }
  return users.slice(0, limit);
}

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)); }

// ── Tool registration ───────────────────────────────────────────

const tool = new Tool({
  name: PROVIDER,
  title: 'X (Twitter)',
  version: '0.2.0',
});

const intArg = (a, k, d) => (a[k] != null ? parseInt(a[k], 10) : d);

// Parse a `--since` argument into a JS millis timestamp (UTC). Accepts:
//   • absolute ISO date    "2026-05-02"      → 2026-05-02T00:00:00Z
//   • absolute ISO datetime "2026-05-02T06:00:00Z"
//   • relative duration    "26h" / "3d" / "120m"
// Returns null when the arg is missing or unparseable. Used by the
// scroll-based methods (fetch-home/favorites/likes/account/search) to
// (1) stop scrolling once the timeline crosses the boundary and (2)
// filter the final result list — saving real wall time on cron loops
// that only care about "what's new since X".
function parseSince(raw) {
  if (raw == null) return null;
  const s = String(raw).trim();
  if (!s) return null;
  // Relative: <num><unit> with unit in m/h/d
  const rel = s.match(/^(\d+)\s*([mhd])$/i);
  if (rel) {
    const n = parseInt(rel[1], 10);
    const unit = rel[2].toLowerCase();
    const ms = unit === 'm' ? 60_000 : unit === 'h' ? 3_600_000 : 86_400_000;
    return Date.now() - n * ms;
  }
  // Absolute: pad bare YYYY-MM-DD to start-of-day UTC, otherwise let
  // Date parse it. Reject NaN.
  const padded = /^\d{4}-\d{2}-\d{2}$/.test(s) ? s + 'T00:00:00Z' : s;
  const t = Date.parse(padded);
  return Number.isNaN(t) ? null : t;
}

// Tweet `created_at` is the legacy Twitter format ("Wed Oct 30
// 18:25:15 +0000 2024"). Date.parse handles it natively.
function tweetCreatedMs(tw) {
  const c = tw?.legacy?.created_at || tw?.tweet?.legacy?.created_at;
  if (!c) return null;
  const t = Date.parse(c);
  return Number.isNaN(t) ? null : t;
}

tool.method('fetch-home', {
  description: 'Fetch your home timeline. feed="following" (default, chronological) or feed="for-you" (algorithmic). Pass --since for date-bounded fetches (early-stops scroll once timeline crosses the boundary).',
  schema: {
    type: 'object',
    properties: {
      feed: { type: 'string', enum: ['following', 'for-you'], description: 'Which timeline tab' },
      limit: { type: 'integer', description: 'Max tweets (default 50)' },
      since: { type: 'string', description: 'ISO date (YYYY-MM-DD), ISO datetime, or duration (e.g. "26h", "3d"). Stops scroll once timeline crosses this boundary and filters out older items.' },
    },
  },
  async handler({ args }) {
    const feed = args.feed || 'following';
    const limit = intArg(args, 'limit', 50);
    const sinceMs = parseSince(args.since);
    const wantFollowing = feed === 'following';
    const op = wantFollowing ? 'HomeLatestTimeline' : 'HomeTimeline';
    return withSession(async (s) => {
      const tw = await captureFeed(s, 'https://x.com/home', op, limit, async (s) => {
        if (!wantFollowing) return;
        // Click the "Following" tab if it isn't already active.
        try {
          await s.eval(`
            (function() {
              const tabs = Array.from(document.querySelectorAll('[role="tab"]'));
              for (const t of tabs) {
                const txt = (t.textContent || '').trim();
                if ((txt === 'Following' || txt === 'Abonnements') &&
                    t.getAttribute('aria-selected') !== 'true') {
                  t.click();
                  return;
                }
              }
            })();
          `);
          await sleep(2000);
        } catch (_) { /* tab may not be present yet — fall through */ }
      }, { sinceMs });
      const items = tw.map((t) => tweetToDict(t)).filter((t) => t && t.user && t.id);
      return { feed: wantFollowing ? 'following' : 'for-you', items, count: items.length, since: args.since ?? null };
    });
  },
});

tool.method('fetch-favorites', {
  description: "Fetch the authenticated user's bookmarked tweets.",
  schema: { type: 'object', properties: { limit: { type: 'integer' } } },
  async handler({ args }) {
    const limit = intArg(args, 'limit', 50);
    return withSession(async (s) => {
      const tw = await captureFeed(s, 'https://x.com/i/bookmarks', 'Bookmarks', limit);
      const items = tw.map((t) => tweetToDict(t)).filter((t) => t && t.user && t.id);
      return { items, count: items.length };
    });
  },
});

tool.method('fetch-likes', {
  description: "Fetch a user's liked tweets. Requires being logged in as that user (likes are private to others).",
  schema: {
    type: 'object',
    properties: {
      handle: { type: 'string', description: 'X handle (without @). Defaults to env X_HANDLE.' },
      limit: { type: 'integer', description: 'Max tweets (default 50)' },
    },
  },
  async handler({ args }) {
    const handle = String(args.handle || process.env.X_HANDLE || '').replace(/^@/, '');
    if (!handle) throw new Error('handle is required (pass --handle or set X_HANDLE)');
    const limit = intArg(args, 'limit', 50);
    return withSession(async (s) => {
      // Stealth path: warm up via /home, jittered scrolls, and harvest
      // tweets from ALL graphql responses on the page (X has renamed
      // the Likes op at least once and bot-detection silently drops
      // the data XHR if we look too robotic). Using opNamePattern=null
      // is fine here — only the likes-tab graphql responses contain
      // Tweet objects on this page.
      const tw = await captureFeed(s, `https://x.com/${handle}/likes`, null, limit, null, { warmup: true });
      const items = tw.map((t) => tweetToDict(t)).filter((t) => t && t.user && t.id);
      return { handle, items, count: items.length };
    });
  },
});

tool.method('fetch-following', {
  description: "Fetch the accounts a user follows.",
  schema: {
    type: 'object',
    properties: {
      handle: { type: 'string', description: 'X handle (without @). Defaults to env X_HANDLE.' },
      limit: { type: 'integer', description: 'Max users (default 200)' },
    },
  },
  async handler({ args }) {
    const handle = String(args.handle || process.env.X_HANDLE || '').replace(/^@/, '');
    if (!handle) throw new Error('handle is required (pass --handle or set X_HANDLE)');
    const limit = intArg(args, 'limit', 200);
    return withSession(async (s) => {
      // Stealth path: warm up + permissive harvest. Note: sidebar
      // recommendations also surface User objects; the dedup-by-rest_id
      // means at most ~5-10 sidebar accounts can leak in. Acceptable
      // noise vs the alternative of fetching nothing.
      const us = await captureUsers(s, `https://x.com/${handle}/following`, null, limit, { warmup: true });
      return { handle, items: us.map(userToDict), count: us.length };
    });
  },
});

tool.method('fetch-account', {
  description: 'Fetch recent tweets from a specific X handle (without the @).',
  schema: {
    type: 'object',
    properties: {
      handle: { type: 'string', description: 'X handle, e.g. "karpathy"' },
      limit: { type: 'integer' },
    },
    required: ['handle'],
  },
  async handler({ args }) {
    if (!args.handle) throw new Error('handle is required');
    const handle = String(args.handle).replace(/^@/, '');
    const limit = intArg(args, 'limit', 50);
    return withSession(async (s) => {
      const tw = await captureFeed(s, `https://x.com/${handle}`, 'UserTweets', limit);
      const items = tw.map((t) => tweetToDict(t)).filter((t) => t && t.user && t.id);
      return { handle, items, count: items.length };
    });
  },
});

tool.method('search', {
  description: 'Search X for tweets matching a query (latest first).',
  schema: {
    type: 'object',
    properties: {
      query: { type: 'string' },
      limit: { type: 'integer' },
    },
    required: ['query'],
  },
  async handler({ args }) {
    if (!args.query) throw new Error('query is required');
    const limit = intArg(args, 'limit', 50);
    const url = `https://x.com/search?q=${encodeURIComponent(args.query)}&f=live`;
    return withSession(async (s) => {
      const tw = await captureFeed(s, url, 'SearchTimeline', limit);
      const items = tw.map((t) => tweetToDict(t)).filter((t) => t && t.user && t.id);
      return { query: args.query, items, count: items.length };
    });
  },
});

tool.method('fetch-thread', {
  description: 'Fetch a tweet plus its conversation context (replies).',
  schema: {
    type: 'object',
    properties: {
      tweet_id: { type: 'string', description: 'Numeric tweet id' },
      limit: { type: 'integer', description: 'Max context tweets (default 100)' },
    },
    required: ['tweet_id'],
  },
  async handler({ args }) {
    if (!args.tweet_id) throw new Error('tweet_id is required');
    const id = String(args.tweet_id);
    const limit = intArg(args, 'limit', 100);
    return withSession(async (s) => {
      const tw = await captureFeed(s, `https://x.com/i/web/status/${id}`, 'TweetDetail', limit);
      if (!tw.length) return { error: `tweet ${id} not found` };
      const focal = tw.find((t) => (t.rest_id || '') === id) || tw[0];
      const context = tw.filter((t) => t !== focal).map((t) => tweetToDict(t));
      return { tweet: tweetToDict(focal), context, count: context.length };
    });
  },
});

tool.method('post-tweet', {
  description: 'Publish a tweet from the authenticated account. NOT for direct agent use — call from publish.sh after human approval.',
  schema: {
    type: 'object',
    properties: { text: { type: 'string' } },
    required: ['text'],
  },
  async handler({ args }) {
    const text = String(args.text || '');
    if (!text) throw new Error('text is empty');
    if (text.length > 280) throw new Error(`text is ${text.length} chars (max 280)`);
    return withSession(async (s) => {
      await s.goto('https://x.com/compose/post', { wait_until: 'domcontentloaded', timeout_ms: NAV_TIMEOUT });
      await s.waitSelector('[data-testid="tweetTextarea_0"]', { timeout_ms: 12000 });
      await s.type('[data-testid="tweetTextarea_0"]', text);
      await sleep(600);
      const respPromise = s.waitResponse('/CreateTweet', { method: 'POST', timeout_ms: 20000 });
      await s.click('[data-testid="tweetButton"]');
      const resp = await respPromise;
      const out = tweetCreatedFromResponse(resp.body);
      if (!out) throw new Error(`post failed: ${resp.body?.errors?.[0]?.message || 'no result'}`);
      return out;
    });
  },
});

tool.method('reply-tweet', {
  description: 'Reply to a tweet from the authenticated account. NOT for direct agent use.',
  schema: {
    type: 'object',
    properties: {
      text: { type: 'string' },
      in_reply_to: { type: 'string', description: 'Numeric tweet id' },
    },
    required: ['text', 'in_reply_to'],
  },
  async handler({ args }) {
    const text = String(args.text || '');
    const target = String(args.in_reply_to || '');
    if (!text) throw new Error('text is empty');
    if (text.length > 280) throw new Error(`text is ${text.length} chars (max 280)`);
    if (!/^\d+$/.test(target)) throw new Error('in_reply_to must be a numeric tweet id');
    return withSession(async (s) => {
      await s.goto(`https://x.com/i/web/status/${target}`, { wait_until: 'domcontentloaded', timeout_ms: NAV_TIMEOUT });
      await s.waitSelector('[data-testid="tweetTextarea_0"]', { timeout_ms: 12000 });
      await s.type('[data-testid="tweetTextarea_0"]', text);
      await sleep(600);
      await s.waitSelector('[data-testid="tweetButtonInline"]', { timeout_ms: 5000 });
      const respPromise = s.waitResponse('/CreateTweet', { method: 'POST', timeout_ms: 20000 });
      await s.click('[data-testid="tweetButtonInline"]');
      const resp = await respPromise;
      const out = tweetCreatedFromResponse(resp.body);
      if (!out) throw new Error(`reply failed: ${resp.body?.errors?.[0]?.message || 'no result'}`);
      return { ...out, in_reply_to: target };
    });
  },
});

tool.run();

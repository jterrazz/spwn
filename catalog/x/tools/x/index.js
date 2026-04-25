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

// ── Shared helpers ──────────────────────────────────────────────

function findAuthor(tw) {
  let found = {};
  (function walk(n) {
    if (!n || typeof n !== 'object' || found.screen_name) return;
    if (n.user_results?.result) {
      const r = n.user_results.result;
      const sn = r.core?.screen_name || r.legacy?.screen_name;
      if (sn) {
        found = { screen_name: sn, name: r.core?.name || r.legacy?.name || null };
        return;
      }
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
    quoted: depth === 0 && quotedResult ? tweetToDict(quotedResult, 1) : null,
    retweeted: depth === 0 && retweetedResult ? tweetToDict(retweetedResult, 1) : null,
    lang: legacy.lang || null,
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
async function captureFeed(s, url, opNamePattern, limit, postNavHook = null) {
  const tweets = [];
  const seen = new Set();
  await s.goto(url, { wait_until: 'domcontentloaded', timeout_ms: NAV_TIMEOUT });
  await sleep(POST_NAV_WAIT);
  if (postNavHook) await postNavHook(s);

  // Initial harvest from any responses already received during goto.
  await harvestFromCaptured(s, opNamePattern, tweets, seen);

  let stale = 0;
  let last = tweets.length;
  for (let i = 0; i < MAX_SCROLLS && tweets.length < limit && stale < STALE_SCROLLS_FOR_END; i++) {
    await s.scroll({ delta_y: 4000, count: 1, wait_ms: SCROLL_WAIT });
    await harvestFromCaptured(s, opNamePattern, tweets, seen);
    if (tweets.length === last) stale++; else { stale = 0; last = tweets.length; }
  }
  return tweets.slice(0, limit);
}

async function harvestFromCaptured(s, opNamePattern, sink, seen) {
  const r = await s.capturedResponses({ url_pattern: '/i/api/graphql/.*' + opNamePattern });
  for (const resp of r.responses) {
    if (resp.body && typeof resp.body === 'object') harvestTweets(resp.body, sink, seen);
  }
}

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)); }

// ── Tool registration ───────────────────────────────────────────

const tool = new Tool({
  name: PROVIDER,
  title: 'X (Twitter)',
  version: '0.2.0',
});

const intArg = (a, k, d) => (a[k] != null ? parseInt(a[k], 10) : d);

tool.method('fetch-home', {
  description: 'Fetch your home timeline. feed="following" (default, chronological) or feed="for-you" (algorithmic).',
  schema: {
    type: 'object',
    properties: {
      feed: { type: 'string', enum: ['following', 'for-you'], description: 'Which timeline tab' },
      limit: { type: 'integer', description: 'Max tweets (default 50)' },
    },
  },
  async handler({ args }) {
    const feed = args.feed || 'following';
    const limit = intArg(args, 'limit', 50);
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
      });
      return { feed: wantFollowing ? 'following' : 'for-you', items: tw.map((t) => tweetToDict(t)), count: tw.length };
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
      return { items: tw.map((t) => tweetToDict(t)), count: tw.length };
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
      return { handle, items: tw.map((t) => tweetToDict(t)), count: tw.length };
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
      return { query: args.query, items: tw.map((t) => tweetToDict(t)), count: tw.length };
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

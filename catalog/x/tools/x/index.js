#!/usr/bin/env node
// outside: spwn:x, a catalog tool the gate serves over MCP and a host script calls as a CLI; every read goes through
// the gate's browser sidecar with the user's session cookies, never through Playwright directly.

import { createRequire } from 'node:module';

import { handleOf, integerOf, sinceOf, tweetTextOf } from './arguments.js';
import { capture, NAV_TIMEOUT, sleep, withSession } from './browsing.js';
import { createdOf, rowsOf, tweetToDict, userToDict } from './tweets.js';

// outside: the gate resolves @spwn/gate-tool through NODE_PATH, which only a CommonJS require reads
const { Tool } = createRequire(import.meta.url)('@spwn/gate-tool');

const TWEETS = ['Tweet', 'TweetWithVisibilityResults'];
const USERS = ['User'];
const COMPOSER = '[data-testid="tweetTextarea_0"]';
const FOLLOWING_TAB = `(() => {
  for (const tab of document.querySelectorAll('[role="tab"]')) {
    const label = (tab.textContent || '').trim();
    if ((label === 'Following' || label === 'Abonnements') && tab.getAttribute('aria-selected') !== 'true') {
      tab.click();
      return;
    }
  }
})();`;

const tool = new Tool({ name: 'x', title: 'X (Twitter)', version: '0.2.0' });

const SINCE = {
  description: 'ISO date (YYYY-MM-DD), ISO datetime, or duration (e.g. "26h", "3d"). Stops scroll once timeline crosses this boundary and filters out older items.',
  type: 'string',
};

async function feed(url, operation, limit, options = {}) {
  return await withSession(async (session) => rowsOf(await capture(session, { limit, operation, typenames: TWEETS, url, ...options })));
}

async function selectFollowing(session) {
  try {
    await session.eval(FOLLOWING_TAB);
    await sleep(2000);
  } catch {
    // value: the tab may not be rendered yet; the For you timeline is read instead
  }
}

tool.method('fetch-home', {
  description: 'Fetch your home timeline. feed="following" (default, chronological) or feed="for-you" (algorithmic). Pass --since for date-bounded fetches (early-stops scroll once timeline crosses the boundary).',
  async handler({ args }) {
    const following = (args.feed ?? 'following') === 'following';
    const items = await feed('https://x.com/home', following ? 'HomeLatestTimeline' : 'HomeTimeline', integerOf(args, 'limit', 50), {
      prepare: following ? selectFollowing : undefined,
      sinceMs: sinceOf(args.since),
    });
    return { count: items.length, feed: following ? 'following' : 'for-you', items, since: args.since ?? null };
  },
  schema: {
    properties: {
      feed: { description: 'Which timeline tab', enum: ['following', 'for-you'], type: 'string' },
      limit: { description: 'Max tweets (default 50)', type: 'integer' },
      since: SINCE,
    },
    type: 'object',
  },
});

tool.method('fetch-favorites', {
  description: "Fetch the authenticated user's bookmarked tweets.",
  async handler({ args }) {
    const items = await feed('https://x.com/i/bookmarks', 'Bookmarks', integerOf(args, 'limit', 50));
    return { count: items.length, items };
  },
  schema: { properties: { limit: { type: 'integer' } }, type: 'object' },
});

// outside: the likes and following pages are read warmed up and from every GraphQL response, since X has renamed the Likes operation and drops the data of a session that looks robotic
tool.method('fetch-likes', {
  description: "Fetch a user's liked tweets. Requires being logged in as that user (likes are private to others).",
  async handler({ args }) {
    const handle = handleOf(args);
    const items = await feed(`https://x.com/${handle}/likes`, null, integerOf(args, 'limit', 50), { warm: true });
    return { count: items.length, handle, items };
  },
  schema: {
    properties: {
      handle: { description: 'X handle (without @). Defaults to env X_HANDLE.', type: 'string' },
      limit: { description: 'Max tweets (default 50)', type: 'integer' },
    },
    type: 'object',
  },
});

tool.method('fetch-following', {
  description: 'Fetch the accounts a user follows.',
  async handler({ args }) {
    const handle = handleOf(args);
    const users = await withSession(async (session) =>
      await capture(session, { limit: integerOf(args, 'limit', 200), typenames: USERS, url: `https://x.com/${handle}/following`, warm: true }),
    );
    return { count: users.length, handle, items: users.map((user) => userToDict(user)) };
  },
  schema: {
    properties: {
      handle: { description: 'X handle (without @). Defaults to env X_HANDLE.', type: 'string' },
      limit: { description: 'Max users (default 200)', type: 'integer' },
    },
    type: 'object',
  },
});

tool.method('fetch-account', {
  description: 'Fetch recent tweets from a specific X handle (without the @).',
  async handler({ args }) {
    const handle = handleOf({ handle: args.handle ?? '' });
    const items = await feed(`https://x.com/${handle}`, 'UserTweets', integerOf(args, 'limit', 50));
    return { count: items.length, handle, items };
  },
  schema: { properties: { handle: { description: 'X handle, e.g. "karpathy"', type: 'string' }, limit: { type: 'integer' } }, required: ['handle'], type: 'object' },
});

tool.method('search', {
  description: 'Search X for tweets matching a query (latest first).',
  async handler({ args }) {
    if (typeof args.query !== 'string' || args.query === '') {
      throw new Error('query is required');
    }
    const url = `https://x.com/search?q=${encodeURIComponent(args.query)}&f=live`;
    const items = await feed(url, 'SearchTimeline', integerOf(args, 'limit', 50));
    return { count: items.length, items, query: args.query };
  },
  schema: { properties: { limit: { type: 'integer' }, query: { type: 'string' } }, required: ['query'], type: 'object' },
});

tool.method('fetch-thread', {
  description: 'Fetch a tweet plus its conversation context (replies).',
  async handler({ args }) {
    const id = String(args.tweet_id ?? '');
    if (id === '') {
      throw new Error('tweet_id is required');
    }
    const tweets = await withSession(async (session) =>
      await capture(session, { limit: integerOf(args, 'limit', 100), operation: 'TweetDetail', typenames: TWEETS, url: `https://x.com/i/web/status/${id}` }),
    );
    if (tweets.length === 0) {
      return { error: `tweet ${id} not found` };
    }
    const focal = tweets.find((tweet) => tweet.rest_id === id) ?? tweets[0];
    const context = tweets.filter((tweet) => tweet !== focal).map((tweet) => tweetToDict(tweet));
    return { context, count: context.length, tweet: tweetToDict(focal) };
  },
  schema: {
    properties: { limit: { description: 'Max context tweets (default 100)', type: 'integer' }, tweet_id: { description: 'Numeric tweet id', type: 'string' } },
    required: ['tweet_id'],
    type: 'object',
  },
});

async function publish(url, text, button) {
  return await withSession(async (session) => {
    await session.goto(url, { timeout_ms: NAV_TIMEOUT, wait_until: 'domcontentloaded' });
    await session.waitSelector(COMPOSER, { timeout_ms: 12_000 });
    await session.type(COMPOSER, text);
    await sleep(600);
    await session.waitSelector(button, { timeout_ms: 5000 });
    const created = session.waitResponse('/CreateTweet', { method: 'POST', timeout_ms: 20_000 });
    await session.click(button);
    const response = await created;
    const posted = createdOf(response.body);
    if (posted === null) {
      throw new Error(`post failed: ${response.body?.errors?.[0]?.message ?? 'no result'}`);
    }
    return posted;
  });
}

tool.method('post-tweet', {
  description: 'Publish a tweet from the authenticated account. NOT for direct agent use: call it from publish.sh after human approval.',
  async handler({ args }) {
    return await publish('https://x.com/compose/post', tweetTextOf(args), '[data-testid="tweetButton"]');
  },
  schema: { properties: { text: { type: 'string' } }, required: ['text'], type: 'object' },
});

tool.method('reply-tweet', {
  description: 'Reply to a tweet from the authenticated account. NOT for direct agent use.',
  async handler({ args }) {
    const text = tweetTextOf(args);
    const target = String(args.in_reply_to ?? '');
    if (!/^\d+$/u.test(target)) {
      throw new Error('in_reply_to must be a numeric tweet id');
    }
    return { ...(await publish(`https://x.com/i/web/status/${target}`, text, '[data-testid="tweetButtonInline"]')), in_reply_to: target };
  },
  schema: {
    properties: { in_reply_to: { description: 'Numeric tweet id', type: 'string' }, text: { type: 'string' } },
    required: ['text', 'in_reply_to'],
    type: 'object',
  },
});

await tool.run();

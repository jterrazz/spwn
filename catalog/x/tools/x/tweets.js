// outside: X's GraphQL shapes vary — promoted tweets, ads and tombstones carry their author elsewhere — so every
// reader here falls back rather than drop the row.

const SELF_HOSTS = new Set(['twitter.com', 'x.com']);

const firstOf = (...values) => values.find((value) => value !== undefined && value !== null) ?? null;

function canonicalAuthor(node) {
  const result = node.user_results?.result;
  const screenName = firstOf(result?.core?.screen_name, result?.legacy?.screen_name);
  return screenName === null ? null : { name: firstOf(result.core?.name, result.legacy?.name), screen_name: screenName };
}

function flatAuthor(node) {
  if (typeof node.screen_name === 'string') {
    return { name: firstOf(node.name), screen_name: node.screen_name };
  }
  const legacy = node.user?.legacy;
  return typeof legacy?.screen_name === 'string' ? { name: firstOf(legacy.name), screen_name: legacy.screen_name } : null;
}

/** The first author found walking a tweet node: the canonical user result, then the flat and wrapped shapes. */
export function findAuthor(node) {
  if (node === null || typeof node !== 'object') {
    return null;
  }
  const own = canonicalAuthor(node) ?? flatAuthor(node);
  if (own !== null) {
    return own;
  }
  for (const child of Array.isArray(node) ? node : Object.values(node)) {
    const found = findAuthor(child);
    if (found !== null) {
      return found;
    }
  }
  return null;
}

/** Every node of `typename` under `node`, each once by rest_id, appended to `sink`. */
export function harvest(node, typenames, sink, seen) {
  if (node === null || typeof node !== 'object') {
    return;
  }
  const { __typename: typename } = node;
  if (typenames.includes(typename) && typeof node.rest_id === 'string' && !seen.has(node.rest_id)) {
    seen.add(node.rest_id);
    sink.push(typename === 'TweetWithVisibilityResults' ? node.tweet : node);
  }
  for (const child of Array.isArray(node) ? node : Object.values(node)) {
    harvest(child, typenames, sink, seen);
  }
}

function sourceOf(html) {
  return />(?<source>[^<]+)</u.exec(html ?? '')?.groups?.source ?? null;
}

function mediaOf(legacy) {
  const items = legacy?.extended_entities?.media ?? legacy?.entities?.media ?? [];
  if (items.length === 0) {
    return null;
  }
  return items.map((item) => {
    const mp4 = (item.video_info?.variants ?? [])
      .filter((variant) => variant.content_type === 'video/mp4')
      .toSorted((left, right) => (right.bitrate ?? 0) - (left.bitrate ?? 0))[0];
    return {
      type: item.type,
      url: item.media_url_https ?? null,
      ...(mp4 === undefined ? {} : { video_url: mp4.url }),
      ...(item.video_info?.duration_millis === undefined ? {} : { duration_ms: item.video_info.duration_millis }),
    };
  });
}

function hostOf(url) {
  try {
    return new URL(url).hostname;
  } catch {
    return '';
  }
}

// value: the dashboard renders an article card from each external link; a link to x.com or twitter.com is a quote marker, surfaced as `quoted`
function urlsOf(legacy) {
  const links = (legacy?.entities?.urls ?? [])
    .filter((link) => typeof link?.expanded_url === 'string' && hostOf(link.expanded_url) !== '' && !SELF_HOSTS.has(hostOf(link.expanded_url)))
    .map((link) => ({ display_url: link.display_url ?? null, expanded_url: link.expanded_url, url: link.url ?? null }));
  return links.length === 0 ? null : links;
}

function kindsOf(tweet, legacy) {
  const retweeted = legacy.retweeted_status_result?.result;
  const quoted = tweet.quoted_status_result?.result ?? legacy.quoted_status_result?.result;
  return {
    is_quote: quoted !== undefined || legacy.is_quote_status === true,
    is_reply: typeof legacy.in_reply_to_status_id_str === 'string',
    is_retweet: retweeted !== undefined || /^RT @\w+:/u.test(legacy.full_text ?? ''),
    quoted,
    retweeted,
  };
}

function countsOf(tweet, legacy) {
  return {
    likes: firstOf(legacy.favorite_count, 0),
    quotes: firstOf(legacy.quote_count, 0),
    replies: firstOf(legacy.reply_count, 0),
    retweets: firstOf(legacy.retweet_count, 0),
    views: tweet.views?.count === undefined ? null : Number.parseInt(tweet.views.count, 10),
  };
}

function textsOf(tweet, legacy) {
  return {
    created_at: firstOf(legacy.created_at),
    in_reply_to: firstOf(legacy.in_reply_to_status_id_str),
    in_reply_to_user: firstOf(legacy.in_reply_to_screen_name),
    lang: firstOf(legacy.lang),
    source: sourceOf(tweet.source),
    text: firstOf(legacy.full_text, legacy.text, ''),
  };
}

/** The row the tool returns for one tweet node, its quoted and retweeted tweets one level deep. */
export function tweetToDict(tweet, depth = 0) {
  const legacy = firstOf(tweet.legacy, tweet.tweet?.legacy, {});
  const author = firstOf(findAuthor(tweet), {});
  const id = String(firstOf(tweet.rest_id, legacy.id_str, ''));
  const { quoted, retweeted, ...kinds } = kindsOf(tweet, legacy);
  const nested = depth === 0;
  return {
    id,
    media: mediaOf(legacy),
    quoted: nested && quoted !== undefined ? tweetToDict(quoted, 1) : null,
    retweeted: nested && retweeted !== undefined ? tweetToDict(retweeted, 1) : null,
    url: id === '' ? null : `https://x.com/${firstOf(author.screen_name, 'i/web')}/status/${id}`,
    urls: urlsOf(legacy),
    user: firstOf(author.screen_name),
    user_name: firstOf(author.name),
    ...textsOf(tweet, legacy),
    ...countsOf(tweet, legacy),
    ...kinds,
  };
}

/** The tweets worth returning: a row with an author and an id. */
export function rowsOf(tweets) {
  return tweets.map((tweet) => tweetToDict(tweet)).filter((row) => row.user !== null && row.id !== '');
}

/** The id, link and author of the tweet a CreateTweet response reports, or null when it reports none. */
export function createdOf(json) {
  const result = json?.data?.create_tweet?.tweet_results?.result;
  if (typeof result?.rest_id !== 'string') {
    return null;
  }
  const row = tweetToDict(result);
  return { external_id: row.id, url: row.url, user: row.user };
}

/** When a tweet was posted, in milliseconds, from X's legacy `created_at` format. */
export function createdMs(tweet) {
  const created = tweet?.legacy?.created_at ?? tweet?.tweet?.legacy?.created_at;
  const ms = created === undefined ? Number.NaN : Date.parse(created);
  return Number.isNaN(ms) ? null : ms;
}

/** The row the tool returns for one user node. */
export function userToDict(user) {
  const legacy = firstOf(user.legacy, {});
  const core = firstOf(user.core, {});
  return {
    bio: firstOf(legacy.description),
    created_at: firstOf(core.created_at, legacy.created_at),
    followers: firstOf(legacy.followers_count, 0),
    following: firstOf(legacy.friends_count, 0),
    handle: firstOf(core.screen_name, legacy.screen_name),
    id: String(firstOf(user.rest_id, legacy.id_str, '')),
    location: firstOf(legacy.location),
    name: firstOf(core.name, legacy.name),
    profile_image: firstOf(legacy.profile_image_url_https),
    tweets: firstOf(legacy.statuses_count, 0),
    url: firstOf(legacy.url),
    verified: user.is_blue_verified === true || legacy.verified === true,
  };
}

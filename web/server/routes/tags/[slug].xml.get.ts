// Per-tag RSS feed: /tags/<slug>.xml — published posts carrying the taxonomy
// slug. (The blog service filters by slug regardless of taxonomy kind, so this
// mirrors the category feed; the path distinguishes the subscriber's intent.)
export default defineEventHandler(async (event) => {
  const apiBase = useRuntimeConfig(event).apiBase
  const url = getRequestURL(event)
  const origin = url.origin
  // Derive the slug from the path (Nitro doesn't expose a `[slug].xml` param).
  const slug = decodeURIComponent((url.pathname.split('/').pop() || '').replace(/\.xml$/, ''))
  const posts = await fetchFeedPosts(apiBase, slug)
  const xml = buildRss({
    origin,
    feedPath: `/tags/${slug}.xml`,
    title: `博客 · 标签「${slug}」`,
    description: `标签「${slug}」下的文章`,
    posts
  })
  setHeader(event, 'content-type', 'application/rss+xml; charset=utf-8')
  return xml
})

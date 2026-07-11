// Per-category RSS feed: /categories/<slug>.xml — published posts carrying the
// taxonomy slug (the blog service filters by slug, same as the archive page).
export default defineEventHandler(async (event) => {
  const apiBase = useRuntimeConfig(event).apiBase
  const url = getRequestURL(event)
  const origin = url.origin
  // Derive the slug from the path (Nitro doesn't expose a `[slug].xml` param).
  const slug = decodeURIComponent((url.pathname.split('/').pop() || '').replace(/\.xml$/, ''))
  const posts = await fetchFeedPosts(apiBase, slug)
  const xml = buildRss({
    origin,
    feedPath: `/categories/${slug}.xml`,
    title: `博客 · 分类「${slug}」`,
    description: `分类「${slug}」下的文章`,
    posts
  })
  setHeader(event, 'content-type', 'application/rss+xml; charset=utf-8')
  return xml
})

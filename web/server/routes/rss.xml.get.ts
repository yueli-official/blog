// Site-wide RSS feed: all published posts. Served from the blog domain (Nitro)
// so readers subscribe to <site>/rss.xml. Reuses the server-side markdown
// renderer for content:encoded.
export default defineEventHandler(async (event) => {
  const apiBase = useRuntimeConfig(event).apiBase
  const origin = getRequestURL(event).origin
  const posts = await fetchFeedPosts(apiBase)
  const xml = buildRss({
    origin,
    feedPath: '/rss.xml',
    title: '博客',
    description: '想法、笔记与记录',
    posts
  })
  setHeader(event, 'content-type', 'application/rss+xml; charset=utf-8')
  return xml
})

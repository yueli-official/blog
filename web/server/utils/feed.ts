import { marked } from 'marked'

// FeedPost mirrors the public PostView fields the feed needs (products/blog/api).
export interface FeedPost {
  title: string
  slug: string
  excerpt?: string
  content?: string
  authorId?: string
  publishedAt?: string
  createdAt?: string
}

function esc(s = ''): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

// CDATA-wrap, guarding against a premature ]]> terminator inside the payload.
function cdata(s = ''): string {
  return `<![CDATA[${String(s).replace(/]]>/g, ']]]]><![CDATA[>')}]]>`
}

// RSS pubDate must be RFC-822; Date.toUTCString() emits a compliant form.
function rfc822(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '' : d.toUTCString()
}

// buildRss renders an RSS 2.0 document (with content:encoded full HTML) from a
// set of published posts. Pure string assembly — no table, the M3 "无表纯渲染".
export function buildRss(opts: {
  origin: string
  feedPath: string
  title: string
  description: string
  posts: FeedPost[]
}): string {
  const { origin, feedPath, title, description, posts } = opts
  const items = posts.map((p) => {
    const url = `${origin}/posts/${p.slug}`
    const date = rfc822(p.publishedAt || p.createdAt)
    const html = marked.parse(p.content ?? '', { async: false, gfm: true, breaks: true }) as string
    return [
      '    <item>',
      `      <title>${esc(p.title)}</title>`,
      `      <link>${esc(url)}</link>`,
      `      <guid isPermaLink="true">${esc(url)}</guid>`,
      date ? `      <pubDate>${date}</pubDate>` : '',
      p.authorId ? `      <dc:creator>${cdata(p.authorId.slice(0, 8))}</dc:creator>` : '',
      `      <description>${cdata(p.excerpt || '')}</description>`,
      `      <content:encoded>${cdata(html)}</content:encoded>`,
      '    </item>'
    ].filter(Boolean).join('\n')
  }).join('\n')

  return `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>${esc(title)}</title>
    <link>${esc(origin)}/</link>
    <description>${esc(description)}</description>
    <language>zh-CN</language>
    <lastBuildDate>${new Date().toUTCString()}</lastBuildDate>
    <atom:link href="${esc(origin + feedPath)}" rel="self" type="application/rss+xml"/>
${items}
  </channel>
</rss>
`
}

// fetchPosts pulls published posts from the blog service public list (optionally
// filtered by a taxonomy slug). Returns [] on any upstream error so the feed
// still renders (an empty but valid channel).
export async function fetchFeedPosts(apiBase: string, taxonomy?: string): Promise<FeedPost[]> {
  try {
    const res = await $fetch<{ data?: { items?: FeedPost[] } }>(`${apiBase}/api/v1/posts`, {
      query: { page: 1, size: 50, ...(taxonomy ? { taxonomy } : {}) }
    })
    return res?.data?.items ?? []
  } catch {
    return []
  }
}

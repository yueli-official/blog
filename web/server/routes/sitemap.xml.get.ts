// XML sitemap for crawlers: home + every published post + taxonomy archive
// pages. Served from the blog domain (Nitro), pulling published content from the
// blog service public list. SEO infra absorbed from the old monolith — a built-in
// feature with sane defaults, not a runtime-config knob.
function esc(s = ''): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}
function lastmod(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '' : d.toISOString().slice(0, 10)
}

export default defineEventHandler(async (event) => {
  const apiBase = useRuntimeConfig(event).apiBase
  const origin = getRequestURL(event).origin
  const [posts, taxes] = await Promise.all([
    $fetch<{ data?: { items?: any[] } }>(`${apiBase}/api/v1/posts`, { query: { page: 1, size: 500 } })
      .then(r => r?.data?.items ?? []).catch(() => []),
    $fetch<{ data?: { items?: any[] } }>(`${apiBase}/api/v1/taxonomies`)
      .then(r => r?.data?.items ?? []).catch(() => [])
  ])

  const urls: string[] = []
  const add = (loc: string, mod?: string) =>
    urls.push(`  <url><loc>${esc(loc)}</loc>${mod ? `<lastmod>${mod}</lastmod>` : ''}</url>`)

  add(`${origin}/`)
  for (const p of posts) add(`${origin}/posts/${p.slug}`, lastmod(p.updatedAt || p.publishedAt || p.createdAt))
  for (const t of taxes) add(`${origin}/c/${t.slug}`)

  setHeader(event, 'content-type', 'application/xml; charset=utf-8')
  return `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls.join('\n')}
</urlset>
`
})

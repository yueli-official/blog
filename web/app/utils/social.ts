export interface SocialPlatform {
  key: string
  label: string
  icon: string
}

export const SOCIAL_PLATFORMS: SocialPlatform[] = [
  { key: 'github', label: 'GitHub', icon: 'i-tabler-brand-github' },
  { key: 'x', label: 'X', icon: 'i-tabler-brand-x' },
  { key: 'weibo', label: '微博', icon: 'i-tabler-brand-weibo' },
  { key: 'zhihu', label: '知乎', icon: 'i-tabler-brand-zhihu' },
  { key: 'bilibili', label: 'Bilibili', icon: 'i-tabler-brand-bilibili' },
  { key: 'youtube', label: 'YouTube', icon: 'i-tabler-brand-youtube' },
  { key: 'telegram', label: 'Telegram', icon: 'i-tabler-brand-telegram' },
  { key: 'linkedin', label: 'LinkedIn', icon: 'i-tabler-brand-linkedin' },
  { key: 'wechat', label: '微信', icon: 'i-tabler-brand-wechat' },
  { key: 'mail', label: '邮箱', icon: 'i-tabler-mail' },
  { key: 'website', label: '网站', icon: 'i-tabler-world' },
]

const byLabel = new Map(SOCIAL_PLATFORMS.map(item => [item.label.toLowerCase(), item]))
const byKey = new Map(SOCIAL_PLATFORMS.map(item => [item.key, item]))
const matchers: Array<[string, string[]]> = [
  ['github', ['github']],
  ['x', ['x.com', 'twitter']],
  ['weibo', ['weibo', '微博']],
  ['zhihu', ['zhihu', '知乎']],
  ['bilibili', ['bilibili', 'b23.tv']],
  ['youtube', ['youtube', 'youtu.be']],
  ['telegram', ['t.me', 'telegram']],
  ['linkedin', ['linkedin']],
  ['wechat', ['wechat', 'weixin', '微信']],
  ['mail', ['mailto', '@']],
]

export function socialPlatform(link: { label?: string, url?: string }): SocialPlatform {
  const exact = byLabel.get((link.label || '').trim().toLowerCase())
  if (exact) return exact
  const candidate = `${link.label || ''} ${link.url || ''}`.toLowerCase()
  for (const [key, values] of matchers) {
    if (values.some(value => candidate.includes(value))) return byKey.get(key)!
  }
  return byKey.get('website')!
}

export function socialIcon(link: { label?: string, url?: string }): string {
  return socialPlatform(link).icon
}

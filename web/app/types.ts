// Outward contracts mirrored from products/blog/api/api/v1.
export interface PostView {
  id: string
  authorId: string
  title: string
  slug: string
  content: string
  excerpt: string
  coverAssetId?: string
  coverUrl?: string
  status: string
  commentStatus: number
  viewCount: number
  pinned: boolean
  featured: boolean
  seriesId?: string
  seriesOrder?: number
  publishedAt?: string
  createdAt: string
  updatedAt: string
  taxonomies?: TaxonomyView[]
}

export interface SeriesView {
  id: string
  slug: string
  name: string
  description?: string
  coverUrl?: string
  authorId: string
  postCount: number
  recentPosts?: PostView[]
}

export interface SeriesDetail {
  series: SeriesView
  posts: PostView[]
}

export interface ListSeries {
  items: SeriesView[]
}

export interface SEOView {
  metaTitle: string
  metaDesc: string
  ogTitle: string
  ogImage: string
  canonicalUrl: string
  robots: string
}

export interface SocialLink {
  label: string
  url: string
}

// AuthorView is the blog-side author profile overlay (M5). displayName/avatarUrl
// may be empty — the UI falls back to the identity id.
export interface AuthorView {
  id: string
  displayName: string
  bio: string
  avatarUrl: string
  bannerUrl: string
  role: string // author | contributor
  status?: string // pending | active | '' (no profile) — on /me/profile
  socialLinks: SocialLink[]
  postCount?: number
  createdAt?: string // profile creation = "joined" date (author page)
}

export interface AuthorPage {
  author: AuthorView
  posts: PostView[]
  total: number
  totalViews: number
  page: number
  size: number
}

// AdminAuthorView is one roster row in the authors page. owner = site owner
// (config), shown read-only; everyone else approved is just 作者.
export interface AdminAuthorView {
  id: string
  displayName: string
  role: string // author | contributor
  status: string // pending | active
  postCount: number
  owner?: boolean // site operator (blog.operatorSubs)
}

export interface AdminAuthorList {
  authors: AdminAuthorView[]
}

// Siblings is a post's prev/next by publish time (article nav, M6).
export interface Siblings {
  prev?: PostView
  next?: PostView
}

export interface ArchiveList {
  items: PostView[]
  total: number
  page: number
  size: number
}

export interface PostDetail {
  post: PostView
  seo?: SEOView
  taxonomies: TaxonomyView[]
  series?: SeriesView
  author?: AuthorView
  liked: boolean
  bookmarked: boolean
}

export interface ListPosts {
  items: PostView[]
  total: number
  page: number
  size: number
}

export interface HomeConfig {
  eyebrow: string
  title: string
  subtitle: string
  siteTitle: string
  siteDescription: string
  supportEmail: string
  footerTagline: string
  footerCopyright: string
}

export interface HomeConfigResponse {
  config: HomeConfig
}

// RelatedPosts is the detail page's "related reading" rail (posts sharing
// taxonomies with the current one).
export interface RelatedPosts {
  items: PostView[]
}

export interface TaxonomyView {
  id: string
  taxonomy: string // category | tag
  name: string
  slug: string
  description?: string
  parentId?: string
  postCount: number
}

export interface ListTaxonomies {
  items: TaxonomyView[]
  total?: number
  page?: number
  size?: number
}

// MyPosts is the author's own posts (any status), distinct from the public
// ListPosts (published only).
export interface MyPosts {
  items: PostView[]
  total: number
  page: number
  size: number
  counts: Record<string, number> // per-status + "all" (filter tabs)
  totalViews: number // sum of view_count across all my posts (dashboard stat)
}

export interface RevisionView {
  id: string
  title: string
  revNote: string
  createdAt: string
}

// CommentView is the public projection of a comment (two-level tree).
export interface CommentView {
  id: string
  parentId?: string
  authorName: string
  isMember: boolean
  content: string
  createdAt: string
  replies?: CommentView[]
}

export interface CommentList {
  items: CommentView[]
  total: number
  page: number
  size: number
}

// CommentAdminView is the moderation projection (status + contact + post).
export interface CommentAdminView {
  id: string
  postId: string
  postTitle?: string
  postSlug?: string
  parentId?: string
  authorName: string
  authorEmail?: string
  userId?: string
  content: string
  status: number
  ip?: string
  createdAt: string
}

export interface MyComments {
  items: CommentAdminView[]
  total: number
  page: number
  size: number
}

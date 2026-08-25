// Outward contracts mirrored from products/blog/api/api/v1.
import type { DiscoveryProjection } from "@yueli/discovery-nuxt/types";
export interface PostView {
  id: string;
  authorId: string;
  title: string;
  slug: string;
  content: string;
  excerpt: string;
  coverAssetId?: string;
  coverUrl?: string;
  status: string;
  commentStatus: number;
  viewCount: number;
  pinned: boolean;
  featured: boolean;
  seriesId?: string;
  seriesOrder?: number;
  publishedAt?: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string;
  taxonomies?: TaxonomyView[];
}

export interface SeriesView {
  id: string;
  slug: string;
  name: string;
  description?: string;
  coverUrl?: string;
  authorId: string;
  postCount: number;
  recentPosts?: PostView[];
}

export interface SeriesDetail {
  series: SeriesView;
  posts: PostView[];
}

export interface ListSeries {
  items: SeriesView[];
}

export interface SEOView {
  metaTitle: string;
  metaDesc: string;
  ogTitle: string;
  ogImage: string;
  canonicalUrl: string;
  robots: string;
}

export interface SocialLink {
  label: string;
  url: string;
}

// AuthorView is the blog-side author profile overlay (M5). displayName/avatarUrl
// may be empty — the UI falls back to the identity id.
export interface AuthorView {
  id: string;
  handle: string;
  displayName: string;
  bio: string;
  avatarUrl: string;
  bannerUrl: string;
  role: string; // author | contributor
  status?: string; // pending | active | '' (no profile) — on /me/profile
  socialLinks: SocialLink[];
  postCount?: number;
  createdAt?: string; // profile creation = "joined" date (author page)
}

export interface AuthorPage {
  author: AuthorView;
  posts: PostView[];
  total: number;
  totalViews: number;
  page: number;
  size: number;
}

// Siblings is a post's prev/next by publish time (article nav, M6).
export interface Siblings {
  prev?: PostView;
  next?: PostView;
}

export interface ArchiveList {
  items: PostView[];
  total: number;
  page: number;
  size: number;
}

export interface PostDetail {
  post: PostView;
  seo?: SEOView;
  discovery?: DiscoveryProjection;
  taxonomies: TaxonomyView[];
  series?: SeriesView;
  author?: AuthorView;
  liked: boolean;
  bookmarked: boolean;
}

export interface URLResolution {
  kind: "canonical" | "alias" | "redirect" | "gone" | "unknown";
  location?: string;
  statusCode?: number;
}

export interface URLResolutionResponse {
  resolution: URLResolution;
}

export interface ListPosts {
  items: PostView[];
  total: number;
  page: number;
  size: number;
}

export interface HomeConfig {
  eyebrow: string;
  title: string;
  subtitle: string;
  siteTitle: string;
  siteDescription: string;
  supportEmail: string;
  footerTagline: string;
  footerCopyright: string;
  friendLinks: FriendLink[];
  contactLinks: ContactLink[];
}

export interface FriendLink {
  label: string;
  url: string;
}

export interface ContactLink {
  value: string;
  url: string;
}

export interface HomeConfigResponse {
  config: HomeConfig;
}

// RelatedPosts is the detail page's "related reading" rail (posts sharing
// taxonomies with the current one).
export interface RelatedPosts {
  items: PostView[];
}

export interface TaxonomyView {
  id: string;
  taxonomy: string; // category | tag
  name: string;
  slug: string;
  description?: string;
  parentId?: string;
  postCount: number;
}

export interface ListTaxonomies {
  items: TaxonomyView[];
  total?: number;
  page?: number;
  size?: number;
}

// MyPosts is the author's own posts (any status), distinct from the public
// ListPosts (published only).
export interface MyPosts {
  items: PostView[];
  total: number;
  page: number;
  size: number;
  counts: Record<string, number>; // per-status + "all" (filter tabs)
  totalViews: number; // sum of view_count across all my posts (dashboard stat)
}

export interface DashboardTrafficPoint {
  day: string;
  views: number;
  uniqueVisitorDays: number;
}

export interface DashboardTopPost {
  id: string;
  title: string;
  slug: string;
  views: number;
  uniqueVisitorDays: number;
}

export interface DashboardTrafficSource {
  source: string;
  views: number;
}

export interface DashboardOverview {
  days: number;
  allTimeViews: number;
  allTimeUniqueVisitorDays: number;
  periodViews: number;
  periodUniqueVisitorDays: number;
  previousPeriodViews: number;
  previousUniqueVisitorDays: number;
  series: DashboardTrafficPoint[];
  topPosts: DashboardTopPost[];
  topSources: DashboardTrafficSource[];
}

export interface RevisionView {
  id: string;
  title: string;
  revNote: string;
  createdAt: string;
}

// CommentView is the public projection of a comment (two-level tree).
export interface CommentView {
  id: string;
  parentId?: string;
  authorName: string;
  avatarUrl?: string;
  isAnonymous: boolean;
  content: string;
  createdAt: string;
  replies?: CommentView[];
}

export interface CommentList {
  items: CommentView[];
  total: number;
  page: number;
  size: number;
}

// CommentAdminView is the moderation projection (status + contact + post).
export interface CommentAdminView {
  id: string;
  postId: string;
  postTitle?: string;
  postSlug?: string;
  parentId?: string;
  authorName: string;
  avatarUrl?: string;
  authorEmail?: string;
  userId?: string;
  content: string;
  status: number;
  ip?: string;
  createdAt: string;
}

export interface MyComments {
  items: CommentAdminView[];
  total: number;
  page: number;
  size: number;
}

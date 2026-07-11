// Social-link icon resolution is shared with the account profile editor — the
// single registry lives in @platform/ui so the editor's preset choices and the
// author-page / byline icons stay in lockstep. Re-exported here so the existing
// auto-imported `socialIcon(link)` call sites (author page, AuthorBox) keep working.
export { socialIcon, socialPlatform, SOCIAL_PLATFORMS } from '@platform/ui/social'

// Re-export the shared date helper so Nuxt auto-imports `rel`/`abs` while the
// implementation lives in @platform/ui (shared across all consumer-site apps).
// See frontend-design-system §13.
export { rel, abs } from '@platform/ui/date'

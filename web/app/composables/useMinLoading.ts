// Re-export the shared loading-debounce composable so Nuxt auto-imports it while
// the implementation lives in @platform/ui (shared across consumer-site apps).
export { useMinLoading } from '@platform/ui/use-min-loading'

import { blogTheme } from '~/config/theme'

// Apply the blog's design tokens on the client. Idempotent with the static CSS
// defaults (@platform/ui/theme.css --ui-radius / --font-sans + app.config's
// primary), so there's no flash — this just makes config/theme.ts the live
// source and is the hook an appearance admin would feed persisted overrides into.
export default defineNuxtPlugin(() => {
  useDesignTokens().applyAll(blogTheme)
})

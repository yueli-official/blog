import type { DesignTokens } from '@platform/ui/tokens'
import { DEFAULT_DESIGN_TOKENS } from '@platform/ui/tokens'

// The blog's design tokens — the single source for this site's look. Re-skin by
// editing here. The static CSS defaults (@platform/ui/theme.css --ui-radius /
// --font-sans + app.config's primary) keep SSR flash-free; theme.client.ts
// re-applies these on the client (idempotent). When a site wants an appearance
// admin, it feeds persisted overrides into useDesignTokens().applyAll().
export const blogTheme: DesignTokens = {
  ...DEFAULT_DESIGN_TOKENS,
  primary: 'blue',    // mirrors app.config.ts platformAppConfig('blog')
  neutral: 'stone',
  radius: '0.5rem',   // mirrors @platform/ui --ui-radius default
  font: 'dm-sans',
  fontSize: 16,
  colorMode: 'system'
}

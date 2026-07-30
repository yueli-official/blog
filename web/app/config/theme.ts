export interface DesignTokens {
  primary: string
  neutral: string
  radius: string
  font: string
  fontSize: number
  colorMode: 'light' | 'dark' | 'system'
}

export const DEFAULT_DESIGN_TOKENS: DesignTokens = {
  primary: 'blue',
  neutral: 'stone',
  radius: '0.5rem',
  font: 'dm-sans',
  fontSize: 16,
  colorMode: 'system',
}

const FONT_STACKS: Record<string, string> = {
  'dm-sans': '"DM Sans", system-ui, sans-serif',
  system: 'system-ui, -apple-system, "Segoe UI", sans-serif',
  serif: '"Noto Serif SC", "Source Han Serif", Georgia, serif',
  mono: '"JetBrains Mono", "Fira Code", Consolas, monospace',
}

export function fontStack(value: string): string {
  return FONT_STACKS[value] || FONT_STACKS['dm-sans']!
}

// The blog's design tokens — the single source for this site's look. Re-skin by
// editing here. The static CSS defaults (@yueli/ui/theme.css --ui-radius /
// --font-sans + app.config's primary) keep SSR flash-free; theme.client.ts
// re-applies these on the client (idempotent). When a site wants an appearance
// admin, it feeds persisted overrides into useDesignTokens().applyAll().
export const blogTheme: DesignTokens = {
  ...DEFAULT_DESIGN_TOKENS,
  primary: 'blue',
  neutral: 'stone',
  radius: '0.5rem',
  font: 'dm-sans',
  fontSize: 16,
  colorMode: 'system'
}

import type { DesignTokens } from '@platform/ui/tokens'
import { fontStack, DEFAULT_DESIGN_TOKENS } from '@platform/ui/tokens'

// useDesignTokens — applies a DesignTokens set at runtime. The reusable token
// technique (borrowed from the legacy admin theme system):
//   radius → --ui-radius     (the single knob the whole rounded-* scale follows)
//   font   → --font-sans + <html> font-size
//   color  → app.config ui.colors  (NuxtUI regenerates its palette vars)
//   mode   → useColorMode().preference
// Copy this composable into any site that wants runtime theming; the option
// lists + shape live in @platform/ui/tokens.
export function useDesignTokens() {
  const appConfig = useAppConfig()
  const colorMode = useColorMode()

  const applyPrimary = (c: string) => { (appConfig.ui as any).colors.primary = c }
  const applyNeutral = (c: string) => { (appConfig.ui as any).colors.neutral = c }
  const applyRadius = (r: string) => {
    if (import.meta.client) document.documentElement.style.setProperty('--ui-radius', r)
  }
  const applyColorMode = (m: string) => { colorMode.preference = m }
  const applyFont = (font: string, size: number) => {
    if (!import.meta.client) return
    document.documentElement.style.setProperty('--font-sans', fontStack(font))
    document.documentElement.style.fontSize = `${size}px`
  }
  const applyAll = (t: Partial<DesignTokens>) => {
    const s = { ...DEFAULT_DESIGN_TOKENS, ...t }
    applyPrimary(s.primary)
    applyNeutral(s.neutral)
    applyRadius(s.radius)
    applyColorMode(s.colorMode)
    applyFont(s.font, s.fontSize)
  }

  return { applyPrimary, applyNeutral, applyRadius, applyColorMode, applyFont, applyAll }
}

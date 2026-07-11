// Single source of truth for page container widths. A page declares its tier via
//   definePageMeta({ width: 'wide' })
// and the default layout applies the matching max-width. No tier → 'narrow'.
export const PAGE_WIDTHS = {
  narrow: 'max-w-3xl', // reading column: article / search / forms (default)
  wide: 'max-w-5xl',   // author console content (the `manage` layout)
  full: 'max-w-6xl',   // front-of-site browse: home + category / tag / series
} as const

export type PageWidth = keyof typeof PAGE_WIDTHS

declare module '#app' {
  interface PageMeta {
    width?: PageWidth
  }
}

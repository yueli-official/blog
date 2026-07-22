// Nuxt 4 config for the blog-site app (content consumer site).
// Extends @platform/auth (the OIDC BFF layer): /auth/* + the /api/v1 proxy that
// injects the user's Bearer token come from the layer, so this app has no devProxy.
const siteBrand = process.env.NUXT_PUBLIC_SITE_BRAND || "博客";

export default defineNuxtConfig({
  extends: [
    "@platform/auth",
    "@platform/site",
    "@platform/manage",
    "@platform/asset",
    "@platform/content",
  ],
  modules: ["@nuxt/ui", "@yueli/ui"],
  // The rich editor's tiptap/prosemirror dedup (vite.optimizeDeps) and the global
  // katex stylesheet now come from the @platform/content layer's nuxt.config, which
  // Nuxt merges into this consumer (verified: layer optimizeDeps lands in the dev
  // dep-optimize metadata). So this app declares neither — it only keeps its own
  // site stylesheet here.
  css: ["~/assets/css/main.css"],
  app: {
    head: {
      // RSS autodiscovery (M3): browsers / readers find the site-wide feed.
      link: [
        {
          rel: "alternate",
          type: "application/rss+xml",
          title: `${siteBrand} RSS`,
          href: "/rss.xml",
        },
      ],
      // Site-wide SEO defaults (per-page useSeoMeta overrides these).
      meta: [
        { property: "og:site_name", content: siteBrand },
        { property: "og:type", content: "website" },
        { name: "twitter:card", content: "summary_large_image" },
      ],
    },
  },
  // Multiple site instances can run from this source tree at once. Each devkit
  // launch gets an instance-specific NUXT_BUILD_DIR so generated Nuxt state is
  // never shared across processes.
  buildDir: process.env.NUXT_BUILD_DIR || ".nuxt",
  devServer: { port: Number(process.env.NUXT_DEV_PORT || "3002") },
  // Offline-resilient: @nuxt/ui pulls in @nuxt/fonts which, by default, probes
  // fonts.google.com (+ google material icons) and retries on failure — noisy and
  // slow on an offline machine. Disable the network providers; fonts resolve from
  // local/system. (Commercial polish: self-host a brand font via @fontsource.)
  runtimeConfig: {
    // SSR-only base for direct (anonymous) public-page reads to the blog service.
    // Authenticated client calls instead go through /api/v1 (the BFF proxy from
    // @platform/auth, which injects the Bearer token).
    apiBase: "http://127.0.0.1:8085",
    // OIDC BFF (layer) config — override in prod with NUXT_* env.
    sealSecret: "dev-blog-seal-secret-change-me-0123456789abcd", // NUXT_SEAL_SECRET
    downstreamBase: "http://127.0.0.1:8085", // NUXT_DOWNSTREAM_BASE — the blog service
    public: {
      siteSlug: "blog-local",
      siteBrand: "博客",
      siteDomain: "localhost:3002",
      assetSpace: "local",
      assetNamespace: "local",
      assetProfile: "blog-default",
      oidcIssuer: "http://localhost:8081",
      oidcClientId: "blog-web",
      oidcRedirectUri: "http://localhost:3002/auth/callback",
      oidcScopes: "openid profile email roles offline_access",
      // User center (account app) — profile editing lives there now. NUXT_PUBLIC_ACCOUNT_URL.
      accountUrl: "http://localhost:3000",
    },
  },
  devtools: { enabled: true },
});

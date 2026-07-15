# Blog product

- Lifecycle: active reusable product type
- Authority: Catalog product type `blog`, `api/` migrations/OpenAPI, `web/` UI
- Consumers: Blog site instances such as `blog-ai` and `blog-ui`
- Verify: `pnpm platformctl verify product --file catalog/overlays/local.yaml --root . blog`

Blog owns posts, publication, subscriptions, comments and its product settings.
It consumes shared classification/content, Identity, Asset and Notification
contracts. `api/` is the domain/API module; `web/` is the public and management
Nuxt app. Site-specific values belong in Catalog, not this source tree.

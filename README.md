# 博客产品

- 生命周期：活跃的可复用产品类型
- 权威来源：Catalog 产品类型 `blog`、`api/` 迁移/OpenAPI、`web/` 界面
- 消费者：`blog-ai`、`blog-ui` 等博客站点实例
- 验证：`pnpm platformctl verify product --file catalog/overlays/local.yaml --root . blog`

Blog 负责文章、发布、订阅、评论及产品设置，并消费共享分类/内容、Identity、Asset 和 Notification 契约。`api/` 是领域与接口模块，`web/` 是公开站与管理 Nuxt 应用。站点专属值必须写入 Catalog。

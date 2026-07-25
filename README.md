# 博客产品

- 生命周期：活跃的可复用产品类型
- 权威来源：Catalog 产品类型 `blog`、`api/` 迁移/OpenAPI、`web/` 界面
- 消费者：`blog-ai`、`blog-ui` 等博客站点实例
- 验证：`pnpm platformctl verify product --file catalog/overlays/local.yaml --root . blog`

Blog 负责文章、发布、订阅、评论及产品设置，并消费共享分类/内容、Identity、Asset、Notification 和 Foundation Traffic 契约。`api/` 是领域与接口模块，`web/` 是公开站与管理 Nuxt 应用。访问量真值位于实例本地 Traffic 表；`post_stats.view_count` 仅是列表与作者仪表盘的可重建查询投影。站点专属值必须写入 Catalog。

## 本站权限

Blog 的角色、授权、申请和自动规则全部存放在站点实例自己的数据库中，不继承 Identity 或其他站点的角色：

- `administrator`（管理员）是受保护角色，可管理全站内容、设置、角色策略和作者申请。
- `author`（作者）可以创建、编辑、发布、归档和删除自己的文章，管理自己文章下的评论，并维护自己的系列。
- 未登录访客只有公开读取能力；普通登录用户可以申请允许申请的角色。
- “注册用户自动成为作者”默认关闭。管理员在权限策略草稿中启用并发布后，用户首次进入本站时会幂等补齐自动授权。
- 管理员可以创建自定义角色、组合 Blog 能力、直接授予或撤销角色，并通过草稿验证和影响预览后发布。

内容资源使用 `Site → Post → Comment` 与 `Site → Series` Scope；作者权限由 `owner` relation 约束，管理员使用受保护能力。`author_profiles` 已退出运行时并由迁移删除：公开作者资料来自 Identity，内容归属来自 Blog 内容表，权限真相只来自 Foundation Authorization。

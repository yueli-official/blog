# 月离博客

Blog 是独立的文章发布消费者产品，拥有文章、系列、分类、标签、评论、订阅、站点设置、权限和管理界面。
`api/` 与 `web/` 是本仓唯一实现真源；仓库不依赖 Platform 源码或工作区私有包。

## 边界

- Blog 自己拥有领域数据、PostgreSQL migration、授权实例、Discovery 发布、流量投影和界面组件。
- Identity 只通过 OIDC issuer、Discovery、JWKS 与公开资料 HTTP 契约提供身份。
- Asset 只通过公开 HTTP 契约管理封面和正文图片；Blog 不导入 Asset 内部代码。
- Foundation 通过正式 Go module 与 JS Release 提供跨产品协议原语。
- 本地多仓编排属于相邻 `workspace`；生产部署属于本仓 Compose。

不可变依赖与能力绑定记录在：

- `deploy/contracts/requirements.json`：消费者需要的能力；
- `deploy/deployment.lock.json`：能力到具体生产者版本的部署锁。

## 本地开发

推荐从相邻 `workspace` 仓启动：

```powershell
# Identity + Account + Blog 专属 Asset + Blog
.\environments\blog-local\run.ps1 -Mode Complete

# 复用已有 Identity，管理 Blog 专属 Asset
.\environments\blog-local\run.ps1 -Mode Hybrid

# 复用已有 Identity 与 Asset，只启动 Blog
.\environments\blog-local\run.ps1 -Mode Attach

.\environments\blog-local\run.ps1 -Action Down
```

所有端口均可通过 `LOCAL_*_PORT` 覆盖；关闭 Blog target 只停止该 Workspace 会话，不会终止其他项目。

## Docker Compose

本仓提供三种生产/预发布拓扑：

```powershell
# 完整独立部署：Identity + Account + Blog 专属 Asset + Blog
Copy-Item .env.example .env
docker compose -f compose.yaml up -d --wait

# 复用已有 Identity，部署 Blog 专属 Asset
Copy-Item deploy/env/hybrid.env.example .env
docker compose -f compose.hybrid.yaml up -d --wait

# 复用已有 Identity 与 Asset
Copy-Item deploy/env/attach.env.example .env
docker compose -f compose.attach.yaml up -d --wait
```

生成 `.env` 后必须填写所有空 secret、管理员 Subject 和外部服务 URL。宿主端口由
`BLOG_API_PORT`、`BLOG_WEB_PORT`、`IDENTITY_PORT`、`IDENTITY_ACCOUNT_PORT`、`ASSET_PORT`
配置，不硬编码占用。

Blog PostgreSQL 使用锁定的 `postgres-zhparser` 镜像，因为现有搜索 migration 明确依赖 `zhparser`；
普通 PostgreSQL 不能替代该契约。邮件默认使用离线安全的 `dev` 输出，生产环境通过
`BLOG_MAILER_MODE=smtp` 与 `BLOG_SMTP_*` 显式启用 SMTP。

## 独立命令

开发者令牌的权限选择、分类层级、媒体上传与完整投稿调用顺序见[开发者令牌 API](docs/developer-tokens.md)。

```powershell
cd api
go run ./cmd/blog
go run ./cmd/errorcatalog

cd ..\web
pnpm install --frozen-lockfile --ignore-workspace
pnpm dev
```

运行配置模板位于 `api/manifest/config/config.example.yaml`。本仓不使用 `doctor.yaml`。

## 验收策略

API、Web、Compose 与浏览器合同均由本仓 CI 拥有。当前迁移批次按约定暂停逐产品测试；完成所有消费者迁移后，
再统一运行 API、前端、容器、Compose 和 Playwright 验收。

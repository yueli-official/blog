# 开发者令牌 API

## 1. 接入信息

| 项目 | 值 |
| --- | --- |
| 产品 | Blog / CG |
| 生产 Origin | `https://cg.yuelili.com` |
| 令牌站点 ID | `blog-main-web` |
| 认证 | `Authorization: Bearer <PAT>` |
| 字段与 DTO | [OpenAPI](../contracts/openapi/blog.json) |

站群统一入口、任务交接与执行规则见 [Workspace 开发者令牌 API](../../workspace/docs/developer-tokens.md)。

Blog 接受账户中心创建的个人访问令牌（PAT）。请求发送到 Blog 的公开 Origin，使用 `Authorization: Bearer <token>`；不需要浏览器会话，也不能用 PAT 换取浏览器登录。

## 2. 权限范围

在账户中心的“开发者令牌”页面选择当前 Blog 实例。目录只显示当前账号实际拥有的权限；每次调用还会重新验证令牌及产品权限。新增权限不会自动加入旧令牌，需创建新令牌并勾选所需操作。

普通投稿建议选择：上传文章图片、读取文章、创建文章、编辑文章、发布文章、创建标签。需要创建分类和调整分类层级时，账号必须拥有本站分类管理权限，并选择“管理分类与标签”。需要自动组织系列时，选择创建系列和编辑系列。

| 令牌操作 | 权限键 | 用途 |
| --- | --- | --- |
| 上传文章图片 | `media.upload` | 正文图片；封面还需要 `blog.post.update` |
| 读取文章 | `blog.post.read` | 读取当前账号有权管理的文章，包括草稿和回收站筛选 |
| 创建文章 | `blog.post.create` | 以本人身份创建草稿 |
| 编辑文章 | `blog.post.update` | 标题、Markdown 正文、摘要、Slug、发布时间、分类标签、系列顺序及封面 |
| 发布文章 | `blog.post.publish` | 将文章状态改为 `published` |
| 下架文章 | `blog.post.archive` | 将文章状态改为 `draft` 或 `archived` |
| 回收与恢复文章 | `blog.post.delete` | 移入回收站、恢复；不开放永久删除 |
| 创建标签 | `blog.tag.create` | 新建标签 |
| 管理分类与标签 | `blog.taxonomy.manage` | 新建分类、修改层级或名称、合并、删除分类和标签 |
| 创建系列 | `blog.series.create` | 以本人身份新建系列 |
| 编辑系列 | `blog.series.update` | 修改有权编辑的系列名称、Slug 和说明 |
| 删除系列 | `blog.series.delete` | 删除系列，文章保留并解除归属 |
| 设置文章推荐与置顶 | `blog.post_flags.manage` | 管理员内容编排 |

分类管理和推荐置顶仍受账号当前管理权限限制；普通作者不能通过自填 scope 获得这些权限。编辑、发布等也不突破文章或系列的资源权限。令牌不开放用户授权治理、站点设置、永久删除和管理后台批处理接口。

## 3. 投稿流程

所有路径相对 Blog Origin。成功返回业务 DTO；失败按 HTTP status 与 Problem 的 `code` 判断，不解析错误文案。

1. **查询已有分类、标签和系列**：`GET /api/v1/taxonomies?taxonomy=category`、`GET /api/v1/taxonomies?taxonomy=tag`、`GET /api/v1/series`。分类标签列表分页，应遍历完再决定是否新建。
2. **按父子关系新建分类**：`POST /api/v1/taxonomies`，请求体例如 `{"taxonomy":"category","name":"Ae","slug":"ae","parentId":"父分类ID"}`。顶层省略 `parentId`。响应 `201`，读取 `taxonomy.id`。新建标签使用 `taxonomy: "tag"`，并选择创建标签权限。
3. **创建文章草稿**：`POST /api/v1/posts`，请求体为 `{"title":"文章标题","content":"Markdown 正文","excerpt":"摘要"}`。响应 `201`，保存 `post.id`、`post.slug`。本地文件的 YAML frontmatter 应先提取为元数据，不直接混入正文。
4. **上传正文图片**：`POST /api/v1/images`，传 `filename`、`mime`、`size`；根据响应的 `uploadUrl`、`uploadHeaders` 上传原始字节，再 `POST /api/v1/images/finalize`，传 `uploadToken`。将返回的 `url` 写入 Markdown。
5. **上传封面**：同一流程使用 `POST /api/v1/posts/{id}/cover` 和 `POST /api/v1/posts/{id}/cover/finalize`，完成后返回 `coverAssetId`、`coverUrl`。需要上传图片和编辑文章两项权限，且账号必须有权编辑该文章。
6. **保存正文与路径**：`PATCH /api/v1/posts/{id}`，传最终 `content`、`slug`、`excerpt` 等字段。`publishedAt` 使用 RFC 3339 格式；这是发布元数据，不是定时任务调度。
7. **设置分类与标签**：`PUT /api/v1/posts/{id}/taxonomies`，传 `{"taxonomyIds":["分类ID","标签ID"]}`，成功 `204`。这是完整替换列表，不是追加；空数组解除归类。
8. **组织系列**：`POST /api/v1/series` 新建系列，然后 `PUT /api/v1/posts/{id}/series`，传 `{"seriesId":"系列ID","seriesOrder":1}`，成功 `204`。空 `seriesId` 解除归属；不必拥有新建系列权限才能选择已有系列。
9. **发布**：`PATCH /api/v1/posts/{id}`，传 `{"status":"published"}`。如果同一请求修改正文，还需编辑文章权限。可以先保留草稿，最后逐篇发布。
10. **核对结果**：`GET /api/v1/posts/mine` 读取管理列表，公开文章通过 `GET /api/v1/posts/{slug}` 验证，浏览器入口为 `/posts/{slug}`。当前公开详情的草稿预览仍使用浏览器/JWT；PAT 读取草稿使用管理列表。

媒体只使用返回的上传地址、请求头、上传令牌和交付 URL，不推导对象 Key 或存储后端。分类目录与磁盘目录的映射由导入客户端记录；遇到请求超时或返回不确定，应先查询确认，不能盲目重试创建。保存文章 ID 后可持续更新，避免重复投稿。

## 4. 接口索引

- `PATCH /api/v1/taxonomies/{id}` 修改分类或标签；分类可用 `parentId` 调整层级。
- `POST /api/v1/taxonomies/{id}/merge`，传 `targetId` 合并同类条目。合并源保留替代关系，删除测试数据时先删合并源再删目标。
- `DELETE /api/v1/taxonomies/{id}` 删除分类或标签；分类仍有子级时必须先处理子级。
- `PATCH /api/v1/series/{id}`、`DELETE /api/v1/series/{id}` 编辑或删除系列。
- `PUT /api/v1/posts/{id}/flags` 设置 `pinned`、`featured`。
- `DELETE /api/v1/posts/{id}` 移入回收站；`POST /api/v1/posts/{id}/restore` 恢复。

多个文件可由导入客户端循环调用以上已授权单项接口，不使用 `/api/v1/posts/batch`。

## 5. 部署与验证

现有生产服务已于 2026-09-12 部署包含 PAT 的固定源码候选，详见[部署记录](../flightdeck/work/2026-09-12-personal-token-publishing/deployment-20260912.md)。生产可用状态不等于正式 SDK Release 已发布；新建环境仍须核对 Provider、共享依赖和站点注册，不能直接把旧依赖版本当成已支持 PAT。

Blog 的令牌验证、Identity 的可授权操作目录、Asset 的媒体授权回查需要同时配置。`siteId`、目录应用 ID、目录 audience 与 Blog 的 OIDC client ID 必须一致。下面的地址由部署方声明，不由上传客户端拼接。

| 运行组件 | Compose 环境变量 | 配置用途 |
| --- | --- | --- |
| Blog | `BLOG_PAT_SITE_ID` | 启用本站 PAT 验证，值等于 `BLOG_OIDC_CLIENT_ID`；留空关闭 |
| Blog | `BLOG_PAT_ALLOW_HTTP` | Identity 内部地址使用 HTTP 时，由部署方显式开启 |
| Identity | `IDENTITY_PAT_APPLICATIONS` | JSON 应用列表，登记本站 ID、名称、权限目录 URL 与 audience |
| Identity | `IDENTITY_PAT_ALLOW_HTTP` | 权限目录位于受信任私有 HTTP 网络时开启 |
| Asset | `ASSET_PAT_AUTHORITIES` | JSON 对象，将本站 ID 映射到 Blog 的媒体授权 URL |
| Asset | `ASSET_PAT_ALLOW_HTTP` | 媒体授权回查位于受信任私有 HTTP 网络时开启 |

完整独立部署的示例已在 [`.env.example`](../.env.example) 中提供，私有地址使用 `identity:8081` 与 `blog-api:8085`。Hybrid 模式只管理 Asset，需在外部 Identity 登记 Blog 权限目录；Attach 模式需在既有 Identity 和 Asset 分别登记。两种模式的示例位于 `deploy/env/`。共享 Provider 的登记列表应保留其他站点条目。

正文/封面上传权限与已发布资产引用同步是两条流程。`BLOG_ASSET_TOKEN_URL`、`BLOG_ASSET_CLIENT_ID`、`BLOG_ASSET_CLIENT_SECRET` 仍需配置为本站机器客户端，不能拿个人令牌代替。

### 本地验收

复用 Workspace 独立 Blog 组合及其显式本地 Go 依赖覆盖。在 `web` 目录运行：

```powershell
node test/e2e/personal-token-publishing.mjs
```

脚本限定开发域名，在真实账户中心创建临时令牌，通过 CLI Playwright 检查桌面和手机权限选择、文章渲染，并验证上传、分类、系列、发布和越权拒绝。结束时撤销本次令牌和临时角色、删除测试分类系列、将测试文章移入回收站。不会修改现有用户令牌或生产内容。

## 6. 参考资料

- [站群开发者令牌 API](../../workspace/docs/developer-tokens.md)
- [产品 OpenAPI](../contracts/openapi/blog.json)
- [当前生产部署表](../../workspace/docs/deployments.md)
- [本次部署与验证边界](../flightdeck/work/2026-09-12-personal-token-publishing/deployment-20260912.md)

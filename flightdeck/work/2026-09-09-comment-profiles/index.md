# 评论用户资料与草稿预览

2026-09-11 已同步正式部署，见[本轮记录](deployment-20260911.md)。此前“仅本地/未部署”描述是历史阶段。


## Status

Finished

## Findings and changes

- Identity 批量接口正式字段为 `items`，Go 客户端和旧测试错误使用 `users`。已修复并覆盖批量昵称、头像字段；单用户 `user` 合同不变。
- 用户报告的 `未命名文章-3/comments` 404 已复现。生产只读查询确认文章为 draft，comment_status=1；正文允许作者预览，评论接口只允许 published。页面改为仅 published 挂载共享 PublicCommentThread，不放宽公开 API。
- 本地真实 Playwright 创建草稿：正文正常、没有评论请求；发布后刷新：评论接口 200、编辑框可见。脚本与截图 `E:/tmp/yueli-docs-publish-20260909/blog-comments-ui.mjs`。创建的文章已清理。
- Go identityclient 测试、本地 Nuxt typecheck 已通过。

## Next

无。

## Deployment and verification

- 2026-09-09 后续授权角色标签修复：已接入 AuthorizationGrantBadge，去掉重复 sourceLabel。初始认领的标签显示管理员，hover/focus 显示“首次管理员认领”。真实本地双宽度验收通过。Web 最新 `server-20260909-grants-1`，保留已验证评论分栏。证据 `E:/tmp/yueli-sites-sync-20260909/grants` 与 `E:/tmp/yueli-docs-publish-20260909/grants-blog-*`；生产管理员会话仍不可用。

- 用户确认 Gallery 后，Blog 评论页启用共享 columns 布局并完整显示已通过状态；Toolbar 贴齐标题右侧。真实本地创建文章/评论验证菜单、状态、排序、390/1440 对齐，无 pageerror，验收文章已删除。
- 线上 Web 后续更新为 `server-20260909-columns-1`，API 保持 comments-1。独立固定 UI 候选、frozen 安装、typecheck/build 通过；线上评论 JS SHA256 与构建一致，公开页通过双宽度 Playwright，生产管理员会话不可用。
- 证据 `E:/tmp/yueli-sites-sync-20260909/columns`、`E:/tmp/yueli-docs-publish-20260909/columns-consumers-blog-docs.json`；Compose 备份 `/projects/yuelili.com/backups/comment-columns-20260909`。

- 独立 sync-1 源码快照定向 Go 测试、Linux API 构建、Nuxt typecheck 和正式 build 通过，保留既有私有依赖快照。
- `blog-api` 和 `blog-web` 已部署 `server-20260909-comments-1`，容器健康；仅客户端批量解码和文章评论挂载条件变化。Compose 备份 `/projects/yuelili.com/backups/comments-20260909`。
- 生产匿名草稿评论保持 404。当前生产公开文章列表为空，未写入测试文章；已发布评论 200 和草稿预览无请求由本地真实创建/发布测试证明。生产没有作者/管理员会话，未冒称该预览线上交互已验证。
- 产物 SHA256、构建日志和 deployed.json 在 `E:/tmp/yueli-sites-sync-20260909/comments-fix`。未提交 Git、未发布依赖包。

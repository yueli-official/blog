# Blog 基础升级与站点复验

## Goal

让 Blog API 与 Web 完整采用 Foundation 的声明式错误目录、直接成功 DTO 和统一失败反馈，并通过本地 Identity/Asset 组合完成真实浏览器验收；基础升级后由用户重新逐页验收，发现的问题在本产品 Work 内处理。

## Status

Open

## Current

Blog 已完成统一错误与 HTTP Result 合同：16 个业务错误由 Foundation v1 catalog 单向生成，68 个 operation 的成功结果与业务 errors 已声明并受 CI 门禁保护；公开参数使用稳定命名类型，前端只按 code/params 解析安全反馈。静态、构建和真实浏览器验收全部通过。

2026-09-07：已完成 Blog 真机候选部署，入口 https://cg.yuelili.com ，登录依赖 account.yuelili.com，媒体依赖 asset.yuelili.com。两组独立 PG 编排与三个产品分库分账号已就绪；HTTPS、登录、后台三次 SSR 刷新、编辑器保存、真实媒体上传/交付、浅深色响应式及主要页面 Axe 已验证。修复正常空库初始化遗漏分类目录/策略、Web 健康检查指向不存在路由的问题。固定候选制品不等于正式 Release；1Panel API 未开启，编排由 SSH 启动，站点代理与证书沿用 1Panel。加密初始备份已复制到本机。

管理员初始化已纠正：仅按用户授权重建 Blog 试验库，保留用户自注册的 Identity 账户并停用部署账号；后续用户已自行认领，不重置或重新开放入口。Identity 的邮箱验证 UUIDv7 缺失已修复。SMTP、平台配置和 Commerce 候选部署已交付，真实用户链路仍待验收。

2026-09-08：权限页修复用户资料响应字段，补齐昵称、头像、主页、申请时间及授权生效时间；API/Web 已部署 `server-20260908-users-1`，无数据库迁移。详见[修复与验收](references/authorization-users.md)。

## Next

用户查看 Blog 在线版文章与评论新布局，反馈归本 Work；见[后台布局与部署证据](references/admin-layout.md)。其他站点等待用户确认，不重复认领或重建数据库。

## Progress

- catalog、生成物、OpenAPI/operation freshness、68/68 覆盖和 CI compatibility diff 已落地。
- Go test/race/vet/govulncheck 与 Web tests/typecheck/build 已通过；同步修复 goldmark 与 gRPC 可达漏洞。
- 用户确认本地数据库均为可丢弃测试数据后，Workspace 已重建 Blog/Asset 数据库并消除历史媒体漂移；首位管理员认领、公开首页/阅读、管理入口、编辑器及真实 201/204 空正文 Playwright 全绿。
- 60 个可能产生产品业务错误的 operation 已声明 errors，16 个 catalog code 均有消费者且无失效路由；真实浏览器验证 400 Problem 不含 raw message，并携带稳定 reason。

## References

- [稳定上下文](context.md)
- [Foundation HTTP Result Contract](../../../../foundation/flightdeck/knowledge/errors/http-result-contract.md)
- [Foundation Error Catalog](../../../../foundation/flightdeck/knowledge/errors/error-catalog.md)

## 本地域名迁移（2026-09-08）
用户授权所有本地开发改用独立域名。Blog 浏览器 Origin 为 http://blog.dev.yuelili.test:3002，共享 Account 为 http://account.dev.yuelili.test:3000。原 URL lifecycle catalog 含旧 IP Origin，切换引发运行定义校验失败；已通过产品 Definition 分别编译旧/新 Origin，事务锁定并严格核对旧版本/digest，只迁移 blog:blog-main 的 Origin 定义及 revision。未修改 SQL migration/checksum 或 URL 历史、业务内容。新 Origin NewPostgres 已接受。
一次性工具 api/.data/local-origin-migration/main.go，迁移前记录 E:/tmp/yueli-media-preset-20260908/blog-origin-before.json。Workspace 已修复 Go overlay 误扫描 .data 旧部署快照的问题并通过回归检查。

本轮最终运行 Session 20260907T232700Z-13888；CLI Playwright 登录、中央 SSO 恢复、20 其他主机 Cookie 隔离通过，本站 Cookie 最大 1741 字节。证据 E:/tmp/yueli-media-preset-20260908/blog-domain-sso-report.json。

## 2026-09-08 Blog 后台紧凑布局

用户指定先部署 Blog 在线版观察效果：共享分页、紧凑网格、评论布局、标题右侧搜索工具。文章/评论/系列已接入并通过本地验收；线上 Web 已更新为 server-20260908-admin-1 且 healthy。详细验证边界见[后台布局](references/admin-layout.md)。只更新本站 Web，既有管理员归属与数据库不变。

分类/标签与分页补齐已部署 server-20260908-admin-2，容器 healthy；线上系列页真实浏览器确认首页末页按钮和数量/页。

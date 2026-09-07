# Blog 基础升级与站点复验

## Goal

让 Blog API 与 Web 完整采用 Foundation 的声明式错误目录、直接成功 DTO 和统一失败反馈，并通过本地 Identity/Asset 组合完成真实浏览器验收；基础升级后由用户重新逐页验收，发现的问题在本产品 Work 内处理。

## Status

Open

## Current

Blog 已完成统一错误与 HTTP Result 合同：16 个业务错误由 Foundation v1 catalog 单向生成，68 个 operation 的成功结果与业务 errors 已声明并受 CI 门禁保护；公开参数使用稳定命名类型，前端只按 code/params 解析安全反馈。静态、构建和真实浏览器验收全部通过。

2026-09-07：已完成 Blog 真机候选部署，入口 https://cg.yuelili.com ，登录依赖 account.yuelili.com，媒体依赖 asset.yuelili.com。两组独立 PG 编排与三个产品分库分账号已就绪；HTTPS、登录、后台三次 SSR 刷新、编辑器保存、真实媒体上传/交付、浅深色响应式及主要页面 Axe 已验证。修复正常空库初始化遗漏分类目录/策略、Web 健康检查指向不存在路由的问题。固定候选制品不等于正式 Release；1Panel API 未开启，编排由 SSH 启动，站点代理与证书沿用 1Panel。加密初始备份已复制到本机。

管理员初始化已纠正：仅按用户授权重建 Blog 试验库，保留用户自注册的 Identity 账户并停用部署账号；后续用户已自行认领，不重置或重新开放入口。Identity 的邮箱验证 UUIDv7 缺失已修复。SMTP、平台配置和 Commerce 候选部署已交付，真实用户链路仍待验收。

## Next

先完成 [Account 真实登录与邮件链路验收](../../../../identity/flightdeck/work/2026-09-07-provider-administration/references/next-acceptance.md)，再回本 Work 继续 Blog 逐页目检。恢复时读 [服务器部署](references/server-deployment.md)，核对现有容器，不重复安装、认领或重建数据库。正式 Release 仍由基础发布 Work 承接。

## Progress

- catalog、生成物、OpenAPI/operation freshness、68/68 覆盖和 CI compatibility diff 已落地。
- Go test/race/vet/govulncheck 与 Web tests/typecheck/build 已通过；同步修复 goldmark 与 gRPC 可达漏洞。
- 用户确认本地数据库均为可丢弃测试数据后，Workspace 已重建 Blog/Asset 数据库并消除历史媒体漂移；首位管理员认领、公开首页/阅读、管理入口、编辑器及真实 201/204 空正文 Playwright 全绿。
- 60 个可能产生产品业务错误的 operation 已声明 errors，16 个 catalog code 均有消费者且无失效路由；真实浏览器验证 400 Problem 不含 raw message，并携带稳定 reason。

## References

- [稳定上下文](context.md)
- [Foundation HTTP Result Contract](../../../../foundation/flightdeck/knowledge/errors/http-result-contract.md)
- [Foundation Error Catalog](../../../../foundation/flightdeck/knowledge/errors/error-catalog.md)

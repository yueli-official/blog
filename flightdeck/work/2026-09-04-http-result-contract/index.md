# 统一错误与 HTTP Result 合同

## Goal

让 Blog API 与 Web 完整采用 Foundation 的声明式错误目录、直接成功 DTO 和统一失败反馈，并通过本地 Identity/Asset 组合完成真实浏览器验收。

## Status

Open

## Current

Blog 已升级 Foundation Go v0.4.1 与 JS v0.7.2；16 个业务错误由 Foundation v1 catalog 单向生成 Go/TypeScript/i18n，旧目录仅作投影。68 个 OpenAPI operation 已纳入 manifest 与覆盖/兼容门禁，10 个创建结果使用 201，8 个无正文结果使用 204；前端页面不再读取 raw message。本地 Blog/Asset 数据库已重建，首位管理员已通过真实浏览器认领，公开阅读、管理、编辑器和状态合同 Playwright 均通过。

## Next

依据 [执行计划](plan.md) 补齐 operation 的业务 errors 声明和 typed cause 映射审计；不发布版本。

## Progress

- catalog、生成物、OpenAPI/operation freshness、68/68 覆盖和 CI compatibility diff 已落地。
- Go test/race/vet/govulncheck 与 Web tests/typecheck/build 已通过；同步修复 goldmark 与 gRPC 可达漏洞。
- 用户确认本地数据库均为可丢弃测试数据后，Workspace 已重建 Blog/Asset 数据库并消除历史媒体漂移；首位管理员认领、公开首页/阅读、管理入口、编辑器及真实 201/204 空正文 Playwright 全绿。

## References

- [稳定上下文](context.md)
- [Foundation HTTP Result Contract](../../../../foundation/flightdeck/knowledge/errors/http-result-contract.md)
- [Foundation Error Catalog](../../../../foundation/flightdeck/knowledge/errors/error-catalog.md)

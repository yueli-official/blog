# 统一错误与 HTTP Result 合同

## Goal

让 Blog API 与 Web 完整采用 Foundation 的声明式错误目录、直接成功 DTO 和统一失败反馈，并通过本地 Identity/Asset 组合完成真实浏览器验收。

## Status

Finished

## Current

Blog 已完成统一错误与 HTTP Result 合同：16 个业务错误由 Foundation v1 catalog 单向生成，68 个 operation 的成功结果与业务 errors 已声明并受 CI 门禁保护；公开参数使用稳定命名类型，前端只按 code/params 解析安全反馈。静态、构建和真实浏览器验收全部通过。

## Next

None.

## Progress

- catalog、生成物、OpenAPI/operation freshness、68/68 覆盖和 CI compatibility diff 已落地。
- Go test/race/vet/govulncheck 与 Web tests/typecheck/build 已通过；同步修复 goldmark 与 gRPC 可达漏洞。
- 用户确认本地数据库均为可丢弃测试数据后，Workspace 已重建 Blog/Asset 数据库并消除历史媒体漂移；首位管理员认领、公开首页/阅读、管理入口、编辑器及真实 201/204 空正文 Playwright 全绿。
- 60 个可能产生产品业务错误的 operation 已声明 errors，16 个 catalog code 均有消费者且无失效路由；真实浏览器验证 400 Problem 不含 raw message，并携带稳定 reason。

## References

- [稳定上下文](context.md)
- [Foundation HTTP Result Contract](../../../../foundation/flightdeck/knowledge/errors/http-result-contract.md)
- [Foundation Error Catalog](../../../../foundation/flightdeck/knowledge/errors/error-catalog.md)

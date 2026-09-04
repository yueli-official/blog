# Plan

## P0 — 盘点与合同

- [ ] 清点全部 operation 的业务 errors；success kind/status 已完成 68/68。
- [x] 建立 Foundation v1 catalog、operation manifest 和 CI drift gate。

## P1 — 实现与前端

- [ ] 完成 typed cause 映射审计；201/204、直接 DTO 与 raw message fallback 已收敛。
- [x] 生成 Go/TypeScript/i18n，并接入 Foundation failure feedback resolver。

## P2 — 验证

- [x] Go 全量/race/vet/govuln、Web tests/typecheck/build 和合同 freshness 全绿。
- [x] 通过 Workspace local checkout 完成 Blog 管理与公开阅读 Playwright，不发布版本。

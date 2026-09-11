# blog 资产引用生命周期

## Status
Finished

## Goal
补齐 blog 自有业务素材登记、替换/删除撤销与失败恢复，并如实提交注册声明。

## Current
2026-09-11 本产品已锁定正式 Asset Go v0.4.0；独立工作树无 replace/源码覆盖完成全量 Go 测试、命令构建、go vet 及真实 PostgreSQL 引用事实源回归。业务查询与引用生命周期代码未修改。以下源码候选阶段记录保留，不能理解为本轮重新部署。
2026-09-11 本地源码和真实 PostgreSQL 回归通过；Workspace 独立组合 20260911T063942Z-37976 已启用当前本地依赖。真实上传、两篇文章共享与重复图片、删除、恢复、移除正文的引用数为 2→1→0→1→0；引用中删除素材返回 409，解引用后可删除。测试文章和素材已通过 API 清理，原有 1 份未被文章使用的素材保留。
CLI Playwright 桌面/移动端验证引用弹窗、跳转 URL、无溢出及缩略图实际解码；修复本地 blog Backend 映射和 Account 独立 Asset 图片代理。未发布 SDK、未部署、未提交。

## Next
None。源码候选已部署并复验；正式SDK版本发布与Git提交尚未执行。

## References
- [上下文](context.md)
- [验证和接入](validation.md)
- [共享 SDK 与待接线补丁](../../../../asset/flightdeck/work/2026-09-11-reference-source-sdk/index.md)

## 2026-09-11 线上交付
用户授权的已有线上实例更新已完成，七服务为server-20260911-lifecycle-1。真实服务/机器API及CLI Playwright复验通过，配置、数据和外部Yotta保留。详见[部署记录](deployment-20260911.md)。此前“未部署”描述为本地阶段状态。

## Git 交付（2026-09-11）
用户已授权本地提交。本次纳入本 Work 的实现、相关验证和部署记录；其他工作改动保留，未推送或发布正式 SDK。此前“未提交”为对应阶段的历史状态。提交及范围汇总见 Workspace `flightdeck/work/2026-09-11-target-stop-isolation/commits.md`。

## 正式 SDK 依赖升级

Go 模块改用 `github.com/yueli-official/asset v0.4.0` 并更新 go.sum，依赖图所需的传递版本由 go mod tidy 收敛。真实 PostgreSQL 回归使用事务/连接局部临时表，不变更业务数据。证据目录：`E:/tmp/yueli-asset-sdk-upgrade-20260911/`，本产品的 `blog-results.json` 与 `blog-reference-db.log`。此次只升级后端依赖，无页面或业务代码修改；沿用此前 CLI Playwright 的界面验收，不将其称为正式依赖运行中的新页面验收。未重新部署、未推送产品分支。

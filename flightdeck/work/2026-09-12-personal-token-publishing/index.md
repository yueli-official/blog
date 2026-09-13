# 开发者令牌一键投稿

Status: Open

## Goal
补齐 Blog 开发者令牌的分类、标签、封面和系列等投稿操作，使程序可完成内容组织、上传和发布，并保持令牌权限与当前账号业务权限的交集。

## Current
2026-09-12 用户授权的本轮生产更新已完成：所属服务使用 `server-20260912-1`，九个相关服务均 healthy。独立构建、线上桌面/手机、登录刷新及普通账户 PAT 边界检查通过；Docs robots.txt 仍有已定位的历史 500。使用固定私有源码/包候选，未发布正式 SDK 或推送 Git。见 [本轮部署与限制](deployment-20260912.md)。

2026-09-12 按用户继续处理缺口的要求，已修复 Blog 构建工具链、CI 合同生成器版本、PAT 部署参数和 CI 缺失的 Asset Token URL。Foundation Go v0.5.0 固定候选通过 tidy 零差异、race、vet、漏洞扫描；Blog 在 GOWORK=off、无 replace、独立模块缓存下安装候选与正式 Asset v0.4.0，全量测试、合同/OpenAPI、模块校验与四个 Linux 命令构建通过。三种 Compose 拓扑的 6 组模型解析检查通过；本机无 Docker，未构建或运行容器。

2026-09-12 已完成本地实现与验收。Blog 动态目录提供 13 项权限，开放分类标签管理、文章归类、系列管理与归属、封面、回收恢复和推荐置顶；所有操作仍检查账号实时能力及令牌 scope。封面须同时有 media.upload 与文章编辑权限，Asset 仅授予 Blog 正文/封面 Profile。管理员令牌设置他人文章系列不再误依赖未开放的 authorization.manage。

Go 全量测试、vet、错误目录与 70 项 HTTP 合同检查通过；CLI Playwright 真实账户中心桌面/手机选择、完整投稿和发布页面通过，最终 68 次请求检查、0 pageerror，13 项收尾请求全部成功。临时令牌和角色已撤销，测试分类系列已删除、文章移入回收站。详见验收记录。

仅修改 Blog；未提交、未部署生产。正式 Foundation v0.4.1 不含 PAT API，已制作候选和待应用依赖升级补丁，主工作树依赖尚未切换到未发布版本。Identity v0.3.3 / Asset v0.4.0 正式服务也没有当前 PAT 目录/媒体回查，完整模板部署还需相应 Provider 发布及组合验收。独立本地组合 20260912T071938Z-54828 沿用既有运行状态，未在本轮操作其生命周期。详见 Foundation 候选记录。

2026-09-13 生产 PAT 图文投稿补充验证：使用现有全权限 PAT 向 `https://cg.yuelili.com` 创建 3 篇 `ae` 分类草稿，分别含 6/1/4 张正文图片并各有封面；共 14 个媒体交付地址真实 GET 全部 `200 image/webp`。直接把源 PNG 交给当前生产图片接口时复现 `502 blog.upstream_failed`（dependency=`asset`）；按当前消费者注册先将静态图预处理为 WebP 后，封面与正文上传、Finalize、文章 PATCH、分类设置及读取验证全部成功。该结果验证当前生产 PAT 投稿链路，但不等同于本地尚未部署的服务端图片处理增强已上线。用户侧映射记录写于 `E:/projects/docs/Note/文档markdown/CG-Blog投稿记录.md`，不含令牌。

2026-09-14 Blog 后台文章列表补齐“批量加入系列”：沿用现有 `PUT /api/v1/posts/{id}/series`，短弹窗选择/搜索系列，逐篇提交；成功项从选择中清除，失败项保留并继续使用列表区域批量结果反馈。Blog Web 已切到 Foundation `js-v0.7.5` 的 `@yueli/content-nuxt 0.2.5`，原 Mammoth browser 声明类型错误消失。`pnpm install --frozen-lockfile`、文章 UI 合同测试 11/11、`git diff --check` 通过；CLI Playwright 使用真实本地登录完成 1440px 批量选择→系列选择→提交→`/api/v1/posts/mine` 回读，以及 390px 弹窗边界验收，临时文章/系列已清理。完整 typecheck 仍有两项与本功能无关的既有错误：`AuthorizationGrantBadge` 当前包未导出，以及评论页 `layout="columns"` 不符合现有类型。未提交、未部署。
## Next
本轮生产更新已完成，新增权限已随固定私有候选上线。正式 Foundation go/v0.5.0 发布与模板正式依赖升级仍按 foundation-release.md 单独推进，不等同于本次服务器更新；旧令牌不会自动获得新增 scope。

## References
- [上下文](context.md)
- [投稿指南](../../../docs/developer-tokens.md)
- [验收记录](acceptance.md)
- [Foundation 固定候选与独立验证](foundation-release.md)
- [待应用的依赖升级补丁](foundation-dependency.patch)
- [令牌实现](../../../api/internal/controller/personal_tokens.go)
- [原基础能力验收](../../../../foundation/flightdeck/work/2026-09-09-personal-token-authorization/acceptance.md)

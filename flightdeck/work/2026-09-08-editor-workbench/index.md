# blog 内容编辑工作台

Status: Finished

## 已完成

- 接入共享 EditorCommandBar：沉浸、公开页、设置、发布/下架、保存；窄屏标题与操作分行。
- 列表提供公开页、快速编辑、完整编辑与双击编辑；选择框与按钮不会误触发导航。
- 发布前保存当前内容；下架意味着匿名原链接不可读，不提供仅链接可见选项。
- 本地 Playwright 覆盖 390/768/1024/1440px、浅深色、沉浸/设置、快速编辑、双击、上下架实际 200→404→200。
- 类型检查和构建通过；[浏览器证据](references/browser-local.json)。

## 交付边界

独立候选构建完成，线上 server-20260908-editor-1 健康；公开页面 390/1440px 复查通过。生产管理员编辑交互未冒充验证，本地已完整验证。

[线上公开页面复查](references/browser-production.json) · [制品与源文件校验](references/candidate.json)。

## 图片输入与文档导入（2026-09-09）

共享 ContentEditor 已支持图片上传框、多图拖拽/粘贴，以及 Markdown、HTML、DOCX、ZIP 和富文本剪贴板导入。图片走原有 Asset Adapter，完成后一次插入，缺图失败不覆盖原文。本站前端已更新 server-20260909-import-1；类型检查、生产构建、线上公开页与编辑器静态资源检查通过，完整真实上传/导入交互在 WWW 本地完成。没有冒充本站线上管理员写操作验收。

[共享实现与验收记录](../../../../foundation/flightdeck/work/2026-09-09-editor-import/index.md)。

## 共享编辑器依赖固定（2026-09-09 完成）
package.json 与 pnpm 锁固定 `content-nuxt 0.2.3-server.20260909.placeholder.1`，将已验证包放入 web/vendor；五个站点的编辑器包 SHA-256 一致。配套 UI、Asset、HTTP 与会话依赖固定为已经部署验收的 WWW 组合，Tiptap 全组 3.31.3，避免旧正式包缺少当前消费接口。此举不代表发布 npm/GitHub Release。
该编辑器已在本轮之前部署到线上，Web 为 server-20260909-placeholder-1；此次把仓库依赖同步到同一编辑器制品，没有重复部署。其正式构建与线上检查证据归 Foundation 编辑器导入 Work。
依赖审计：E:/tmp/yueli-editor-consumers-20260909/dependencies.json。保留其他未提交改动；本轮未提交或推送 Git。

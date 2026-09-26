# Blog 后台品牌布局

Status: Finished（已验收并部署 `server-20260926-admin-brand-1`）

## 交付

2026-09-26 用户确认 Commerce 方案后，授权推广本产品后台。顶部标题卡、真实统计、集合卡与紧凑工具栏沿用本站蓝色。覆盖总览及文章、评论、分类、标签、系列、设置、资源策略、权限与申请。公开页面和沉浸式编辑器不启用后台品牌主题。

- Foundation `AdminOverview` 共享排版与窄屏让位逻辑，`AdminMetricCard` 共享统计卡；产品通过 `artwork` 插槽提供独立的稿纸与羽笔 SVG，图案不放进公共库。
- 搜索/筛选移到集合内部，表头保持轻背景，保留原有权限、筛选、排序、分页和编辑流程。
- 窄屏按内容摘要组织，列表操作使用共享 `AdminRowActions` 的更多菜单。

## 验收证据

- 当前代码 `pnpm typecheck` 退出码 0。
- 共享 UI typecheck、141/141 单测、tarball 独立消费者安装/构建通过，包含两个新组件及主题 CSS。
- 三站接入共享组件后的生产构建均通过。随后本产品窄屏样式、菜单及文案微调通过最终类型检查、Nuxt 实页编译和 Playwright；未对最终微调重新生成生产制品。
- CLI Playwright 全后台检查 34 项通过；最终受影响列表复查 12 项通过。覆盖 320/390/768/1024/1440，深浅色偏好、主容器与表格溢出、Tabs、真实统计与装饰插槽、搜索/清空、筛选、更多菜单快速编辑/取消、公开页主题隔离。代表页 Axe WCAG AA 零违规；最终运行无捕获到的页面错误或水合告警。WWW 按主站设计始终保持黑银色。
- 证据：[完整报告](../../../web/test-results/admin-brand-20260926/report.json)、[最终列表复查](../../../web/test-results/admin-brand-20260926/report-list.json)、[桌面总览](../../../web/test-results/admin-brand-20260926/light-dashboard-1440.png)。生成物在 Git 忽略目录。

## 恢复与边界

## 追加复查

- 修正资源引用提示的表面颜色；权限申请空态隐藏无用全选框，并压平嵌套集合边框。
- 账户入口接入共享壳顶栏右侧；评论搜索左对齐规则在 Foundation 共享工具栏修复，Docs 同步受益。
- Playwright 实测三站顶栏账户菜单均可打开；390px 下账户按钮可见且页面无横向溢出。Blog/Docs 评论搜索从集合左侧起排；Blog 资源提示为卡片白色；权限空态无复选框且内层集合边框半径为 0。Foundation 与三站类型检查通过。

- 预览：http://blog.dev.yuelili.test:3002/manage
- Workspace Session：`20260926T051209Z-17720`；最后 `dev status --check` 为 ready。保留供用户预览。
- 使用 Workspace 显式 `--local foundation` 的依赖覆盖验证新共享接口；生产站点使用固定预览 tarball 随 Web 镜像交付，未发布 npm 包。
- 不修改 Workspace 合同或顶层 Flightdeck，不停其他产品进程；Commerce 原预览保留。

## 2026-09-26 正式部署

Blog Web 已部署 `yueli/blog-web:server-20260926-admin-brand-1`，容器 healthy、RestartCount=0；API 未变。正式域名 Playwright 检查首页 1440/390px 无横向溢出或 pageerror，`/manage` 未登录正确跳转 Account `/login`。没有使用生产管理员会话验证写操作。

文章窄屏隐藏占位封面与冗长 slug，标题最多两行，显示真实状态/更新时间；封面加载失败回退到已有占位图。

# Blog 紧凑后台布局

2026-09-08 用户要求先将 Gallery 已验收的四项设计应用到 Blog 在线版，再决定其他站点。

## 交付
文章、评论、系列三页采用共享紧凑分页；文章网格使用自动列数，并移除网格中的表格排序标题。评论采用 Foundation compact 阅读布局。三页搜索进入标题 tools slot，文章筛选与排序使用共享弹窗、视图组显示选中状态；主操作窄屏保持标题右侧。系列不再固定第一页，按实际总数切片分页。

共享新增 PageHeader tools slot、CollectionHeaderTools、CollectionPanel compactGrid。其他消费者未启用新属性时保持原行为。

## 首版验证（admin-1）
- Blog 与 Foundation 类型检查通过，Foundation 35 文件 / 106 测试通过，设计扫描无发现。
- CLI Playwright：390/768/1440/3840 三页无溢出/pageerror；文章网格分别为 2/3/5/17 列；切换、排序应用、筛选取消、主操作位置通过。
- 评论有内容与 100 页折叠使用浏览器响应夹具验证，不写数据库。
- 独立 tgz + lock 安装与 Nuxt 生产构建成功，无相邻源码 overlay；不是正式 Release。
- 服务器 Web 为 yueli/blog-web:server-20260908-admin-1，Compose --wait healthy；API 保持 server-20260908-users-1，无迁移。
- 线上 CLI 浏览器确认 HTTPS 和新标题工具区加载。历史部署会话已失效；另一个既有测试会话无作者权限，文章 API 403，未提升权限或代用户认领。因此不声称线上管理员全交互验收通过。

证据目录 E:/tmp/yueli-media-preset-20260908：blog-admin-browser-report.json、blog-admin-source-manifest.json、blog-admin-build.log、screenshots/blog-admin-*。远端 /projects/yuelili.com/.deploy/media-preset-20260908；blog-compose.before-admin.yaml 是更新前配置备份，不能覆盖其他产品。

## 后续
用户已确认 Blog 布局并授权推广；共用规则由 Foundation 知识拥有，各产品记录自己的验收与部署。

## 分类、标签与分页补齐
2026-09-08 用户指出遗漏分类/标签页，并要求恢复首页/末页、每页数量改为“数量 / 页”。TaxonomyManager 已接入共享标题工具和紧凑分页，排序弹窗沿用现有名称/Slug/文章数和正倒序；无业务筛选条件，不增空筛选按钮。分类树先保持完整层级排序，再对结果切片；服务端分页的标签/搜索结果保持原协议。共享分页恢复 first/last，选项显示“30 个 / 页”等。

Blog typecheck、共享分页两项测试、390/1440 分类/标签/文章 CLI 浏览器通过；浏览器 3000 条分类夹具验证 100 页末页与首页往返，没有写数据库。证据 blog-taxonomy-report.json、blog-taxonomy-build.log、screenshots/blog-taxonomy-*。候选 Web server-20260908-admin-2。

## 用户确认后的最终推广版本
Blog Web 已更新为 `server-20260908-admin-4`，包含权限页剩余私有分页接入与共享默认布局。独立候选构建、类型检查及正式页面两种宽度检查通过；线上系列页共享分页和标题工具可见，容器 healthy。API 与数据库未更改。源码输入哈希为 `admin-4-source-inputs.json`，制品校验为 `admin-4-web.sha256`，正式页面证据为 `admin-4-production-result.json`，均在本轮外部制品目录。

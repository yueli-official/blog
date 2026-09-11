# Blog 服务器候选部署

## Current

2026-09-07 已部署；用户反馈后已纠正管理员初始化，Blog 重建为未认领状态，等待用户自己认领。SSH 别名 bt；Ubuntu 24.04 / amd64，2 CPU、7.5 GiB RAM，1Panel v1.10.34-lts，Docker 28.0.1 / Compose 2.33.1。

| 入口 | 用途 | 宿主机回环端口 |
| --- | --- | --- |
| https://cg.yuelili.com | Blog Web / 同源 BFF | 13002 |
| https://account.yuelili.com | Account / Identity OIDC | 13000；/oauth2/ 转到 18081 |
| https://asset.yuelili.com | Asset API 与媒体 | 18082 |
| 无公开入口 | Notification API，由 Account 管理 | 18089 |
| Account `/commerce-api` 仅公开支付回调 | Commerce API，由 Account 管理 | 18084 |

Blog API 仅映射 127.0.0.1:18085。两个 PG 不发布宿主机端口。

## 目录与所有权

五份 Compose 位于 `/projects/yuelili.com/{databases/core,databases/content,identity,asset,blog}/docker-compose.yml`。

- core：PostgreSQL 18.4，容器 yueli-pg-core，Identity、Asset、Notification 和 Commerce 分库分账号。
- content：锁定源码部署合同的 PG16/zhparser 镜像，容器 yueli-pg-content，Blog 专属库和账号。
- PG 数据绑定到所属 databases/<group>/data/postgres，不属于应用编排。
- Identity 的 Redis、Publisher 运行密钥和离线根密钥分别持久化到 identity/data/redis、publisher、publisher-offline；离线根目录不挂载进运行中的 API。
- Asset 媒体持久化到 asset/data/assets。各产品 config/runtime.yaml 从部署输入生成，权限受限，不能输出或提交秘密。
- 首次初始化发现 GoFrame 数据库结构读取未采用原 Compose 嵌套环境变量，因此使用显式运行配置文件挂载。

1Panel API 尚未启用，Compose 由 SSH 启动，面板可能标为 Local。开启 API 后用面板“路径选择”导入上述现有文件；不要直接改 1Panel SQLite 伪造记录。

代理写在 1Panel 各站 proxy/yueli-app.conf；Account 另有 proxy/yueli-oidc.conf，直接同源转发 OIDC 到 Identity，保留浏览器 302 回调。Account 应用代理会在服务器端跟随重定向，导致回调缺少浏览器事务 Cookie，不能撤掉这条显式代理。cg 旧 WordPress rewrite include 已停用，旧文件及静态目录保留。

## 制品与源码

- Identity 当前为 yueli/identity:server-20260907-8，Account 为 yueli/identity-account:server-20260907-8，Notification 为 yueli/notification:server-20260907-1，Commerce 为 yueli/commerce:server-20260907-2；包含邮箱验证 ID、平台管理员认领、登录配置验证弹窗/超时与邮件管理。Asset、Blog API/Web 仍为 server-20260907-1。配置修复与限制见 [Identity Work](../../../../../identity/flightdeck/work/2026-09-07-provider-administration/index.md)。
- Blog bootstrap：yueli/blog-bootstrap:server-20260907-2；正常空库必须创建分类 catalog 与默认 policy，不能依赖 BLOG_DEV_SEED。
- 后端从干净 HEAD 快照按 GOWORK=off / Linux amd64 / CGO_ENABLED=0 编译，使用已发布 Go 依赖。
- 正式 Asset Nuxt 0.1.0 缺少 upload / tailwind.css 导出。本次从 Foundation、Identity Nuxt、Asset Nuxt 的 HEAD 打包七个独立 server.20260907.1 候选 tgz，再做独立 lock、frozen install 与 Nuxt 生产构建；没有 sibling overlay，没有创建或移动 tag、Release。
- Nitro 的 Windows junction 转换为包内相对符号链接，真实 Linux 容器运行已验证。
- 本地忽略目录 .data/server-deploy-20260907 保留源码归档、候选包、lock、构建日志和脚本；远端对应 /projects/yuelili.com/.deploy/20260907。
- source-manifest.json 记录源码 SHA 与包 SHA256；source-fixes.patch 记录 Blog bootstrap、Web health 修复。正式制品发布仍待基础服务发布 Work。
- 不重跑 setup_server.py：初始安装器已添加对现有产品容器的拒绝保护。日常使用当前项目 Compose；初始 bundle 是归档，不能覆盖当前修复后的配置。

## 管理员与备份

前次擅自预设部署管理员不符合用户意图，已纠正：bootstrap@account.yuelili.com 停用，角色、Identity 会话及 OIDC 授权/刷新状态已撤销；用户自行注册的账户和会话保留。仅 Blog 试验库按用户授权清空重建；当前无授权 Grant、无预设 Bootstrap subject。用户自行登录 /manage/setup 显式认领，代理不代认领，也不擅自将普通账户提升为全局管理员。

旧 bootstrap_server.py 已禁用，原脚本仅归档为 .disabled；bootstrap-account.json 已改为 revoked 标记，旧密码不能使用。Blog 密封密钥已随重建轮换，使旧部署账号的产品会话失效。Identity 的 ADMIN_EMAIL 与 runtime rbac.bootstrapAdminEmail 均为空，不能在后续更新时恢复旧管理员。

纠正后的加密备份：/projects/yuelili.com/backups/20260907-claim-reset/initial.tar.gz.enc，已复制到本机 .data/server-deploy-20260907/claim-reset-backup/。解密密钥单独位于私有目录 backup-key.txt。旧 initial / before-claim-reset 备份仅为历史归档，不应恢复其管理员归属。

备份包含三份逻辑数据库 dump、配置和密钥、媒体、站点代理及初始化资料。已验证 pg_restore --list、加密后解密字节一致、跨机器 SHA256 一致；未进行整站灾难恢复演练，未配置定时异地备份。制品另有本机 bundle.tar.gz 与修复后的 bootstrap-fixed。

## 验收证据

- 八个常驻新容器 healthy；初测合计 Docker 报告内存约 332 MiB，不是负载上限。两个产品账号互访对方数据库被拒绝。
- 三个数据库 migration 成功，未修改旧迁移。独立临时库通过正常初始化幂等/无演示数据与演示夹具幂等两项 Go 集成回归，测试库已移除。
- 真实 HTTPS OIDC 登录进入 /manage；后台三次刷新均 200，SSR 包含 data-admin-shell，无固定中转占位。
- UI 保存 200；图片 init 201 → 同源 PUT 200 → finalize 200 → WebP 交付 200；临时文章发布、公开阅读及图片渲染成功，随后移入回收站。
- 文章与后台完成 390/768/1024/1440px × light/dark，无横向溢出或 pageerror；移动编辑器设置面板 x=0、width=390。
- 首页、控制台、文章列表、设置页 Axe serious/critical 均 0。
- 证据：web/test-results/server-20260907/ 下的 report.json、accessibility.json 和截图。
- 未完整复验评论提交、多用户权限、SMTP/外部登录、故障恢复或正式 Release 安装，不把它们描述为已通过。

## Next

基础服务平台管理员已支持 https://account.yuelili.com/setup 显式认领；资源与邮件控制面复用该平台权限，仍不改变 Blog 站点归属。Identity 新增迁移 0032–0034，旧迁移字节未覆盖。正式平台认领由用户本人操作。

纠正后曾验证 Blog claimed=false、canClaim=true，匿名 POST claim 返回 401，代理没有点击认领。平台认领交付后的最后只读检查中，Blog 已为 claimed=true；后续继续用户目检，不重建或重新开放该站点。

“发送失败”已通过原 PG 集成测试复现为 email_verifications.id 非空约束错误；补齐 Foundation UUIDv7 写入后，同一真实临时 PG 测试通过（覆盖单次消费、过期、用途隔离与 UUIDv7），HTTP 邮箱验证/密码重置捕获邮件测试也通过，临时测试库已删除。补丁记录在远端 .deploy/20260907/identity-verification-fix.patch。

Identity 已接入 Notification，后台 https://account.yuelili.com/admin/mail 可管理 SMTP 并测试发送。SMTP 已由用户提供凭据并授权启用（gz-smtp.qcloudmail.com:465/TLS，发件人 account@mail.yuelili.com），持久化配置的认证检查通过，真实收件仍待用户测试；不会回退到日志假投递。最新四库加密备份位于 `/projects/yuelili.com/backups/20260907-provider-admin/`，本机副本在 Identity `.data/provider-admin-20260907/backup/`。Google 容器代理链路已接通；用户新订阅的 71 个节点已加入 Mihomo，当前选择新订阅日本东京09，Google 网络探测已恢复 200，真实授权登录仍由用户验收。

1Panel API 开启后纳管已有 Compose；不重复部署、不恢复旧管理员、不替用户认领。

本机失败的第一次 bundle 复制目录清理被自动审批拦截（仅返回 blocked by policy），保留在忽略目录；不影响服务器、成功制品或备份。

2026-09-07 支付管理交付：`/admin/payments` 可配置三种商户渠道，`/admin/platform` 已简化；商户凭据仍为空，不能声称真实收款已验收。Commerce 独立配置与迁移在 `/projects/yuelili.com/commerce`。最新备份为 `/projects/yuelili.com/backups/20260907-commerce-configured/`（五库），Identity 本机 `.data/provider-admin-20260907/payments/backup/` 已保存同校验和副本。


## 2026-09-08 权限页资料更新

当前本站 API/Web 均为 `server-20260908-users-1`，容器 healthy；包含此前媒体 preset、Docs 多语言标题及侧栏更新。权限页面补齐用户资料与时间，无数据库迁移。制品与部署校验保存在 `E:/tmp/yueli-media-preset-20260908`。

## 2026-09-08 紧凑后台布局
Blog Web 当前 server-20260908-admin-1；API 仍 server-20260908-users-1。无迁移，制品校验、容器 healthy 与本地浏览器通过，线上测试账号无作者权限。详见[布局交付](admin-layout.md)。

当前 Web 已更新为 server-20260908-admin-2（分类/标签接入、分页首页末页和数量/页）；容器 healthy，线上 CLI 浏览器在系列页确认新版共享分页，分类/标签治理交互仍以本地管理员验收为准。API 未更新。

## 2026-09-09 当前源码同步

用户授权更新生产。Blog API/Web 已切至 `server-20260909-sync-1`；PAT 能力目录和受限 BFF 转发已接入 Identity/Asset，七个固定候选包与独立构建通过，容器 healthy。正式首页 390/1440、OIDC、权限目录可达与未选权限拒绝通过；未用生产测试账号代用户创作。共享 Provider 与完整证据见 [Docs 部署记录](../../../../../docs/flightdeck/work/2026-09-09-project-docs-publishing/deployment.md)。未提交 Git 或发布正式依赖 Release。

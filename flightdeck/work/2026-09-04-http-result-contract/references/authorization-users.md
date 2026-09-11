# 权限页用户信息修复（2026-09-08）

- 根因：Identity 批量用户接口实际返回 `{items}`，两站页面按 `{users}` 读取，导致昵称映射为空；Docs 申请行还直接渲染 subject。
- 两站改用真实公开资料，显示昵称、头像与 Account `/u/{userKey}` 主页链接。头像由 Asset SDK 生成 preset URL。资料加载失败显示明确反馈。
- 申请显示已有 createdAt；用户授权显示授权生效时间，API 补出既有 grant.ValidFrom。没有数据库迁移，没有改变用户、申请或权限数据。
- 两站 Go controller 回归测试、OpenAPI 合同检查（Docs 72 / Blog 68 operations）、前端 typecheck/build 通过。
- 已部署：Docs 和 Blog 的 API/Web 均为 `server-20260908-users-1`，容器 healthy。生产只读核验两站 Identity 代理返回昵称 yueli、头像 200、Account 主页可达、匿名管理入口正确要求登录。未冒充线上管理员执行操作。
- 本地 Blog CLI Playwright 已验证真实昵称、主页跳转、1440/390 无横向溢出、三次 SSR 刷新，无 pageerror。申请是只读响应夹具；共享旧 Blog API 的授权时间亦用夹具，不重启其他会话服务。
- 本地 Docs CLI Playwright 补验已通过：真实 Identity 昵称、真实 API 授权时间、Account 主页跳转、1440/390 响应式、三次 SSR 刷新，无 pageerror；申请仅使用只读响应夹具。手机截图已目检。此前登录探针继承已关闭代理 127.0.0.1:10808；设置本次进程 NO_PROXY 后组合与浏览器认证正常，无需产品改动。验收结束已通过 Workspace CLI 停止本会话 Docs 隔离组合，保留既有共享 Provider/Blog。
- 正式制品、部署脚本、校验和与浏览器报告：`E:/tmp/yueli-media-preset-20260908`（deploy-users.py、authorization-users.mjs、authorization-users-report.json）。无 Git 提交或推送。

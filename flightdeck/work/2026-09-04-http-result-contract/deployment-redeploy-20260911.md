# 2026-09-11 已上线实例重部署

本产品已完成用户授权的其他改版提交、候选验证及线上切换。服务：`blog-api、blog-web`；最终镜像标签：`server-20260911-redeploy-1`；健康检查通过。此次部署完成不改变本 Work 中其他事项的状态。

固定源码候选通过 Go 全量测试、静态检查和 Linux 构建；有前端的产品同时通过单测、类型检查和生产构建。七库备份验证可读取，原运行配置与已执行迁移字节保留，本轮没有新增数据库迁移。

线上 CLI Playwright 覆盖六站桌面/手机页面及图片、五个产品的登录跳转、WWW/Docs 导航、Commerce 钱包/收益/账单、Paste 添加文件交互，以及已有真实账号的 Commerce 页面和 Account 开发者令牌页面；13 项页面/视口检查、16 项交互检查通过，无页面异常。没有在生产执行管理员编辑、支付、创建令牌或内容修改，不据此宣称这些流程已重新完整验收。

Asset 库存、注册策略与已有引用保持不变，Docs 正文 1338 条引用（393 个素材），Account 头像/封面各一条引用；Blog/WWW 机器身份读取 Asset 注册均为 200，引用同步无错误。外部 Yotta 未变更。

本机证据：`E:/tmp/yueli-sites-redeploy-20260911/`；服务器制品和检查：`/projects/yuelili.com/.deploy/sites-redeploy-20260911/`；备份：`/projects/yuelili.com/backups/sites-redeploy-20260911/`。源码 SHA/制品哈希见 `release-manifest.json`，健康与配置见 `verification.json`、`config-verification.json`；本机浏览器结果见 `acceptance.json` 和 `screenshots/`。

Go 引用消费者使用正式 Asset v0.4.0；Foundation Go 与前端共享模块使用已提交源码的固定候选，未另行发布其正式 SDK。产品分支本轮仅本地提交，未推送。当前线上清单与重部署方法见[Workspace 部署表](../../../../workspace/docs/deployments.md)。

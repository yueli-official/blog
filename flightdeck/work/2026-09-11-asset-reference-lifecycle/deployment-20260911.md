# 2026-09-11 线上资产引用机制更新

用户明确授权部署已有在线站点。已更新 Asset、Identity、Docs API、CG Blog API、WWW API、Commerce API 和 Account Web，七个镜像均为 `server-20260911-lifecycle-1`。其他产品 Web 保留原制品；未上线的 Gallery/Resource/Shop/Licensing 未创建线上实例，外部 Yotta 未修改。

## 结果

- Blog、WWW 启用专用机器身份和15秒权威快照对账；原站点配置与凭据保留，新增凭据仅保存在服务器私有文件中。新客户端真实 OAuth token 和 Asset registration GET 均成功。
- Account、Docs、Blog、WWW 声明升至 revision 2，referenceLifecycle.complete=true。使用生产原声明仅补充生命周期，WWW增补自身机器身份绑定；原 profiles/storage policy 完整相同。Yotta仍为revision 1且未声明，没有代其标记完善。
- 原线上引用保持：Docs正文393张素材/1338条引用、文档集封面1条、账号头像与封面各1条。CG与WWW生产当前没有自有素材；自动登记能力已启用，不能声称本次新补了不存在的素材。
- Pay当前无文件订单、无Asset服务身份配置。已更新消费者代码，条件式引用worker仍不启用；未创建消费者、导入演示目录或操作资金。未来正式文件消费者接入须配置自己的服务身份和site范围。
- 没有运行数据库迁移、改写原配置值、替换媒体对象或删除未使用素材。六库备份可读取，Compose/运行配置/.env逐项对比只出现预期镜像与新机器身份接线，原迁移字节和库存相同；部署后配置亦私有备份。

## 验证与边界

六API候选定向Go测试与Linux构建通过。干净Account源码快照完成安装、frozen lock、typecheck和正式构建，不包含临时验收路由。依赖为固定Asset源码及原部署Foundation源码快照；Identity候选锁补入共享Markdown解析依赖。属于独立源码候选部署，不是正式SDK版本发布；未Git提交、推送或创建Tag。

七个更新服务和四个保留Web共11项健康，引用worker启动后无对账错误。CLI Playwright在CG、Docs、WWW、Pay的390/1440页面通过渲染、无横向溢出和无pageerror；四站登录跳转到正式Identity。使用既有真实生产普通账号完成Pay OIDC，并读取钱包、收益、账单页面，未做支付操作。

Account真实线上制品在390/1440验证四项“已声明完善”、一项外部旧声明、revision 2历史展开。此后台浏览器测试使用生产数据库只读导出的注册数据作为API夹具，并非冒充管理员真实登录；真实M2M与数据库验证另行执行。

候选预检首次因非root容器无法读取私有目录下的声明文件失败；只调整无秘密声明文件的读取权限后通过，发生在切换前。未造成生产服务失败。

## 证据

- 本机 `E:/tmp/yueli-reference-deploy-20260911/`：源码/依赖快照、Git SHA与补丁、SHA256、构建日志、verification.json、config-verification.json、acceptance.json、registration-browser-report.json和截图。
- 服务器 `/projects/yuelili.com/.deploy/reference-lifecycle-20260911/`：固定镜像构建上下文、切换脚本、原状态/当前验证、真实注册数据、私有操作日志。
- 备份 `/projects/yuelili.com/backups/reference-lifecycle-20260911/`：六库pg_dump、前后配置和SHA256SUMS。

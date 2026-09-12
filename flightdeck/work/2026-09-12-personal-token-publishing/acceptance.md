# 验收

日期：2026-09-12。修改范围为 Blog 开发者令牌投稿授权，未部署生产。

## 自动化

- 使用 Workspace CLI 生成的 `20260912T071938Z-54828/go.work`，其中 Foundation、Identity、Asset、Blog 均显式选择本地 checkout。
- `go test -timeout 60s github.com/yueli-official/blog/api/...` 通过；需要 BLOG_PG_HOST 的旧集成测试未在本轮 go test 中开启，避免其重置共享数据库。真实 PostgreSQL 验证由下述独立组合执行。
- 新增端点准入允许/拒绝矩阵、当前管理员/作者动态权限目录、目录不对 PAT 开放、媒体 Profile 限制、封面缺上传权限拒绝、仅 scope 不产生管理员等测试。
- 原管理员撤权测试继续通过；新增真实场景验证撤销临时管理员后，原令牌的分类修改及他人文章系列赋值均从允许变为 403。
- `go vet`、错误目录生成物检查、HTTP operations 生成物检查、HTTP contract checker 通过，合同仍为 70 项操作。发现并修复原生成器遗漏两项 PAT 接口错误声明的问题，生成后合同文件内容保持原状。
- `git diff --check` 通过。无 DTO 或 OpenAPI 路径变化。

## 真实服务与 CLI Playwright

运行入口：[personal-token-publishing.mjs](../../../web/test/e2e/personal-token-publishing.mjs)。证据在忽略的 `web/test-results/personal-token-publishing/`。

最终 `acceptance.json`：68 次请求检查通过、0 pageerror、13 项收尾请求全部成功。

- Account 实时发现 13 项 Blog 权限，桌面 1280 与手机 390 选择和创建临时令牌通过；截图已检查，无水平溢出。
- 使用 PAT 经 Blog 同源 BFF 创建父子分类、修改分类、创建及合并标签、创建文章并设置分类标签。
- 创建/编辑系列、设置文章系列顺序、推荐置顶通过。
- 正文与封面均执行真实上传初始化 → PUT 字节 → Finalize；发布后检查分类、系列、封面 DTO，以及公开页面正文图片加载。
- 仅标签 scope 创建分类/修改分类 403；缺少上传或编辑 scope 上传封面 403；仅编辑设置推荐置顶 403；仅发布夹带正文修改 403。
- PAT 用户授权治理与后台批处理被拒绝；普通作者看不到分类管理权限，不能编辑他人文章归类或系列。普通作者可创建标签及给自己文章归类。
- 临时管理员仅持文章编辑 scope 可设置他人文章系列，不要求额外授权管理 scope；撤权后立即拒绝。
- 下架、回收、恢复、删除系列、删除分类标签通过。令牌撤销后请求 401。
- 临时令牌和角色已撤销；测试分类系列删除，文章移入回收站，保留本地测试用户和资产/历史记录。

首次收尾因验收脚本遗漏合并源记录而先删除合并目标，被既有替代关系外键拒绝；已按源→目标顺序通过业务 API 清理，并修正脚本保留合并源及严格断言收尾成功。最终运行额外验证了 PAT 删除系列、分类和标签，全部通过。初次记录保留为 `acceptance-initial.json`。

## 运行与发布边界

旧 Blog Session 已 stale，通过 Workspace CLI 关闭后恢复独立组合，未停止其他站点或共享 Provider。

当前 Session：`20260912T071938Z-54828`。浏览器入口：`http://blog.dev.yuelili.test:3002`、`http://account-blog.dev.yuelili.test:3600`。

沿用原 PAT 验收配置：GF_PAT_APPLICATIONS 只声明 Blog 本地权限目录；GF_BLOG_PERSONALTOKENS_SITEID=blog-main-web；GF_ASSET_PERSONALTOKENS_AUTHORITIES 指向 Blog 的媒体授权接口；GF_ASSET_BACKENDS 声明本地 local/blog Backend。后端仅绑定回环地址，前端按 Environment 合同开放开发域名入口。

正式 Foundation v0.4.1 缺少已有 PAT API，未覆盖依赖直接编译仍失败。本轮没有发布基础库、改生产部署合同或部署；不得将本地覆盖验收视作正式依赖/线上验证。

## 后续：独立依赖与构建缺口

用户随后要求处理上述缺口。已完成 Foundation 固定 Go v0.5.0 候选门禁，以及 Blog 无 GOWORK/replace 的独立安装、全量后端测试、合同/OpenAPI、go mod verify 与四项 Linux 构建。修复 Blog Docker 工具链和 CI 生成器锁定，补齐 PAT 运行配置及 Asset Token URL。Compose 三种拓扑的 CI/示例环境 6 组模型检查通过；本机无 Docker，未执行镜像构建或容器启动。

详细版本、哈希、日志路径、发布说明与剩余 Provider 正式版本边界见 [Foundation 候选记录](foundation-release.md)。这是本地候选独立消费验收，不是已发布 Go Proxy/SumDB 验证。主工作树 go.mod/go.sum 暂未指向未发布版本；待应用补丁已准备。既有 Playwright 68 次请求结果保留，后续没有修改前端或业务 Go 实现，因此未重复运行。

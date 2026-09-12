# Foundation Go 正式依赖候选

日期：2026-09-12。此文件记录固定候选与独立消费者证据；没有创建本地/远端 tag、GitHub Release 或更新生产。

## 候选

- 仓库：`yueli-official/foundation`。
- 建议 tag：`go/v0.5.0`；模块版本 `github.com/yueli-official/foundation/go v0.5.0`。兼容新增按发布策略升 minor。
- 固定提交：`3ad8f48f942ddf5b1e7370b0d9b16fcc9bb999b2`，已在本地提交，源码工作树无改动。
- 核实远端最新 Go tag 为 `go/v0.4.1`；本地提交领先远端 main 27 个提交，无分叉。候选覆盖固定提交中的整个 Go 模块，不只截取 PAT 文件。
- 使用 `golang.org/x/mod/zip.CreateFromVCS` 从该提交打包，未依赖当前工作树文件或 `replace`。
- 模块 Sum：`h1:TLCiB6PKoCiBF9WXLBEQQdQDegVCeVII584ybH2vjZI=`。
- go.mod Sum：`h1:vZ++Q3f4Q7pFzvcCR6bLTORM8vy5dy1StdU1X1A3IrQ=`。
- 本地模块 ZIP SHA-256：`4a435988c6379e26d60966a83b25dc7852a3c7f5bff04c015a46ae1f6086027b`。

## 发布说明候选

Foundation Go v0.5.0 将个人访问令牌认证与权限发现纳入可独立安装的 Go 模块，使 Blog 等消费者能够脱离 Workspace 源码覆盖进行构建。

- `auth` 新增在线 PAT 验证、站点能力 scope 编解码、JWT/PAT 组合验证和个人令牌标记；失败的 PAT 不回退为 JWT，验证不缓存成功结果。
- 新增由消费站点拥有的权限目录客户端与 DTO。令牌 scope 仅限制用户已有权限，消费者仍须执行账号实时授权及资源权限检查。
- `authorization.EffectiveAccessQuery.IncludeDescendants` 支持发现下级作用域可委托能力，不能代替资源授权；PostgreSQL 投影重建对序列化冲突和死锁进行有限重试。
- `httpcontract` 增加项目级合同生成与检查、额外成功响应声明及兼容性比较，完善 OAuth 原生成功响应支持。

迁移：更新模块版本并执行 `go mod tidy`。已有 JWT 消费者可继续使用原入口；启用 PAT 的消费者需显式登记站点 ID、权限目录和验证地址，并保留每次业务授权。新增字段为可选；本次不修改业务成功 DTO 为 Envelope。

本候选只交付 Go module，不发布 Foundation JS bundle，也未声称 JS 发布门禁已执行。

## 独立验证

证据根目录为忽略的 `blog/.data/pat-foundation-release-20260912/`，包含候选模块代理、固定源码解包、独立消费者、逐项日志、JSON 结果和 Linux 二进制。

Foundation 从固定模块 ZIP 解包，使用 `GOWORK=off`：

- `go mod tidy` 后 go.mod/go.sum 哈希不变。
- `go test -race -timeout 120s ./...` 通过。
- `go vet ./...` 通过。
- CI 同版本 `govulncheck@v1.6.0 ./...` 通过：0 个可达漏洞；扫描报告另有 1 个仅出现在依赖模块层面的未触达漏洞。

Blog 复制当前 API、生成合同到独立目录，只在该副本中升级 Foundation 至候选 `v0.5.0`；Asset 使用正式 `v0.4.0`：

- `GOWORK=off`，无 `replace`，独立 `GOMODCACHE`。Foundation 通过本地 Go 模块代理安装；仅该未发布模块在当前进程设置 `GONOSUMDB`，其他模块沿用校验。
- `go mod tidy`、全量 `go test -timeout 60s ./...`、`go vet ./...`、错误目录与生成物检查、HTTP operations 和 70 项 HTTP 合同检查通过。
- 实际导出 OpenAPI 与仓库合同一致。
- `go mod verify` 通过。
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` 构建 Blog API、healthcheck、bindingcheck、bootstrap 四个容器入口命令通过。
- 数据库破坏性测试保持显式 opt-in，未连接正在运行的开发库；真实数据库及浏览器行为沿用本 Work 已完成的 68 次请求验收。

初次解包的 go.mod/go.sum 为只读，已仅解除临时副本只读属性后执行 tidy。初版验收脚本曾复用 OpenAPI 输出文件名保存报告，造成误报；已分开路径并重新导出，正式合同没有变化。

## Blog 已修复的构建与部署配置

- Blog API Dockerfile 与独立 Identity provision 构建阶段使用 Go `1.25.13`，与模块最低版本一致。
- CI 中的 HTTP 合同生成器跟随 `go.mod` 选定版本，移除两处单独固定的 `@v0.4.1`。
- CI 与完整独立环境示例补齐 `BLOG_ASSET_TOKEN_URL`。
- Blog/Identity/Asset 的 PAT 运行参数已在 Blog 拥有的 Compose 模板与环境示例中接通。
- 使用本机已有的 Compose 官方解析库 `compose-go/v2 v2.14.0` 加载三种拓扑，每种分别校验 CI 环境及示例环境，合计 6 组通过；检查 JSON 权限登记、站点/audience 一致性及媒体回查地址。
- 本机没有 Docker CLI，未执行容器镜像构建或启动。上述 Linux 编译与 Compose 模型检查不能视作容器运行验收。

## 尚需交付的正式版本

1. 经用户明确授权后发布 Foundation `go/v0.5.0`，从公开 Go Proxy 和 SumDB 重新验证该版本。
2. 对 Blog 应用已预检的 [依赖升级补丁](foundation-dependency.patch)。当前主工作树的 go.mod/go.sum 保留正式 `v0.4.1`，避免直接写入尚不可下载的依赖；候选验证不代表这一缺口已在正式环境消失。
3. 完整模板重部署还需要 Identity 和 Asset 的 PAT 服务版本：已核实 Identity `v0.3.3` 无当前权限目录，Asset `v0.4.0` 无媒体授权回查。Blog 部署锁仍为 Identity `v0.2.2` / Asset `v0.3.0`，本轮未擅自换成未发布提交或声称已完成 Provider 发布候选验收。

Foundation 远端发布的待审批动作是：在上述固定提交创建并推送 `go/v0.5.0`，创建该 tag 的 GitHub Release；不据此推送 main 或发布其他包、服务。依据 [Foundation 发布策略](../../../../foundation/docs/release-policy.md)，用户需明确发布动作与目标 tag。

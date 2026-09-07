# Context

- Blog 拥有文章、评论、分类等业务 DTO 与错误语义；Foundation 只拥有 catalog/operation schema、生成器和 Runtime。
- Blog 媒体继续遵守 Asset MediaKey/SDK 合同，不在本站推导对象 Key 或 Backend URL。
- 本地验证通过 Workspace CLI 显式选择 Blog/Foundation/Identity/Asset checkout；不提交相邻源码路径。
- 本阶段不创建 tag、Release 或镜像。

- 2026-09-06 用户明确选择基础升级后的逐站人工复验顺序：Blog 第一站；其余站点次序尚未选择。复验是本 Blog 产品 Work 的后续完成门禁，不另建跨站验收 Work。

- 2026-09-07 用户授权真机试部署：SSH bt，Blog 使用 cg.yuelili.com（允许替换旧静态站），Account/Asset 使用同名子域。服务器使用 1Panel，部署目录沿用 /projects/yuelili.com；数据库按 core/content/vector 能力分组，独立于产品编排，每产品独立库与账号，持久文件绑定到各自 data/。
- 管理员必须由用户本人使用自己的账号显式认领；不得因邮箱问题未回复而创建或指定部署管理员，不得擅自将普通账户提升为全局管理员。

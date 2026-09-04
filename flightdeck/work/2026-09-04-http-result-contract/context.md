# Context

- Blog 拥有文章、评论、分类等业务 DTO 与错误语义；Foundation 只拥有 catalog/operation schema、生成器和 Runtime。
- Blog 媒体继续遵守 Asset MediaKey/SDK 合同，不在本站推导对象 Key 或 Backend URL。
- 本地验证通过 Workspace CLI 显式选择 Blog/Foundation/Identity/Asset checkout；不提交相邻源码路径。
- 本阶段不创建 tag、Release 或镜像。

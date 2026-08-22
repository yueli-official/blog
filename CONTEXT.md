# Blog

Blog 是面向公开阅读的文章发布与归档上下文；它用主分类、自由标签和有序系列组织文章，但不把这些概念混成一种通用 Taxonomy。

## Language

**Post（文章）**:
作者持续编辑并按发布状态对外展示的一篇独立长短文。
_Avoid_: 条目、内容对象、帖子集合

**Trashed Post（回收站文章）**:
作者已从常规管理视图移入回收站、但仍可恢复且保留原发布状态的 Post；只有回收站内的 Post 才能永久删除。
_Avoid_: Archived Post、已物理删除文章

**Category（分类）**:
由运营治理、支持多级父子关系的主要浏览结构；一篇文章可以直接归属多个 Category，父级浏览包含全部后代。
_Avoid_: Taxonomy、Tag、Series

**Tag（标签）**:
描述文章细节的扁平自由关键词；同义词和旧名称解析到 canonical Tag，但 Tag 永远没有父子层级。
_Avoid_: Taxonomy、Category、Tag Group

**Series（系列）**:
按明确顺序组织多篇文章的专题或连载；一篇文章至多属于一个 Series。
_Avoid_: Category、Collection、Tag

**Author（作者）**:
拥有文章写入权限的 Identity User；Blog 只持有写作资格和角色，公开姓名与头像仍属于 Identity。
_Avoid_: 本地用户、Creator、投稿者

**Subscriber（订阅者）**:
完成双重确认且当前仍订阅 Blog 邮件更新的邮箱受众；Subscriber 不等于 Identity User，也不代表已登录成员。
_Avoid_: User、Member、Follower

**Visitor Day（访客日）**:
在一个自然日内按浏览器派生标记去重的访问者计数；同一人在不同日期会产生多个访客日。
_Avoid_: User、Member、跨周期唯一访客

**Traffic Source（流量来源）**:
一次公开阅读进入 Blog 前的归因来源，只保留 `direct`、`internal` 或规范化外部主机名，不保存来源路径与查询参数。
_Avoid_: 完整 Referrer URL、用户来源、获客渠道

**Measured Views（实测浏览量）**:
Traffic 根据真实阅读事件累积的文章浏览量；编辑操作不得重写或伪造这一统计。
_Avoid_: 展示浏览量、手工浏览量

**SEO Metadata（搜索元数据）**:
文章针对搜索结果、规范地址和社交分享提供的可选覆盖值；未填写时使用 Post 的标题、摘要和标准地址。
_Avoid_: Post 标题、Post 摘要、必填 SEO 文案

**Friend Link（友链）**:
由站点管理员维护、按明确顺序展示在公开页脚的外部网站引用；每项只有展示内容和 HTTP(S) 地址，不代表认证、
合作背书或内容同步关系。
_Avoid_: Navigation Item、Partner、Subscriber、完整外站档案

**Contact Link（联系入口）**:
由站点管理员维护、按顺序展示在公开页脚的联系方法；每项只包含展示内容与可选 HTTP(S)/mailto 地址，可表达
邮箱、QQ群号或其他公开联系方式，不建立账号、成员或订阅关系。
_Avoid_: Subscriber、Identity User、Friend Link、客服工单

公开页脚的品牌描述使用 Site Description 作为唯一内容源；旧 `footerTagline` 只保留为兼容字段，由新 Web 保存时与
Site Description 同步，不再提供独立编辑入口。

**Asset Site Key（资源站点键）**:
Blog 在统一 Asset catalog 中选择 site/profile 规则的稳定产品键；它与部署、Traffic、Search 和 Privacy 使用的
Site Slug 分离。默认资源站点键为 `blog`，即使部署实例 slug 或 OIDC client 使用 `blog-main`。
_Avoid_: Site Slug、Asset Namespace、OIDC Client ID

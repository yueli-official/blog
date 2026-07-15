# Blog

Blog 是面向公开阅读的文章发布与归档上下文；它用主分类、自由标签和有序系列组织文章，但不把这些概念混成一种通用 Taxonomy。

## Language

**Post（文章）**:
作者持续编辑并按发布状态对外展示的一篇独立长短文。
_Avoid_: 条目、内容对象、帖子集合

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

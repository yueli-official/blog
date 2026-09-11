import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const readApp = (path) => readFileSync(resolve(root, path), "utf8");

test("new posts enter the full editor without a title modal", () => {
  const page = readApp("pages/manage/posts/index.vue");
  const dashboard = readApp("pages/manage/index.vue");
  const createDraft = readApp("composables/useCreatePostDraft.ts");

  assert.match(page, /useCreatePostDraft/);
  assert.match(dashboard, /useCreatePostDraft/);
  assert.match(createDraft, /title: "未命名文章"/);
  assert.match(
    createDraft,
    /navigateTo\(`\/manage\/posts\/\$\{response\.post\.slug\}`\)/,
  );
  assert.doesNotMatch(
    page,
    /v-model:open="showCreate"|创建并编辑|给文章起个标题/,
  );
});

test("article sharing uses the shared bounded action set", () => {
  const actions = readApp("components/PostActions.vue");
  assert.match(actions, /ContentShareActions/);
  assert.doesNotMatch(actions, /QQ 空间|qzone|ShareBar/);
});

test("archive history loads in resilient scroll-sized batches", () => {
  const archive = readApp("pages/archive.vue");

  assert.match(archive, /const size = 30/);
  assert.match(archive, /IntersectionObserver/);
  assert.match(archive, /autoLoadAnchor/);
  assert.match(archive, /loadError/);
  assert.match(archive, /finally/);
  assert.doesNotMatch(archive, />Archive</);
});

test("series has a dedicated management surface and navigation entry", () => {
  const layout = readApp("layouts/manage.vue");
  const page = readApp("pages/manage/series.vue");

  assert.match(layout, /to: "\/manage\/series"/);
  assert.match(layout, /label: "系列"/);
  assert.match(page, /CollectionPanel/);
  assert.match(page, /\/api\/v1\/series/);
  assert.match(page, /新建系列/);
  assert.match(page, /编辑系列/);
  assert.match(page, /label="Slug"/);
  assert.match(page, /form\.slug/);
});

test("taxonomy and series collections expose their public pages beside edit", () => {
  const taxonomy = readApp("components/TaxonomyManager.vue");
  const series = readApp("pages/manage/series.vue");

  assert.match(taxonomy, /`\/category\/\$\{tax\.slug\}`/);
  assert.match(taxonomy, /`\/tags\/\$\{tax\.slug\}`/);
  assert.match(taxonomy, /i-tabler-external-link/);
  assert.match(taxonomy, /AdminRowActions/);
  assert.match(series, /`\/series\/\$\{item\.slug\}`/);
  assert.match(series, /i-tabler-external-link/);
  assert.match(series, /AdminRowActions/);
});

test("post editor keeps the writing canvas and settings inspector in one workspace", () => {
  const editor = readApp("pages/manage/posts/[slug].vue");

  assert.match(editor, /data-blog-editor-workspace/);
  assert.match(editor, /data-blog-editor-inspector/);
  assert.match(editor, /aria-label="文章标题"/);
  assert.match(editor, /data-blog-inspector-content/);
  assert.match(editor, /data-blog-inspector-publishing/);
  assert.match(editor, /data-blog-inspector-seo/);
  assert.match(editor, /import \{ EditorInspector \} from "@yueli\/ui\/admin"/);
  assert.match(editor, /<EditorInspector/);
  assert.doesNotMatch(editor, /data-blog-editor-title-region/);
  assert.match(editor, /data-blog-inspector-content[\s\S]*label="路径标识"/);
  assert.match(editor, /data-blog-inspector-content[\s\S]*label="摘要"/);
  assert.match(editor, /沉浸式协作/);
  assert.match(editor, /#toolbar-actions/);
  assert.match(editor, /--content-editor-toolbar-top/);
  assert.match(editor, /<EditorCommandBar/);
  assert.match(editor, /settings-label="文章设置"/);
  assert.match(editor, /settingsOpen && !immersiveCollaboration/);
  assert.match(editor, /xl:pr-\[27rem\]/);
  assert.match(
    editor,
    /label: "内容", value: "content", icon: "i-tabler-photo"/,
  );
  assert.match(
    editor,
    /label: "发布", value: "publishing", icon: "i-tabler-calendar"/,
  );
  assert.match(editor, /label: "SEO", value: "seo", icon: "i-tabler-search"/);
  assert.match(editor, /variant="pill"/);
  assert.match(editor, /settingsSection/);
  const inspector = editor.slice(
    editor.indexOf("<EditorInspector"),
    editor.indexOf("</EditorInspector>"),
  );
  const inspectorFields = [
    ...inspector.matchAll(/<U(?:Input|Textarea|Select|SelectMenu)\b[\s\S]*?>/g),
  ].map((match) => match[0]);
  assert.ok(inspectorFields.length > 0);
  for (const field of inspectorFields)
    assert.match(field, /class="[^"]*w-full/);
  assert.match(editor, /data-blog-lifecycle-actions/);
  assert.match(editor, /title="文章设置"/);
  assert.match(editor, /label="摘要"/);
  assert.match(editor, /label="分类"/);
  assert.match(editor, /label="标签"/);
  assert.match(editor, /label="系列"/);
  assert.match(editor, /label="发布日期"/);
  assert.match(editor, /label: "转为草稿"/);
  assert.match(editor, /label: "移入回收站"/);
  assert.doesNotMatch(editor, /aria-label="推荐设置"/);
  assert.doesNotMatch(editor, /aria-label="搜索优化"/);
  assert.doesNotMatch(
    editor,
    /data-blog-cover-desktop|data-blog-cover-mobile|data-blog-editor-properties/,
  );
  assert.doesNotMatch(editor, /settingsSections|data-blog-settings-accordion/);
  assert.doesNotMatch(editor, /<USlideover/);
  assert.doesNotMatch(editor, /placeholder="新建系列"/);
  assert.doesNotMatch(editor, /label="归档"/);
  assert.doesNotMatch(editor, /:disabled="post\.status !== 'published'"/);
  assert.match(editor, /publishedAt \? \{ publishedAt \} : \{\}/);
  assert.match(editor, /publishedAtError/);
  assert.match(editor, /:max="maximumPublishedAt"/);
  assert.doesNotMatch(editor, /仅用于调整文章显示日期|暂不支持定时发布/);
  assert.doesNotMatch(editor, />当前状态</);
});

test("writer-facing quick and bulk actions no longer create new archived posts", () => {
  const quickEdit = readApp("components/BlogPostQuickEditModal.vue");
  const index = readApp("pages/manage/posts/index.vue");

  assert.doesNotMatch(quickEdit, /label: '归档', value: 'archived'/);
  assert.doesNotMatch(index, /label: "归档", value: "archive"/);
});

test("public taxonomy and series directories omit redundant explanatory copy", () => {
  const category = readApp("pages/category/index.vue");
  const tags = readApp("pages/tags/index.vue");
  const series = readApp("pages/series/index.vue");

  for (const page of [category, tags, series]) {
    assert.doesNotMatch(
      page,
      /按主题浏览全部文章|按热度或拼音浏览|按主题成体系连载/,
    );
  }
});

test("comment moderation emphasizes exceptions and collects low-frequency actions", () => {
  const comments = readApp("pages/manage/comments.vue");

  assert.match(comments, /CommentModerationCollection/);
  assert.match(comments, /lifecycleChange: changeLifecycle/);
  assert.match(comments, /label: "移入回收站"/);
  assert.match(comments, /label: "永久删除"/);
  assert.match(comments, /emptyTrash/);
  assert.match(comments, /status: meta\(comment\.status\)/);
  assert.match(comments, /approve: comment\.status === 2/);
  assert.match(comments, /actions: rowActionItems\(comment\)/);
  assert.match(comments, /:format-date="dateTime"/);
  assert.doesNotMatch(comments, />状态<\/span>/);
  assert.doesNotMatch(comments, /<CollectionPanel|<CollectionSortHeader/);
});

test("reader comments use the shared bounded thread and server-side ordering", () => {
  const comments = readApp("components/CommentSection.vue");
  assert.match(comments, /PublicCommentThread/);
  assert.match(comments, /sortOrder: order\.value/);
  assert.match(comments, /allow-anonymous/);
  assert.match(comments, /input-position="bottom"/);
  assert.doesNotMatch(comments, /border-t border-default/);
  assert.doesNotMatch(comments, /<CommentForm/);
});

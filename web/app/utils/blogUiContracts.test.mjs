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
  assert.match(series, /`\/series\/\$\{item\.slug\}`/);
  assert.match(series, /i-tabler-external-link/);
});

test("post settings use searchable organization controls without duplicate creation chrome", () => {
  const editor = readApp("pages/manage/posts/[slug].vue");

  assert.match(editor, /to="\/manage\/series"/);
  assert.doesNotMatch(editor, /placeholder="新建系列"/);
  assert.doesNotMatch(editor, /label="归档"/);
  assert.doesNotMatch(editor, /:disabled="post\.status !== 'published'"/);
  assert.match(editor, /publishedAt \? \{ publishedAt \} : \{\}/);
  assert.match(editor, /publishedAtError/);
  assert.match(editor, /:max="maximumPublishedAt"/);
  assert.match(editor, /暂不支持定时发布/);
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

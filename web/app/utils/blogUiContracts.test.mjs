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

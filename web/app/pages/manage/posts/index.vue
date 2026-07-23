<script setup lang="ts">
import { PageHeader } from "@yueli/ui/dashboard/pattern";
import {
  ManageTaxonomyChips,
  ManageEmpty,
  SkeletonList,
} from "@platform/manage/components";
import {
  CollectionLifecycleTabs,
  CollectionPanel,
  CollectionViewToggle,
} from "@yueli/ui/collection/pattern";
import {
  createCollectionRouteQueryCodec,
  createJsonCollectionQueryPolicy,
  type CollectionControl,
  type CollectionControlValue,
  type CollectionPanelMessages,
  type CollectionWorkflow,
} from "@yueli/ui/collection";
import { useVueCollectionWorkflow } from "@yueli/ui/collection/vue";
import { createVueRouterCollectionQuerySync } from "@yueli/ui/collection/vue-router";
import type {
  PostView,
  MyPosts,
  ListTaxonomies,
  AdminAuthorList,
} from "~/types";

interface BatchResult {
  changed: number;
  failures: Array<{ id: string; code: string; message: string }>;
  interrupted?: boolean;
  message?: string;
}

// Author console: my posts — status tabs + search + category/tag/author filters +
// list/grid view, server-side paginated, with always-on selection feeding a sticky
// footer (select-all + batch actions + pagination). Auth-gated; client-fetched
// (needs the author's BFF-injected Bearer).
definePageMeta({ layout: "manage", middleware: "auth" });
useSeoMeta({ title: "文章 · 控制台" });

const { call } = useApi();
const { isOwner, status: authorStatus, pending: mePending } = useMe();
const router = useRouter();
const mounted = ref(false);

const ALL = "__all__"; // USelect items can't carry an empty-string value
type PostStatus =
  "" | "published" | "draft" | "archived" | "issues" | "private";
type PostSort = "updated" | "title" | "published";
type PostDirection = "asc" | "desc";
type PostCollectionView = "list" | "grid";
type PostFlag = "all" | "pinned" | "featured";
interface PostCollectionQuery {
  q: string;
  status: PostStatus;
  page: number;
  size: number;
  sort: PostSort;
  direction: PostDirection;
  view: PostCollectionView;
  category: string;
  tag: string;
  author: string;
  flag: PostFlag;
}

const defaultQuery: PostCollectionQuery = {
  q: "",
  status: "",
  page: 1,
  size: 15,
  sort: "updated",
  direction: "desc",
  view: "list",
  category: ALL,
  tag: ALL,
  author: "mine",
  flag: "all",
};
const statuses = [
  "",
  "published",
  "draft",
  "archived",
  "issues",
  "private",
] as const;
const sorts = ["updated", "title", "published"] as const;
const pageSizes = [10, 15, 30, 50] as const;
const views = ["list", "grid"] as const;
const flags = ["all", "pinned", "featured"] as const;
const queryPolicy = createJsonCollectionQueryPolicy<PostCollectionQuery>();
const counts = ref<Record<string, number>>({});
const searchInput = ref("");
const sync = createVueRouterCollectionQuerySync({
  router,
  codec: createCollectionRouteQueryCodec({
    q: { kind: "string", default: defaultQuery.q, maxLength: 200 },
    status: { kind: "enum", values: statuses, default: defaultQuery.status },
    page: { kind: "positive-integer", default: defaultQuery.page },
    size: {
      kind: "positive-integer",
      values: pageSizes,
      default: defaultQuery.size,
    },
    sort: { kind: "enum", values: sorts, default: defaultQuery.sort },
    direction: {
      kind: "enum",
      values: ["asc", "desc"] as const,
      default: defaultQuery.direction,
    },
    view: { kind: "enum", values: views, default: defaultQuery.view },
    category: {
      kind: "string",
      default: defaultQuery.category,
      maxLength: 128,
    },
    tag: { kind: "string", default: defaultQuery.tag, maxLength: 128 },
    author: { kind: "string", default: defaultQuery.author, maxLength: 128 },
    flag: { kind: "enum", values: flags, default: defaultQuery.flag },
  }),
});
const {
  snapshot: collection,
  workflow,
  reload,
} = useVueCollectionWorkflow({
  initialQuery: defaultQuery,
  queryPolicy,
  keyOf: (post: PostView) => post.id,
  querySync: sync,
  dataQueryKey,
  load,
});

const query = computed(() => collection.value.query);
function updateQuery(patch: Partial<PostCollectionQuery>, resetPage = true) {
  workflow.setQuery({
    ...query.value,
    ...patch,
    ...(resetPage ? { page: 1 } : {}),
  });
}
const status = computed({
  get: () => query.value.status,
  set: (value: PostStatus) => updateQuery({ status: value }),
});
const q = computed(() => query.value.q);
const page = computed({
  get: () => query.value.page,
  set: (value: number) => updateQuery({ page: value }, false),
});
const size = computed({
  get: () => query.value.size,
  set: (value: number) => updateQuery({ size: value }),
});
const sort = computed({
  get: () => query.value.sort,
  set: (value: PostSort) => updateQuery({ sort: value }),
});
const direction = computed({
  get: () => query.value.direction,
  set: (value: PostDirection) => updateQuery({ direction: value }),
});
const viewMode = computed({
  get: () => query.value.view,
  set: (value: PostCollectionView) => updateQuery({ view: value }, false),
});
const categoryId = computed({
  get: () => query.value.category,
  set: (value: string) => updateQuery({ category: value }),
});
const tagId = computed({
  get: () => query.value.tag,
  set: (value: string) => updateQuery({ tag: value }),
});
const authorFilter = computed({
  get: () => query.value.author,
  set: (value: string) => updateQuery({ author: value }),
});
const flag = computed({
  get: () => query.value.flag,
  set: (value: PostFlag) => updateQuery({ flag: value }),
});

let searchTimer: ReturnType<typeof setTimeout> | undefined;
watch(searchInput, (value) => {
  if (searchTimer) clearTimeout(searchTimer);
  searchTimer = setTimeout(() => updateQuery({ q: value.trim() }), 300);
});

// ── filters: category + tag + (admin) author ──────────────────────────────────
async function load(
  nextQuery: Readonly<PostCollectionQuery>,
  activeWorkflow: CollectionWorkflow<PostView, string, PostCollectionQuery>,
) {
  const token = activeWorkflow.beginLoad();
  try {
    const requestedTaxonomyIds = [nextQuery.category, nextQuery.tag].filter(
      (value) => value !== ALL,
    );
    const result = await call<MyPosts>("/api/v1/posts/mine", {
      query: {
        status: nextQuery.status || undefined,
        q: nextQuery.q || undefined,
        taxonomyIds: requestedTaxonomyIds.length
          ? requestedTaxonomyIds
          : undefined,
        pinned: nextQuery.flag === "pinned" ? true : undefined,
        featured: nextQuery.flag === "featured" ? true : undefined,
        all: nextQuery.author === "all" ? true : undefined,
        authorId: ["mine", "all"].includes(nextQuery.author)
          ? undefined
          : nextQuery.author,
        sort: nextQuery.sort,
        direction: nextQuery.direction,
        page: nextQuery.page,
        size: nextQuery.size,
      },
    });
    const lastPage = Math.max(1, Math.ceil(result.total / nextQuery.size));
    if (nextQuery.page > lastPage) {
      activeWorkflow.setQuery({ ...nextQuery, page: lastPage });
      return;
    }
    if (
      activeWorkflow.resolveLoad(token, {
        items: result.items,
        total: result.total,
      })
    )
      counts.value = result.counts;
  } catch {
    activeWorkflow.rejectLoad(token, {
      key: "blog.posts.collection.load_failed",
    });
  }
}

function dataQueryKey(nextQuery: Readonly<PostCollectionQuery>) {
  const { view: _view, ...dataQuery } = nextQuery;
  return JSON.stringify(dataQuery);
}

searchInput.value = collection.value.query.q;
watch(q, (value) => {
  if (searchInput.value !== value) searchInput.value = value;
});
onMounted(() => {
  mounted.value = true;
});
onScopeDispose(() => {
  if (searchTimer) clearTimeout(searchTimer);
});

const pending = computed(
  () =>
    collection.value.loadState === "loading" ||
    collection.value.loadState === "refreshing",
);

// Any data-query transition clears product-owned batch feedback. The public
// workflow owns query/selection invariants; this page still owns its feedback.
watch(
  [
    status,
    q,
    categoryId,
    tagId,
    authorFilter,
    flag,
    sort,
    direction,
    size,
    page,
  ],
  () => {
    batchAction.value = undefined;
    batchResult.value = undefined;
  },
);

// taxonomy + author option sources (authors fetched only for admins)
const { data: taxData } = await useAsyncData(
  "manage-taxes",
  () => call<ListTaxonomies>("/api/v1/taxonomies"),
  { server: false, default: () => ({ items: [] }) },
);
const catOptions = computed(() => [
  { label: "全部分类", value: ALL },
  ...(taxData.value?.items ?? [])
    .filter((t) => t.taxonomy === "category")
    .map((t) => ({ label: t.name, value: t.id })),
]);
const tagOptions = computed(() => [
  { label: "全部标签", value: ALL },
  ...(taxData.value?.items ?? [])
    .filter((t) => t.taxonomy === "tag")
    .map((t) => ({ label: t.name, value: t.id })),
]);
const { data: authorData } = await useAsyncData(
  "manage-author-roster",
  () =>
    isOwner.value
      ? call<AdminAuthorList>("/api/v1/admin/authors")
      : Promise.resolve({ authors: [] }),
  { server: false, default: () => ({ authors: [] }) },
);
const authorOptions = computed(() => [
  { label: "我的文章", value: "mine" },
  { label: "全部作者", value: "all" },
  ...(authorData.value?.authors ?? []).map((a) => ({
    label: a.displayName || a.id.slice(0, 8),
    value: a.id,
  })),
]);
const authorName = (id: string) =>
  (authorData.value?.authors ?? []).find((a) => a.id === id)?.displayName ||
  id.slice(0, 8);
const showAuthor = computed(
  () => isOwner.value && authorFilter.value !== "mine",
);

const activeFilters = computed(() => [
  ...(categoryId.value !== ALL
    ? [
        {
          key: "category",
          label: `分类：${catOptions.value.find((item) => item.value === categoryId.value)?.label || categoryId.value}`,
        },
      ]
    : []),
  ...(tagId.value !== ALL
    ? [
        {
          key: "tag",
          label: `标签：${tagOptions.value.find((item) => item.value === tagId.value)?.label || tagId.value}`,
        },
      ]
    : []),
  ...(authorFilter.value !== "mine"
    ? [
        {
          key: "author",
          label: `作者：${authorOptions.value.find((item) => item.value === authorFilter.value)?.label || authorFilter.value}`,
        },
      ]
    : []),
  ...(flag.value !== "all"
    ? [{ key: "flag", label: flag.value === "pinned" ? "已置顶" : "已精选" }]
    : []),
]);
function clearActiveFilters() {
  categoryId.value = ALL;
  tagId.value = ALL;
  authorFilter.value = "mine";
  flag.value = "all";
}

const flagItems = [
  { label: "全部", value: "all" },
  { label: "置顶", value: "pinned" },
  { label: "精选", value: "featured" },
];
const sortItems = [
  { label: "最近更新", value: "updated" },
  { label: "标题", value: "title" },
  { label: "发布日期", value: "published" },
];
const collectionControls = computed<CollectionControl[]>(() => [
  {
    kind: "select",
    id: "category",
    label: "文章分类",
    value: categoryId.value,
    options: catOptions.value,
    icon: "i-tabler-folder",
    class: "w-36",
  },
  {
    kind: "select",
    id: "tag",
    label: "文章标签",
    value: tagId.value,
    options: tagOptions.value,
    icon: "i-tabler-hash",
    class: "w-36",
  },
  ...(isOwner.value
    ? [
        {
          kind: "select" as const,
          id: "author",
          label: "文章作者",
          value: authorFilter.value,
          options: authorOptions.value,
          icon: "i-tabler-user",
          class: "w-36",
        },
      ]
    : []),
  {
    kind: "select",
    id: "flag",
    label: "文章标记",
    value: flag.value,
    options: flagItems,
    icon: "i-tabler-flag",
    class: "w-28",
  },
  {
    kind: "select",
    id: "sort",
    label: "文章排序",
    value: sort.value,
    options: sortItems,
    icon: "i-tabler-arrows-sort",
    class: "w-32",
  },
  {
    kind: "direction",
    id: "direction",
    label: "排序方向",
    value: direction.value,
    ascendingLabel: "切换为倒序",
    descendingLabel: "切换为正序",
  },
]);
const collectionMessages: CollectionPanelMessages = {
  searchPlaceholder: "搜索标题 / slug…",
  searchAction: "搜索",
  filtersAction: "筛选",
  activeFilters: (count) => `筛选（${count}）`,
  clearFilters: "清除筛选",
  selectPage: "选择当前页文章",
  selectItem: (label) => `选择文章：${label}`,
  bulkRegion: "文章批量操作",
  selected: (count) => `已选择 ${count} 篇文章`,
  selectAllResults: "选择全部结果",
  clearSelection: "取消选择",
  emptyTitle: "没有匹配的文章",
  emptyDescription: "请调整搜索条件或筛选项后重试。",
  errorTitle: "文章加载失败",
  retry: "重新加载",
  showing: (first, last, count) => `显示 ${first}–${last}，共 ${count} 篇`,
  pageSize: "每页",
  pageSizeControl: "每页文章数量",
  pageSizeOption: (value) => `${value} 篇`,
};
function changeCollectionControl(id: string, value: CollectionControlValue) {
  if (id === "category") categoryId.value = String(value);
  if (id === "tag") tagId.value = String(value);
  if (id === "author") authorFilter.value = String(value);
  if (id === "flag" && flags.includes(value as PostFlag))
    flag.value = value as PostFlag;
  if (id === "sort" && sorts.includes(value as PostSort))
    sort.value = value as PostSort;
  if (id === "direction" && (value === "asc" || value === "desc"))
    direction.value = value;
}
function submitCollectionSearch(value: string) {
  if (searchTimer) clearTimeout(searchTimer);
  searchInput.value = value;
  updateQuery({ q: value.trim() });
}
const items = computed(() => collection.value.items);
const postKey = (post: PostView) => post.id;
const postLabel = (post: PostView) => post.title || "无标题";
function taxonomyChips(post: PostView) {
  return (post.taxonomies ?? []).map((item) => ({
    key: item.id,
    label: item.name,
    kind: item.taxonomy === "tag" ? ("tag" as const) : ("category" as const),
  }));
}
const total = computed(() => collection.value.total);
const tabs = computed(() => {
  const c = counts.value;
  const t = [
    { key: "", label: "全部", count: c.all ?? 0 },
    { key: "published", label: "已发布", count: c.published ?? 0 },
    { key: "draft", label: "草稿", count: c.draft ?? 0 },
    { key: "archived", label: "归档", count: c.archived ?? 0 },
    { key: "issues", label: "待完善", count: c.issues ?? 0 },
  ];
  if (c.private) t.push({ key: "private", label: "私密", count: c.private });
  return t;
});

// ── write gate: only an approved (active) author — or the owner — can write.
// Becoming an author is done from the public site header (申请成为作者), not here.
const canWrite = computed(
  () => isOwner.value || authorStatus.value === "active",
);

const quickEditTarget = ref<PostView>();
const showQuickEdit = ref(false);
function openQuickEdit(post: PostView) {
  quickEditTarget.value = post;
  showQuickEdit.value = true;
}
async function onQuickEditSaved(updated: PostView) {
  quickEditTarget.value = updated;
  await reload();
}
// ── selection + batch (always-on; driven from the sticky footer) ──────────────
const selectedIds = computed(() =>
  collection.value.selection.mode === "keys"
    ? collection.value.selection.keys
    : [],
);
const selectionCount = computed(() => collection.value.selection.count);
const isPageSelected = computed(() => collection.value.isPageSelected);
const isPageIndeterminate = computed(
  () => collection.value.isPageIndeterminate,
);
const isSelected = (id: string) => workflow.isSelected(id);
const toggleOne = (id: string, selected?: boolean) => {
  if (selected === undefined || selected !== workflow.isSelected(id))
    workflow.toggleKey(id);
};
const togglePage = (selected?: boolean | "indeterminate") =>
  workflow.togglePage(selected === true);
const clearSelection = () => workflow.clearSelection();
function replaceSelection(ids: readonly string[]) {
  workflow.clearSelection();
  for (const id of ids) workflow.toggleKey(id);
}
const batchAction = ref<string | undefined>(undefined);
const batchBusy = ref(false);
const batchResult = ref<BatchResult | undefined>(undefined);
const batchItems = [
  { label: "发布", value: "publish" },
  { label: "转草稿", value: "draft" },
  { label: "归档", value: "archive" },
  { label: "删除", value: "delete" },
];
const showBatchConfirm = ref(false);
async function runBatch() {
  if (!batchAction.value || !selectedIds.value.length) return;
  batchBusy.value = true;
  try {
    const res = await call<BatchResult>("/api/v1/posts/batch", {
      method: "POST",
      body: { ids: [...selectedIds.value], action: batchAction.value },
    });
    batchResult.value = res;
    const failedIds = new Set(res.failures.map((item) => item.id));
    replaceSelection(selectedIds.value.filter((id) => failedIds.has(id)));
    if (!res.failures.length) clearSelection();
    batchAction.value = undefined;
    showBatchConfirm.value = false;
    await reload();
  } catch (e: any) {
    batchResult.value = {
      changed: 0,
      failures: [],
      interrupted: true,
      message:
        e?.data?.message || "批量请求中断，已保留当前选择，请核对状态后重试。",
    };
    await reload();
  } finally {
    batchBusy.value = false;
  }
}
// delete is destructive → confirm first; everything else applies straight away.
function applyBatch() {
  if (!batchAction.value || !selectedIds.value.length) return;
  if (batchAction.value === "delete") {
    showBatchConfirm.value = true;
    return;
  }
  runBatch();
}

// gateLoading: still resolving whether the caller can write (useMe) → show a
// full skeleton, not a flash of "你还不是作者". showSkeleton: the list itself is
// (re)loading (filter / page change) → skeleton just the list, keep the toolbar.
const gateLoading = useMinimumLoading(
  computed(() => !mounted.value || mePending.value),
);
const showSkeleton = useMinimumLoading(computed(() => pending.value));

const showCreate = ref(false);
const title = ref("");
const creating = ref(false);
const createError = ref("");
watch(showCreate, (open) => {
  if (open) createError.value = "";
});
async function create() {
  if (!title.value.trim()) return;
  creating.value = true;
  createError.value = "";
  try {
    const res = await call<{ post: PostView }>("/api/v1/posts", {
      method: "POST",
      body: { title: title.value },
    });
    showCreate.value = false;
    title.value = "";
    navigateTo(`/manage/posts/${res.post.slug}`);
  } catch (e: any) {
    createError.value = e?.data?.message || "创建失败，请重试";
  } finally {
    creating.value = false;
  }
}

const firstFailedPost = computed(() => {
  const failedID = batchResult.value?.failures[0]?.id;
  return failedID
    ? items.value.find((item) => item.id === failedID)
    : undefined;
});
</script>

<template>
  <div>
    <PageHeader title="文章">
      <template #subtitle>管理你的全部文章</template>
      <template #actions>
        <UButton
          v-if="canWrite"
          icon="i-tabler-plus"
          label="写新文章"
          @click="
            () => {
              showCreate = true;
            }
          "
        />
        <UButton
          v-else-if="authorStatus === 'pending'"
          icon="i-tabler-clock"
          label="作者申请审核中"
          color="neutral"
          variant="subtle"
          disabled
        />
      </template>
    </PageHeader>

    <SkeletonList v-if="gateLoading" :rows="8" />

    <ManageEmpty
      v-else-if="!canWrite && !items.length"
      :icon="
        authorStatus === 'pending' ? 'i-tabler-clock' : 'i-tabler-user-edit'
      "
      :text="
        authorStatus === 'pending'
          ? '作者申请审核中,通过后即可写文章'
          : '你还不是作者 —— 在站点右上角头像菜单「申请成为作者」'
      "
    />

    <template v-else-if="canWrite || items.length">
      <CollectionLifecycleTabs v-model="status" :items="tabs" class="mb-4" />

      <UAlert
        v-if="batchResult"
        class="mb-3"
        :color="
          batchResult.interrupted || batchResult.failures.length
            ? 'warning'
            : 'success'
        "
        variant="subtle"
        :icon="
          batchResult.interrupted || batchResult.failures.length
            ? 'i-tabler-alert-triangle'
            : 'i-tabler-circle-check'
        "
        :title="
          batchResult.interrupted
            ? '批量请求中断'
            : `已处理 ${batchResult.changed} 篇文章`
        "
        :description="
          batchResult.interrupted
            ? batchResult.message
            : batchResult.failures.length
              ? `${batchResult.failures.length} 篇文章仍需完善。`
              : '批量操作已完成。'
        "
        :actions="
          firstFailedPost
            ? [
                {
                  label: '去完善',
                  to: `/manage/posts/${firstFailedPost.slug}`,
                  color: 'warning',
                  variant: 'link',
                },
              ]
            : undefined
        "
        close
        @update:open="batchResult = undefined"
      />

      <CollectionPanel
        v-model:search="searchInput"
        :items="items"
        :item-key="postKey"
        :item-label="postLabel"
        :controls="collectionControls"
        :messages="collectionMessages"
        :state="collection.issue ? 'error' : showSkeleton ? 'loading' : 'ready'"
        :error-message="collection.issue?.key"
        :total="total"
        :page="page"
        :page-size="size"
        :page-sizes="pageSizes"
        :active-filter-count="activeFilters.length"
        selectable
        :selection-count="selectionCount"
        :page-selected="isPageSelected"
        :page-indeterminate="isPageIndeterminate"
        :is-selected="isSelected"
        :layout="viewMode === 'grid' ? 'grid' : 'rows'"
        label="文章列表"
        @search="submitCollectionSearch"
        @control-change="changeCollectionControl"
        @clear-filters="clearActiveFilters"
        @retry="reload"
        @toggle-page="togglePage"
        @toggle-item="toggleOne"
        @clear-selection="clearSelection"
        @page-change="
          (value) => {
            page = value;
          }
        "
        @page-size-change="
          (value) => {
            size = value;
          }
        "
      >
        <template #view>
          <CollectionViewToggle
            v-model="viewMode"
            :items="[
              { key: 'list', label: '列表视图', icon: 'i-tabler-list' },
              { key: 'grid', label: '网格视图', icon: 'i-tabler-layout-grid' },
            ]"
          />
        </template>

        <template #columns>
          <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3">
            <span>名称、分类与标签</span>
            <span class="w-20 text-right">操作</span>
          </div>
        </template>

        <template #bulk-actions>
          <USelect
            v-model="batchAction"
            :items="batchItems"
            placeholder="批量操作"
            size="xs"
            class="w-28"
          />
          <UButton
            size="xs"
            color="primary"
            variant="soft"
            :disabled="!batchAction"
            :loading="batchBusy"
            @click="applyBatch"
            >应用</UButton
          >
        </template>

        <template #item="{ item: p }">
          <div
            v-if="viewMode === 'list'"
            class="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-3"
          >
            <div class="flex min-w-0 items-center gap-3">
              <div
                class="size-12 shrink-0 overflow-hidden rounded-lg bg-elevated"
              >
                <img
                  v-if="p.coverUrl"
                  :src="p.coverUrl"
                  :alt="p.title"
                  class="size-full object-cover"
                />
                <div
                  v-else
                  class="blog-cover-placeholder blog-cover-placeholder--tiny grid size-full place-items-center bg-gradient-to-br from-primary/10 to-transparent"
                >
                  <UIcon
                    name="i-tabler-feather"
                    class="blog-cover-icon size-5 text-primary/30"
                  />
                </div>
              </div>
              <div class="min-w-0">
                <p class="truncate text-sm font-medium text-highlighted">
                  {{ p.title || "(无标题)" }}
                </p>
                <div
                  class="mt-0.5 flex min-w-0 items-center gap-2 text-xs text-muted"
                >
                  <span
                    v-if="showAuthor"
                    class="inline-flex items-center gap-1 text-primary"
                    ><UIcon name="i-tabler-user" class="size-3" />{{
                      authorName(p.authorId)
                    }}</span
                  >
                  <span v-if="showAuthor" class="text-dimmed">·</span>
                  <span class="truncate font-mono">{{ p.slug }}</span>
                  <span class="text-dimmed">·</span>
                  <ClientOnly
                    ><span class="shrink-0">{{
                      rel(p.publishedAt || p.createdAt)
                    }}</span
                    ><template #fallback>…</template></ClientOnly
                  >
                </div>
                <ManageTaxonomyChips class="mt-1.5" :items="taxonomyChips(p)" />
              </div>
            </div>
            <div class="flex w-20 justify-end gap-1">
              <UTooltip text="快速编辑">
                <UButton
                  icon="i-tabler-pencil"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  square
                  :aria-label="`快速编辑文章：${p.title || '无标题'}`"
                  @click="openQuickEdit(p)"
                />
              </UTooltip>
              <UTooltip text="编辑文章">
                <UButton
                  :to="`/manage/posts/${p.slug}`"
                  icon="i-tabler-file-pencil"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  square
                  :aria-label="`编辑文章：${p.title || '无标题'}`"
                />
              </UTooltip>
            </div>
          </div>

          <div
            v-else
            class="group -m-4 cursor-pointer overflow-hidden rounded-lg"
            @click="navigateTo(`/manage/posts/${p.slug}`)"
          >
            <div class="relative aspect-[16/10] overflow-hidden bg-elevated">
              <img
                v-if="p.coverUrl"
                :src="p.coverUrl"
                :alt="p.title"
                class="size-full object-cover"
              />
              <div
                v-else
                class="blog-cover-placeholder grid size-full place-items-center bg-gradient-to-br from-primary/10 to-transparent"
              >
                <UIcon
                  name="i-tabler-feather"
                  class="blog-cover-icon size-7 text-primary/30"
                />
              </div>
              <UTooltip text="快速编辑">
                <UButton
                  icon="i-tabler-pencil"
                  color="neutral"
                  variant="solid"
                  size="xs"
                  square
                  class="absolute right-2 top-2"
                  :aria-label="`快速编辑文章：${p.title || '无标题'}`"
                  @click.stop="openQuickEdit(p)"
                />
              </UTooltip>
            </div>
            <div class="min-w-0 p-3">
              <h3 class="truncate text-sm font-medium text-highlighted">
                {{ p.title || "(无标题)" }}
              </h3>
              <p class="mt-1 truncate text-xs text-dimmed">
                <ClientOnly
                  >{{ rel(p.publishedAt || p.createdAt)
                  }}<template #fallback>…</template></ClientOnly
                >
              </p>
              <ManageTaxonomyChips class="mt-2" :items="taxonomyChips(p)" />
            </div>
          </div>
        </template>
      </CollectionPanel>
    </template>

    <UModal
      v-model:open="showBatchConfirm"
      title="删除文章"
      :description="`确定删除选中的 ${selectedIds.length} 篇文章?此操作不可撤销。`"
      :ui="{ footer: 'justify-end' }"
    >
      <template #footer>
        <UButton
          label="取消"
          color="neutral"
          variant="outline"
          @click="
            () => {
              showBatchConfirm = false;
            }
          "
        />
        <UButton
          label="删除"
          icon="i-tabler-trash"
          color="error"
          :loading="batchBusy"
          @click="runBatch"
        />
      </template>
    </UModal>

    <BlogPostQuickEditModal
      v-model:open="showQuickEdit"
      :post="quickEditTarget"
      @saved="onQuickEditSaved"
    />

    <UModal
      v-model:open="showCreate"
      title="写新文章"
      :ui="{ footer: 'justify-end' }"
    >
      <template #body>
        <div class="space-y-4">
          <UAlert
            v-if="createError"
            color="error"
            variant="subtle"
            icon="i-tabler-alert-circle"
            title="创建失败"
            :description="createError"
            role="alert"
          />
          <UFormField label="标题" required>
            <UInput
              v-model="title"
              placeholder="给文章起个标题"
              class="w-full"
              autofocus
              @keyup.enter="create"
            />
          </UFormField>
        </div>
      </template>
      <template #footer="{ close }">
        <UButton
          label="取消"
          color="neutral"
          variant="outline"
          @click="close"
        />
        <UButton
          label="创建并编辑"
          icon="i-tabler-arrow-right"
          trailing
          :loading="creating"
          :disabled="!title.trim()"
          @click="create"
        />
      </template>
    </UModal>
  </div>
</template>

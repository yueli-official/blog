<script setup lang="ts">
import { PageHeader } from "@yueli/ui/dashboard/pattern";
import { useActionFeedback } from "@yueli/ui/feedback";
import { ActionFeedbackButton } from "@yueli/ui/feedback/pattern";
import {
  createCollectionRouteQueryCodec,
  createJsonCollectionQueryPolicy,
  type CollectionControl,
  type CollectionControlValue,
  type CollectionPanelMessages,
  type CollectionPanelState,
  type CollectionWorkflow,
} from "@yueli/ui/collection";
import { useVueCollectionWorkflow } from "@yueli/ui/collection/vue";
import { createVueRouterCollectionQuerySync } from "@yueli/ui/collection/vue-router";
import { CollectionPanel } from "@yueli/ui/collection/pattern";
import type { ListTaxonomies, TaxonomyView } from "~/types";

const props = defineProps<{ kind: "category" | "tag" }>();
const isCategory = computed(() => props.kind === "category");
const label = computed(() => (isCategory.value ? "分类" : "标签"));
const { isOwner } = useMe();
const { call } = useApi();
const router = useRouter();
const ROOT = "__root__";

type TaxonomySort = "postCount" | "name" | "slug";
type Direction = "asc" | "desc";
interface TaxonomyQuery {
  q: string;
  sort: TaxonomySort;
  direction: Direction;
  page: number;
  size: number;
}
interface TaxonomyRow {
  tax: TaxonomyView;
  depth: number;
}
const sorts = ["postCount", "name", "slug"] as const;
const directions = ["asc", "desc"] as const;
const pageSizes = [15, 30, 60, 100] as const;
const defaultQuery: TaxonomyQuery = {
  q: "",
  sort: props.kind === "category" ? "name" : "postCount",
  direction: props.kind === "category" ? "asc" : "desc",
  page: 1,
  size: 30,
};
const all = ref<TaxonomyView[]>([]);
function treeRows(
  items: readonly TaxonomyView[],
  query: Readonly<TaxonomyQuery>,
): TaxonomyRow[] {
  if (!isCategory.value || query.q.trim())
    return items.map((tax) => ({ tax, depth: 0 }));
  const comparator = (a: TaxonomyView, b: TaxonomyView) => {
    const multiplier = query.direction === "asc" ? 1 : -1;
    if (query.sort === "name")
      return a.name.localeCompare(b.name, "zh-CN") * multiplier;
    if (query.sort === "slug") return a.slug.localeCompare(b.slug) * multiplier;
    return (
      ((a.postCount || 0) - (b.postCount || 0) ||
        a.name.localeCompare(b.name, "zh-CN")) * multiplier
    );
  };
  const byParent = new Map<string, TaxonomyView[]>();
  for (const item of items) {
    const parent = item.parentId || "";
    if (!byParent.has(parent)) byParent.set(parent, []);
    byParent.get(parent)!.push(item);
  }
  const result: { tax: TaxonomyView; depth: number }[] = [];
  const walk = (parent: string, depth: number) => {
    for (const item of [...(byParent.get(parent) || [])].sort(comparator)) {
      result.push({ tax: item, depth });
      walk(item.id, depth + 1);
    }
  };
  walk("", 0);
  return result;
}
async function loadTaxonomies(
  query: Readonly<TaxonomyQuery>,
  workflow: CollectionWorkflow<TaxonomyRow, string, TaxonomyQuery>,
) {
  const token = workflow.beginLoad();
  const flat = !isCategory.value || Boolean(query.q.trim());
  try {
    const data = await call<ListTaxonomies>("/api/v1/taxonomies", {
      query: {
        taxonomy: props.kind,
        q: query.q.trim() || undefined,
        sort: query.sort,
        direction: query.direction,
        page: flat ? query.page : undefined,
        size: flat ? query.size : undefined,
      },
    });
    all.value = data.items ?? [];
    const rows = treeRows(all.value, query);
    const total = data.total ?? rows.length;
    const lastPage = flat ? Math.max(1, Math.ceil(total / query.size)) : 1;
    if (query.page > lastPage) {
      workflow.setQuery({ ...query, page: lastPage });
      return;
    }
    workflow.resolveLoad(token, { items: rows, total });
  } catch {
    workflow.rejectLoad(token, { key: "blog.taxonomy.collection.load_failed" });
  }
}
const {
  snapshot: collection,
  workflow,
  reload: refresh,
} = useVueCollectionWorkflow({
  initialQuery: defaultQuery,
  queryPolicy: createJsonCollectionQueryPolicy<TaxonomyQuery>(),
  keyOf: (row: TaxonomyRow) => row.tax.id,
  querySync: createVueRouterCollectionQuerySync({
    router,
    codec: createCollectionRouteQueryCodec({
      q: { kind: "string", default: "" },
      sort: { kind: "enum", values: sorts, default: defaultQuery.sort },
      direction: {
        kind: "enum",
        values: directions,
        default: defaultQuery.direction,
      },
      page: { kind: "positive-integer", default: 1 },
      size: { kind: "positive-integer", values: pageSizes, default: 30 },
    }),
  }),
  dataQueryKey: (query) => JSON.stringify(query),
  load: loadTaxonomies,
});
const query = computed(() => collection.value.query);
function updateQuery(patch: Partial<TaxonomyQuery>, resetPage = true) {
  workflow.setQuery({
    ...query.value,
    ...patch,
    ...(resetPage ? { page: 1 } : {}),
  });
}
const q = computed(() => query.value.q);
const searchInput = ref(q.value);
const sort = computed({
  get: () => query.value.sort,
  set: (value: TaxonomySort) => updateQuery({ sort: value }),
});
const direction = computed({
  get: () => query.value.direction,
  set: (value: Direction) => updateQuery({ direction: value }),
});
const page = computed({
  get: () => query.value.page,
  set: (value: number) => updateQuery({ page: value }, false),
});
const size = computed({
  get: () => query.value.size,
  set: (value: number) => updateQuery({ size: value }),
});
let searchTimer: ReturnType<typeof setTimeout> | undefined;
watch(searchInput, (value) => {
  if (searchTimer) clearTimeout(searchTimer);
  searchTimer = setTimeout(() => updateQuery({ q: value.trim() }), 300);
});
watch(q, (value) => {
  if (searchInput.value !== value) searchInput.value = value;
});
onScopeDispose(() => {
  if (searchTimer) clearTimeout(searchTimer);
});
function submitSearch(value: string) {
  if (searchTimer) clearTimeout(searchTimer);
  updateQuery({ q: value.trim() });
}
const flat = computed(() => !isCategory.value || Boolean(q.value.trim()));
const rows = computed(() => collection.value.items);
const sortItems = [
  { label: "按文章数", value: "postCount" },
  { label: "按名称", value: "name" },
  { label: "按 Slug", value: "slug" },
];
const controls = computed<CollectionControl[]>(() => [
  {
    kind: "select",
    id: "sort",
    label: `${label.value}排序`,
    value: sort.value,
    options: sortItems,
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
function changeControl(id: string, value: CollectionControlValue) {
  if (id === "sort" && typeof value === "string")
    sort.value = value as TaxonomySort;
  if (id === "direction" && (value === "asc" || value === "desc"))
    direction.value = value;
}
const messages = computed<CollectionPanelMessages>(() => ({
  searchPlaceholder: `搜索${label.value}名称、slug 或描述…`,
  searchAction: "搜索",
  filtersAction: "筛选",
  activeFilters: (count) => `筛选（${count}）`,
  clearFilters: "清除筛选",
  selectPage: "选择当前页",
  selectItem: (name) => `选择：${name}`,
  bulkRegion: "批量操作",
  selected: (count) => `已选择 ${count} 个`,
  selectAllResults: "选择全部结果",
  clearSelection: "取消选择",
  emptyTitle: `还没有${label.value}`,
  emptyDescription: `没有匹配的${label.value}。`,
  errorTitle: "加载失败",
  retry: "重新加载",
  showing: (first, last, count) => `显示 ${first}–${last}，共 ${count} 个`,
  pageSize: "每页",
  pageSizeControl: "每页数量",
  pageSizeOption: (value) => `${value} 个`,
}));
const panelState = computed<CollectionPanelState>(() =>
  collection.value.issue
    ? "error"
    : ["idle", "loading", "refreshing"].includes(collection.value.loadState)
      ? "loading"
      : "ready",
);
const rowKey = (row: TaxonomyRow) => row.tax.id;
const rowLabel = (row: TaxonomyRow) => row.tax.name;

const panel = ref(false);
const current = ref<TaxonomyView | null>(null);
const mergeTarget = ref("");
const operationBusy = ref<"" | "merge" | "delete">("");
const operationError = ref("");
const saveError = ref("");
const confirmingDelete = ref(false);
const form = reactive({ name: "", slug: "", description: "", parentId: ROOT });
const slugTouched = ref(false);
const {
  status: saveStatus,
  pending: markSaving,
  success: markSaved,
  reset: resetSave,
} = useActionFeedback();
const options = ref<TaxonomyView[]>([]);
const optionsLoading = ref(false);
watch(
  [all, flat],
  ([items, isFlat]) => {
    if (!isFlat) options.value = [...items];
  },
  { immediate: true },
);
const optionSource = computed(() =>
  options.value.length ? options.value : all.value,
);
const parentItems = computed(() => [
  { label: "顶级分类", value: ROOT },
  ...optionSource.value
    .filter((item) => item.id !== current.value?.id)
    .map((item) => ({ label: item.name, value: item.id })),
]);
const mergeTargets = computed(() =>
  optionSource.value
    .filter((item) => item.id !== current.value?.id)
    .map((item) => ({
      label: `${item.name} · ${item.postCount || 0} 篇文章`,
      value: item.id,
    })),
);

function clientSlug(value: string) {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}

watch(
  () => form.name,
  (name) => {
    if (!current.value && !slugTouched.value) form.slug = clientSlug(name);
  },
);

function resetPanelState() {
  resetSave();
  mergeTarget.value = "";
  operationBusy.value = "";
  operationError.value = "";
  saveError.value = "";
  confirmingDelete.value = false;
}

async function ensureOptions() {
  if (optionsLoading.value || (!flat.value && options.value.length)) return;
  optionsLoading.value = true;
  try {
    const response = await call<ListTaxonomies>("/api/v1/taxonomies", {
      query: { taxonomy: props.kind, sort: "name", direction: "asc" },
    });
    options.value = response.items;
  } finally {
    optionsLoading.value = false;
  }
}

function openCreate() {
  resetPanelState();
  current.value = null;
  Object.assign(form, { name: "", slug: "", description: "", parentId: ROOT });
  slugTouched.value = false;
  panel.value = true;
  void ensureOptions();
}

function openEdit(item: TaxonomyView) {
  resetPanelState();
  current.value = item;
  Object.assign(form, {
    name: item.name,
    slug: item.slug,
    description: item.description || "",
    parentId: item.parentId || ROOT,
  });
  slugTouched.value = true;
  panel.value = true;
  void ensureOptions();
}

async function save() {
  if (!form.name.trim()) return;
  markSaving();
  saveError.value = "";
  try {
    const body: Record<string, unknown> = {
      name: form.name.trim(),
      slug: form.slug.trim() || undefined,
      description: form.description.trim(),
    };
    if (!current.value) body.taxonomy = props.kind;
    if (isCategory.value)
      body.parentId = form.parentId === ROOT ? "" : form.parentId;

    const response = current.value
      ? await call<{ taxonomy: TaxonomyView }>(
          `/api/v1/taxonomies/${current.value.id}`,
          { method: "PATCH", body },
        )
      : await call<{ taxonomy: TaxonomyView }>("/api/v1/taxonomies", {
          method: "POST",
          body,
        });
    current.value = response.taxonomy;
    Object.assign(form, {
      name: response.taxonomy.name,
      slug: response.taxonomy.slug,
      description: response.taxonomy.description || "",
      parentId: response.taxonomy.parentId || ROOT,
    });
    options.value = [];
    await refresh();
    markSaved();
  } catch (err: any) {
    resetSave();
    saveError.value =
      err?.data?.message || "保存失败；中文名请确认已手动填写 slug";
  }
}

async function mergeCurrent() {
  if (!current.value || !mergeTarget.value || operationBusy.value) return;
  operationBusy.value = "merge";
  operationError.value = "";
  try {
    await call(`/api/v1/taxonomies/${current.value.id}/merge`, {
      method: "POST",
      body: { targetId: mergeTarget.value },
    });
    panel.value = false;
    options.value = [];
    await refresh();
  } catch (err: any) {
    operationError.value = err?.data?.message || "合并失败，请重试";
  } finally {
    operationBusy.value = "";
  }
}

async function deleteCurrent() {
  if (!current.value || operationBusy.value) return;
  operationBusy.value = "delete";
  operationError.value = "";
  try {
    await call(`/api/v1/taxonomies/${current.value.id}`, { method: "DELETE" });
    panel.value = false;
    options.value = [];
    await refresh();
  } catch (err: any) {
    operationError.value = err?.data?.message || "可能仍有子分类";
  } finally {
    operationBusy.value = "";
  }
}

function armDelete() {
  confirmingDelete.value = true;
}
function cancelDelete() {
  confirmingDelete.value = false;
}
</script>

<template>
  <div class="space-y-5">
    <PageHeader :title="label">
      <template #subtitle>{{
        isCategory
          ? "维护文章分类层级和公开路径。"
          : "维护文章标签，合并重复词并保持检索清晰。"
      }}</template>
      <template #actions>
        <UButton
          v-if="isOwner"
          icon="i-tabler-plus"
          :label="`新建${label}`"
          @click="openCreate"
        />
      </template>
    </PageHeader>

    <div
      v-if="!isOwner"
      class="blog-manage-panel rounded-2xl border-dashed py-16 text-center text-muted"
    >
      <UIcon name="i-tabler-lock" class="mx-auto size-8" />
      <p class="mt-2 text-sm">仅站长可治理全站{{ label }}。</p>
    </div>

    <CollectionPanel
      v-else
      v-model:search="searchInput"
      :label="`${label}列表`"
      :items="rows"
      :item-key="rowKey"
      :item-label="rowLabel"
      :controls="controls"
      :messages="messages"
      :state="panelState"
      :error-message="collection.issue ? '无法加载数据，请稍后重试。' : ''"
      :total="collection.total"
      :page="page"
      :page-size="size"
      :page-sizes="pageSizes"
      @search="submitSearch"
      @control-change="changeControl"
      @retry="refresh"
      @page-change="page = $event"
      @page-size-change="size = $event"
    >
      <template #columns>
        <div
          class="grid grid-cols-[minmax(0,1fr)_7rem_2rem] gap-3 px-3 sm:px-4"
        >
          <span>名称</span><span class="text-right">文章数</span
          ><span class="sr-only">操作</span>
        </div>
      </template>
      <template #item="{ item: row }">
        <button
          type="button"
          class="blog-manage-row group grid w-full grid-cols-[minmax(0,1fr)_2.75rem] items-center gap-3 px-3 py-3 text-left focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-primary sm:px-4 lg:grid-cols-[minmax(14rem,1fr)_8rem_2.75rem]"
          @click="openEdit(row.tax)"
        >
          <span
            class="flex min-w-0 items-center gap-3"
            :style="
              isCategory
                ? { paddingLeft: `${Math.min(row.depth, 4) * 22}px` }
                : undefined
            "
          >
            <UIcon
              v-if="isCategory && row.depth > 0"
              name="i-tabler-corner-down-right"
              class="size-4 shrink-0 text-dimmed"
            />
            <span
              class="grid size-10 shrink-0 place-items-center rounded-lg bg-elevated text-dimmed transition group-hover:bg-primary/10 group-hover:text-primary"
            >
              <UIcon
                :name="isCategory ? 'i-tabler-folder' : 'i-tabler-hash'"
                class="size-4"
              />
            </span>
            <span class="min-w-0">
              <span class="line-clamp-1 text-sm font-medium text-highlighted">{{
                row.tax.name
              }}</span>
              <span
                class="mt-0.5 block line-clamp-1 font-mono text-xs text-muted"
                >/{{ row.tax.slug }}</span
              >
              <span
                v-if="row.tax.description"
                class="mt-1 block line-clamp-1 text-xs text-muted"
                >{{ row.tax.description }}</span
              >
            </span>
          </span>
          <span
            class="col-start-1 pl-13 text-xs text-muted lg:col-start-auto lg:pl-0 lg:text-right"
          >
            <span class="font-semibold text-highlighted">{{
              row.tax.postCount || 0
            }}</span>
            篇文章
          </span>
          <span
            class="row-start-1 col-start-2 grid size-11 place-items-center text-muted lg:row-auto lg:col-start-auto"
            aria-hidden="true"
          >
            <UIcon name="i-tabler-pencil" class="size-4" />
          </span>
        </button>
      </template>
    </CollectionPanel>

    <USlideover
      v-model:open="panel"
      :title="current ? `编辑${label}` : `新建${label}`"
    >
      <template #body>
        <div class="space-y-5">
          <div class="space-y-4">
            <UAlert
              v-if="saveError"
              color="error"
              variant="subtle"
              icon="i-tabler-alert-circle"
              title="保存失败"
              :description="saveError"
            />
            <UFormField label="名称" required>
              <UInput
                v-model="form.name"
                :placeholder="`${label}名称`"
                class="w-full"
                autofocus
              />
            </UFormField>
            <UFormField
              label="Slug"
              :help="
                current
                  ? '改动会影响公开链接。'
                  : '英文名自动生成；中文名请手动填写。'
              "
            >
              <UInput
                v-model="form.slug"
                placeholder="例如 tech"
                class="w-full"
                @input="slugTouched = true"
              />
            </UFormField>
            <UFormField v-if="isCategory" label="父分类">
              <USelectMenu
                v-model="form.parentId"
                :items="parentItems"
                value-key="value"
                placeholder="选择父分类"
                :search-input="{ placeholder: '搜索分类…' }"
                :loading="optionsLoading"
                class="w-full"
              />
            </UFormField>
            <UFormField label="描述">
              <UTextarea v-model="form.description" :rows="3" class="w-full" />
            </UFormField>
            <ActionFeedbackButton
              block
              :status="saveStatus"
              idle-label="保存"
              pending-label="保存中"
              success-label="已保存"
              :disabled="!form.name.trim()"
              @click="save"
            />
          </div>

          <template v-if="current">
            <USeparator />
            <section
              aria-labelledby="blog-taxonomy-management"
              class="space-y-4"
            >
              <div>
                <h3
                  id="blog-taxonomy-management"
                  class="text-sm font-medium text-highlighted"
                >
                  管理{{ label }}
                </h3>
                <p class="mt-1 text-xs text-muted">
                  合并会迁移文章关联；删除有子分类时会被后端拒绝。
                </p>
              </div>
              <UAlert
                v-if="operationError"
                color="error"
                variant="subtle"
                icon="i-tabler-alert-circle"
                title="操作失败"
                :description="operationError"
              />
              <div class="rounded-xl border border-default bg-elevated/35 p-3">
                <UFormField
                  :label="`合并到其他${label}`"
                  :help="`当前关联 ${current.postCount || 0} 篇文章`"
                >
                  <div class="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]">
                    <USelectMenu
                      v-model="mergeTarget"
                      :items="mergeTargets"
                      value-key="value"
                      placeholder="选择目标…"
                      :search-input="{ placeholder: `搜索目标${label}…` }"
                      :loading="optionsLoading"
                      class="w-full"
                    />
                    <UButton
                      label="合并"
                      icon="i-tabler-arrows-join"
                      color="warning"
                      variant="soft"
                      :disabled="!mergeTarget"
                      :loading="operationBusy === 'merge'"
                      @click="mergeCurrent"
                    />
                  </div>
                </UFormField>
              </div>
              <div class="rounded-xl border border-error/25 bg-error/5 p-3">
                <div
                  v-if="!confirmingDelete"
                  class="flex items-center justify-between gap-3"
                >
                  <div class="min-w-0">
                    <p class="text-sm font-medium text-highlighted">
                      删除{{ label }}
                    </p>
                    <p class="mt-1 text-xs text-muted">
                      文章会解除关联；有子分类时后端会拒绝。
                    </p>
                  </div>
                  <UButton
                    label="删除"
                    icon="i-tabler-trash"
                    color="error"
                    variant="soft"
                    size="sm"
                    @click="armDelete"
                  />
                </div>
                <div v-else>
                  <p class="text-sm text-highlighted">
                    确定删除「{{ current.name }}」？此操作不可恢复。
                  </p>
                  <div class="mt-3 flex justify-end gap-2">
                    <UButton
                      label="取消"
                      color="neutral"
                      variant="ghost"
                      size="sm"
                      @click="cancelDelete"
                    />
                    <UButton
                      label="确认删除"
                      icon="i-tabler-trash"
                      color="error"
                      size="sm"
                      :loading="operationBusy === 'delete'"
                      @click="deleteCurrent"
                    />
                  </div>
                </div>
              </div>
            </section>
          </template>
        </div>
      </template>
    </USlideover>
  </div>
</template>

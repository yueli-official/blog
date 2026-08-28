<script setup lang="ts">
import {
  createCollectionRouteQueryCodec,
  createJsonCollectionQueryPolicy,
  type CollectionWorkflow,
} from "@yueli/ui/collection";
import { useVueCollectionWorkflow } from "@yueli/ui/collection/vue";
import { createVueRouterCollectionQuerySync } from "@yueli/ui/collection/vue-router";
import { CommentModerationCollection } from "@yueli/ui/comments/admin";
import type {
  CommentModerationCollectionActions,
  CommentModerationCollectionModel,
  CommentModerationItem,
  CommentModerationLifecycle,
} from "@yueli/ui/comments/admin";
import { createBlogNotifier } from "~/utils/feedback";
import type { CommentAdminView, MyComments } from "~/types";

// Author moderation console: comments on my posts, filterable by status. Anonymous
// comments arrive as 待审核 (pending); approving makes them public. Auth-gated;
// the list needs the author's Bearer (BFF-injected), so client-only.
definePageMeta({ layout: "manage", middleware: "auth" });
useSeoMeta({ title: "评论 · 控制台" });

const { call } = useApi();
const toast = createBlogNotifier(useToast());
const router = useRouter();

type CommentStatus = "2" | "1" | "3" | "4" | "0";
const lifecycleStatusMap: Record<CommentModerationLifecycle, CommentStatus> = {
  all: "0",
  pending: "2",
  approved: "1",
  spam: "3",
  trash: "4",
};
const statusLifecycleMap: Record<CommentStatus, CommentModerationLifecycle> = {
  "0": "all",
  "2": "pending",
  "1": "approved",
  "3": "spam",
  "4": "trash",
};
interface CommentCollectionQuery {
  q: string;
  status: CommentStatus;
  sortBy: "created";
  sortOrder: "asc" | "desc";
  page: number;
  size: number;
}

const statuses = ["2", "1", "3", "4", "0"] as const;
const pageSizes = [20, 50, 100] as const;
const defaultQuery: CommentCollectionQuery = {
  q: "",
  status: "0",
  sortBy: "created",
  sortOrder: "desc",
  page: 1,
  size: 20,
};
const queryPolicy = createJsonCollectionQueryPolicy<CommentCollectionQuery>();
const searchInput = ref("");
const sync = createVueRouterCollectionQuerySync({
  router,
  codec: createCollectionRouteQueryCodec({
    q: { kind: "string", default: defaultQuery.q, maxLength: 200 },
    status: { kind: "enum", values: statuses, default: defaultQuery.status },
    sortBy: {
      kind: "enum",
      values: ["created"] as const,
      default: defaultQuery.sortBy,
    },
    sortOrder: {
      kind: "enum",
      values: ["asc", "desc"] as const,
      default: defaultQuery.sortOrder,
    },
    page: { kind: "positive-integer", default: defaultQuery.page },
    size: {
      kind: "positive-integer",
      values: pageSizes,
      default: defaultQuery.size,
    },
  }),
});

async function load(
  nextQuery: Readonly<CommentCollectionQuery>,
  activeWorkflow: CollectionWorkflow<
    CommentAdminView,
    string,
    CommentCollectionQuery
  >,
) {
  const token = activeWorkflow.beginLoad();
  try {
    const result = await call<MyComments>("/api/v1/comments/mine", {
      query: {
        status: Number(nextQuery.status),
        keyword: nextQuery.q || undefined,
        sortBy: nextQuery.sortBy,
        sortOrder: nextQuery.sortOrder,
        page: nextQuery.page,
        size: nextQuery.size,
      },
    });
    const lastPage = Math.max(1, Math.ceil(result.total / nextQuery.size));
    if (nextQuery.page > lastPage) {
      activeWorkflow.setQuery({ ...nextQuery, page: lastPage });
      return;
    }
    activeWorkflow.resolveLoad(token, {
      items: result.items,
      total: result.total,
    });
  } catch {
    activeWorkflow.rejectLoad(token, {
      key: "blog.comments.collection.load_failed",
    });
  }
}

const {
  snapshot: collection,
  workflow,
  reload,
} = useVueCollectionWorkflow({
  initialQuery: defaultQuery,
  queryPolicy,
  keyOf: (comment: CommentAdminView) => comment.id,
  querySync: sync,
  dataQueryKey: (query) => JSON.stringify(query),
  load,
});
const query = computed(() => collection.value.query);
function updateQuery(patch: Partial<CommentCollectionQuery>, resetPage = true) {
  workflow.setQuery({
    ...query.value,
    ...patch,
    ...(resetPage ? { page: 1 } : {}),
  });
}
const q = computed(() => query.value.q);
const status = computed({
  get: () => query.value.status,
  set: (value: CommentStatus) => updateQuery({ status: value }),
});
const lifecycle = computed(() => statusLifecycleMap[status.value]);
const sortBy = computed(() => query.value.sortBy);
const sortOrder = computed({
  get: () => query.value.sortOrder,
  set: (value: "asc" | "desc") => updateQuery({ sortOrder: value }),
});
function changeColumnSort() {
  sortOrder.value = sortOrder.value === "asc" ? "desc" : "asc";
}
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
searchInput.value = collection.value.query.q;
watch(q, (value) => {
  if (searchInput.value !== value) searchInput.value = value;
});

const busy = ref("");
const emptyingTrash = ref(false);
const showDelete = ref(false);
const deleteTarget = ref<CommentAdminView | null>(null);
const mounted = ref(false);
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
const showSkeleton = useMinimumLoading(
  computed(() => !mounted.value || pending.value),
);
const items = computed(() => collection.value.items);
const total = computed(() => collection.value.total);

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

async function setStatus(id: string, nextStatus: number) {
  busy.value = id;
  try {
    await call(`/api/v1/comments/${id}`, {
      method: "PATCH",
      body: { status: nextStatus },
    });
    if (workflow.isSelected(id)) workflow.toggleKey(id);
    await reload();
  } catch (e: any) {
    toast.add({
      title: "操作失败",
      description: e?.data?.message,
      color: "error",
    });
  } finally {
    busy.value = "";
  }
}

async function remove(id: string) {
  busy.value = id;
  try {
    await call(`/api/v1/comments/${id}`, { method: "DELETE" });
    if (workflow.isSelected(id)) workflow.toggleKey(id);
    await reload();
  } catch (e: any) {
    toast.add({
      title: "删除失败",
      description: e?.data?.message,
      color: "error",
    });
  } finally {
    busy.value = "";
  }
}

async function emptyTrash() {
  if (emptyingTrash.value) return false;
  emptyingTrash.value = true;
  try {
    for (let batch = 0; batch < 100; batch += 1) {
      const result = await call<MyComments>("/api/v1/comments/mine", {
        query: {
          status: 4,
          sortBy: "created",
          sortOrder: "asc",
          page: 1,
          size: 100,
        },
      });
      if (!result.items.length) break;
      const outcomes = await Promise.allSettled(
        result.items.map((comment) =>
          call(`/api/v1/comments/${comment.id}`, { method: "DELETE" }),
        ),
      );
      if (outcomes.some((outcome) => outcome.status === "rejected")) {
        throw new Error("部分评论未能永久删除");
      }
      if (result.items.length < 100) break;
    }
    clearSelection();
    await reload();
    return true;
  } catch (error: any) {
    toast.add({
      title: "回收站未清空",
      description: error?.message || "请稍后重试。",
      color: "error",
    });
    return false;
  } finally {
    emptyingTrash.value = false;
  }
}

const batchAction = ref<"" | "1" | "3" | "4">("");
const batchItems = computed(() =>
  lifecycle.value === "trash"
    ? [{ label: "恢复", value: "1" }]
    : lifecycle.value === "spam"
      ? [
          { label: "恢复", value: "1" },
          { label: "移入回收站", value: "4" },
        ]
      : [
          { label: "通过", value: "1" },
          { label: "标记垃圾", value: "3" },
          { label: "移入回收站", value: "4" },
        ],
);
const batchRunning = ref(false);
const batchResult = ref<{ success: number; failed: number } | null>(null);
watch([q, status, sortBy, sortOrder, page, size], () => {
  batchAction.value = "";
  batchResult.value = null;
});
async function applyBatch() {
  if (!batchAction.value || !selectedIds.value.length || batchRunning.value)
    return;
  const ids = [...selectedIds.value];
  batchRunning.value = true;
  try {
    const results = await Promise.allSettled(
      ids.map((id) =>
        call(`/api/v1/comments/${id}`, {
          method: "PATCH",
          body: { status: Number(batchAction.value) },
        }),
      ),
    );
    const failedIds = ids.filter(
      (_, index) => results[index]?.status === "rejected",
    );
    batchResult.value = {
      success: ids.length - failedIds.length,
      failed: failedIds.length,
    };
    if (failedIds.length) replaceSelection(failedIds);
    else clearSelection();
    batchAction.value = "";
    await reload();
  } finally {
    batchRunning.value = false;
  }
}

function askRemove(comment: CommentAdminView) {
  deleteTarget.value = comment;
  showDelete.value = true;
}

async function confirmRemove() {
  if (!deleteTarget.value) return;
  const id = deleteTarget.value.id;
  await remove(id);
  showDelete.value = false;
  if (deleteTarget.value?.id === id) deleteTarget.value = null;
}

const statusMeta: Record<
  number,
  {
    label: string;
    color: "warning" | "success" | "error" | "neutral";
    icon: string;
  }
> = {
  1: { label: "已通过", color: "success", icon: "i-tabler-circle-check" },
  2: { label: "待审核", color: "warning", icon: "i-tabler-clock" },
  3: { label: "垃圾评论", color: "error", icon: "i-tabler-alert-triangle" },
  4: { label: "回收站", color: "neutral", icon: "i-tabler-trash" },
};
const meta = (s: number) => statusMeta[s] || statusMeta[2]!;
function rowActionItems(comment: CommentAdminView) {
  const disabled = busy.value === comment.id;
  if (comment.status === 4) {
    return [
      [
        {
          id: "restore",
          label: "恢复评论",
          icon: "i-tabler-restore",
          disabled,
          onSelect: () => void setStatus(comment.id, 1),
        },
      ],
      [
        {
          id: "delete-permanently",
          label: "永久删除",
          icon: "i-tabler-trash-x",
          tone: "danger" as const,
          disabled,
          onSelect: () => askRemove(comment),
        },
      ],
    ];
  }
  return [
    [
      comment.status === 3
        ? {
            id: "restore",
            label: "恢复评论",
            icon: "i-tabler-restore",
            disabled,
            onSelect: () => void setStatus(comment.id, 1),
          }
        : {
            id: "spam",
            label: "标记为垃圾",
            icon: "i-tabler-alert-triangle",
            disabled,
            onSelect: () => void setStatus(comment.id, 3),
          },
      {
        id: "trash",
        label: "移入回收站",
        icon: "i-tabler-trash",
        disabled,
        onSelect: () => void setStatus(comment.id, 4),
      },
    ],
  ];
}
function submitCollectionSearch(value: string) {
  if (searchTimer) clearTimeout(searchTimer);
  searchInput.value = value;
  updateQuery({ q: value.trim() });
}
function changeLifecycle(value: CommentModerationLifecycle) {
  status.value = lifecycleStatusMap[value];
}
function moderationItem(comment: CommentAdminView): CommentModerationItem {
  return {
    id: comment.id,
    content: comment.content,
    createdAt: comment.createdAt,
    authorName: comment.authorName || "匿名用户",
    avatarUrl: comment.avatarUrl,
    authorEmail: comment.authorEmail,
    anonymous: !comment.userId,
    reply: Boolean(comment.parentId),
    approve: comment.status === 2,
    approving: busy.value === comment.id,
    actions: rowActionItems(comment),
    ...(comment.status === 1 ? {} : { status: meta(comment.status) }),
    source: {
      label: comment.postTitle || comment.postSlug || "文章已删除",
      ...(comment.postSlug ? { to: `/posts/${comment.postSlug}` } : {}),
      icon: "i-tabler-article",
    },
  };
}
const moderationModel = computed<CommentModerationCollectionModel>(() => ({
  search: searchInput.value,
  searchPlaceholder: "搜索评论内容、评论者或文章…",
  items: items.value.map(moderationItem),
  state: collection.value.issue
    ? "error"
    : showSkeleton.value
      ? "loading"
      : "ready",
  errorMessage: collection.value.issue?.key,
  total: total.value,
  page: page.value,
  pageSize: size.value,
  pageSizes,
  activeFilterCount: 0,
  controls: [],
  sortOrder: sortOrder.value,
  lifecycle: lifecycle.value,
  emptyingTrash: emptyingTrash.value,
  selection: {
    enabled: true,
    count: selectionCount.value,
    pageSelected: isPageSelected.value,
    pageIndeterminate: isPageIndeterminate.value,
    isSelected,
  },
}));
const moderationActions: CommentModerationCollectionActions = {
  updateSearch: (value) => {
    searchInput.value = value;
  },
  search: submitCollectionSearch,
  controlChange: () => undefined,
  clearFilters: () => changeLifecycle("all"),
  retry: reload,
  sort: changeColumnSort,
  lifecycleChange: changeLifecycle,
  emptyTrash,
  approve: (id) => setStatus(id, 1),
  pageChange: (value) => {
    page.value = value;
  },
  pageSizeChange: (value) => {
    size.value = value;
  },
  togglePage: (value) => togglePage(value),
  toggleItem: (id, selected) => toggleOne(id, selected),
  clearSelection,
};
</script>

<template>
  <div class="space-y-5">
    <ManagePageHeader title="评论" />

    <CommentModerationCollection
      :model="moderationModel"
      :actions="moderationActions"
      :format-date="dateTime"
    >

      <template #bulk-actions>
        <USelect
          v-model="batchAction"
          :items="batchItems"
          value-key="value"
          placeholder="批量操作"
          size="xs"
          class="w-28"
        />
        <UButton
          label="应用"
          color="primary"
          variant="soft"
          size="xs"
          :loading="batchRunning"
          :disabled="!batchAction"
          @click="applyBatch"
        />
      </template>

    </CommentModerationCollection>

    <UAlert
      v-if="batchResult"
      class="mt-3"
      :color="batchResult.failed ? 'warning' : 'success'"
      variant="subtle"
      :icon="
        batchResult.failed ? 'i-tabler-alert-triangle' : 'i-tabler-circle-check'
      "
      :title="`已处理 ${batchResult.success} 条评论`"
      :description="
        batchResult.failed
          ? `${batchResult.failed} 条评论处理失败，已保留选中。`
          : '批量操作已完成。'
      "
      close
      @update:open="batchResult = null"
    />

    <UModal
      v-model:open="showDelete"
      title="永久删除评论"
      :description="`永久删除「${deleteTarget?.authorName || '匿名用户'}」的这条评论及其回复？此操作不可撤销。`"
      :ui="{ footer: 'justify-end' }"
    >
      <template #footer="{ close }">
        <UButton
          label="取消"
          color="neutral"
          variant="outline"
          @click="close"
        />
        <UButton
          label="永久删除"
          icon="i-tabler-trash-x"
          color="error"
          :loading="busy === deleteTarget?.id"
          @click="confirmRemove"
        />
      </template>
    </UModal>
  </div>
</template>

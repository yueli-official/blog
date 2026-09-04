<script setup lang="ts">
import { useActionFeedback } from "@yueli/ui/feedback";
import { ActionFeedbackButton } from "@yueli/ui/feedback/pattern";
import { AdminRowActions } from "@yueli/ui/admin";
import type { AdminRowActionItem } from "@yueli/ui/admin";
import {
  type CollectionPanelMessages,
  type CollectionPanelState,
} from "@yueli/ui/collection";
import { CollectionPanel } from "@yueli/ui/collection/pattern";
import type { ListSeries, SeriesView } from "~/types";

definePageMeta({ layout: "manage", middleware: "auth" });
useSeoMeta({ title: "系列 · 控制台" });

const { call } = useApi();
const { can, isAdministrator, profile } = useMe();
const canCreate = computed(() => can("blog.series.create"));
const mounted = ref(false);
onMounted(() => {
  mounted.value = true;
});
function canEditSeries(series: SeriesView) {
  return isAdministrator.value || series.authorId === profile.value?.id;
}
const search = ref("");
const {
  data,
  pending,
  error,
  refresh,
} = await useAsyncData("manage-series", () => call<ListSeries>("/api/v1/series"), {
  server: false,
  default: () => ({ items: [] }),
});

const items = computed(() => {
  const query = search.value.trim().toLocaleLowerCase("zh-CN");
  const all = [...(data.value?.items ?? [])].sort((left, right) =>
    left.name.localeCompare(right.name, "zh-CN"),
  );
  if (!query) return all;
  return all.filter((item) =>
    [item.name, item.slug, item.description]
      .filter(Boolean)
      .some((value) => String(value).toLocaleLowerCase("zh-CN").includes(query)),
  );
});

const messages: CollectionPanelMessages = {
  searchPlaceholder: "搜索系列名称、slug 或描述…",
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
  emptyTitle: "还没有系列",
  emptyDescription: "创建系列后，可以在文章编辑器中安排连载顺序。",
  errorTitle: "加载失败",
  retry: "重新加载",
  showing: (first, last, count) => `显示 ${first}–${last}，共 ${count} 个`,
  pageSize: "每页",
  pageSizeControl: "每页数量",
  pageSizeOption: (value) => `${value} 个`,
};
const panelState = computed<CollectionPanelState>(() =>
  error.value ? "error" : !mounted.value || pending.value ? "loading" : "ready",
);

const editorOpen = ref(false);
const deleteOpen = ref(false);
const current = ref<SeriesView | null>(null);
const form = reactive({ name: "", slug: "", description: "" });
const slugTouched = ref(false);
const slugError = ref("");
const operationError = ref("");
const deleting = ref(false);
const {
  status: saveStatus,
  pending: markSaving,
  success: markSaved,
  reset: resetSave,
} = useActionFeedback();

function clientSlug(value: string) {
  return value
    .normalize("NFKC")
    .toLocaleLowerCase("zh-CN")
    .trim()
    .replace(/[^\p{L}\p{N}]+/gu, "-")
    .replace(/(^-|-$)/gu, "");
}

watch(
  () => form.name,
  (name) => {
    if (!current.value && !slugTouched.value) form.slug = clientSlug(name);
  },
);

function touchSlug() {
  slugTouched.value = true;
  slugError.value = "";
}

function openCreate() {
  current.value = null;
  form.name = "";
  form.slug = "";
  form.description = "";
  slugTouched.value = false;
  slugError.value = "";
  operationError.value = "";
  resetSave();
  editorOpen.value = true;
}

function openEdit(series: SeriesView) {
  current.value = series;
  form.name = series.name;
  form.slug = series.slug;
  form.description = series.description ?? "";
  slugTouched.value = true;
  slugError.value = "";
  operationError.value = "";
  resetSave();
  editorOpen.value = true;
}

async function save() {
  const name = form.name.trim();
  const slug = form.slug.trim();
  if (!name || !slug || saveStatus.value === "pending") return;
  markSaving();
  slugError.value = "";
  operationError.value = "";
  try {
    if (current.value) {
      await call(`/api/v1/series/${current.value.id}`, {
        method: "PATCH",
        body: { name, slug, description: form.description.trim() },
      });
    } else {
      await call("/api/v1/series", {
        method: "POST",
        body: { name, slug, description: form.description.trim() },
      });
    }
    await refresh();
    markSaved();
    editorOpen.value = false;
  } catch (exception: any) {
    resetSave();
    const failureCode =
      exception?.data?.code || exception?.data?.failure?.code;
    if (failureCode === "blog.slug_taken") {
      slugError.value = "该 Slug 已被使用，请换一个。";
    } else {
      operationError.value =
        blogFailureMessage(exception, "暂时无法保存系列，请稍后重试。");
    }
  }
}

function requestDelete() {
  deleteOpen.value = true;
}

function seriesRowActions(item: SeriesView): AdminRowActionItem[] {
  return [
    {
      id: "view",
      label: `查看前台系列：${item.name}`,
      icon: "i-tabler-external-link",
      to: `/series/${item.slug}`,
      target: "_blank",
      rel: "noopener",
    },
    {
      id: "edit",
      label: `编辑系列：${item.name}`,
      icon: "i-tabler-pencil",
      hidden: !canEditSeries(item),
      onSelect: () => openEdit(item),
    },
  ];
}

async function removeSeries() {
  if (!current.value || deleting.value) return;
  deleting.value = true;
  operationError.value = "";
  try {
    await call(`/api/v1/series/${current.value.id}`, { method: "DELETE" });
    deleteOpen.value = false;
    editorOpen.value = false;
    current.value = null;
    await refresh();
  } catch (exception: any) {
    operationError.value =
      blogFailureMessage(exception, "暂时无法删除系列，请稍后重试。");
    deleteOpen.value = false;
  } finally {
    deleting.value = false;
  }
}
</script>

<template>
  <div class="space-y-5">
    <ManagePageHeader title="系列">
      <template #actions>
        <UButton
          v-if="canCreate"
          icon="i-tabler-plus"
          label="新建系列"
          @click="openCreate"
        />
      </template>
    </ManagePageHeader>

    <CollectionPanel
      v-model:search="search"
      label="系列列表"
      :items="items"
      :item-key="(item: SeriesView) => item.id"
      :item-label="(item: SeriesView) => item.name"
      :messages="messages"
      :state="panelState"
      :error-message="error ? '无法加载系列，请稍后重试。' : ''"
      :total="items.length"
      :page="1"
      :page-size="30"
      :page-sizes="[30]"
      @search="search = $event"
      @retry="refresh"
    >
      <template #columns>
        <div class="grid grid-cols-[minmax(0,1fr)_8rem_5.75rem] gap-3 px-3 sm:px-4">
          <span>系列</span>
          <span class="text-right">已发布文章</span>
          <span class="text-right">操作</span>
        </div>
      </template>

      <template #item="{ item }">
        <div
          class="grid w-full grid-cols-[minmax(0,1fr)_5.75rem] items-center gap-3 px-3 py-3 transition-colors hover:bg-muted sm:px-4 lg:grid-cols-[minmax(14rem,1fr)_8rem_5.75rem]"
        >
          <button
            type="button"
            class="group min-w-0 rounded-lg text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary disabled:cursor-default"
            :disabled="!canEditSeries(item)"
            @click="openEdit(item)"
          >
            <span class="flex min-w-0 items-center gap-3">
              <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-elevated text-dimmed transition group-hover:bg-primary/10 group-hover:text-primary">
                <UIcon name="i-tabler-stack-2" class="size-4" />
              </span>
              <span class="min-w-0">
                <span class="line-clamp-1 text-sm font-medium text-highlighted">{{ item.name }}</span>
                <span class="mt-0.5 block line-clamp-1 font-mono text-xs text-muted">/{{ item.slug }}</span>
                <span v-if="item.description" class="mt-1 block line-clamp-1 text-xs text-muted">{{ item.description }}</span>
              </span>
            </span>
          </button>
          <span class="col-start-1 pl-13 text-xs text-muted lg:col-start-auto lg:pl-0 lg:text-right">
            <span class="font-semibold text-highlighted">{{ item.postCount || 0 }}</span> 篇
          </span>
          <AdminRowActions
            class="row-start-1 col-start-2 lg:row-auto lg:col-start-auto"
            :label="`${item.name} 的操作`"
            :items="seriesRowActions(item)"
          />
        </div>
      </template>
    </CollectionPanel>

    <USlideover
      v-model:open="editorOpen"
      :title="current ? '编辑系列' : '新建系列'"
      :ui="{ content: 'w-full max-w-xl bg-default', body: 'bg-muted p-4 sm:p-5', footer: 'bg-default' }"
    >
      <template #body>
        <div class="space-y-4 rounded-xl bg-default p-4 ring-1 ring-default">
          <UAlert
            v-if="operationError"
            color="error"
            variant="soft"
            icon="i-tabler-alert-circle"
            title="操作失败"
            :description="operationError"
          />
          <UFormField label="名称" required>
            <UInput v-model="form.name" class="w-full" autofocus placeholder="系列名称" />
          </UFormField>
          <UFormField label="Slug" required :error="slugError || undefined">
            <UInput
              v-model="form.slug"
              class="w-full"
              placeholder="例如 release-notes"
              @input="touchSlug"
            />
          </UFormField>
          <UFormField label="描述">
            <UTextarea v-model="form.description" class="w-full" :rows="4" placeholder="简要说明这个系列" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full items-center justify-between gap-3">
          <UButton
            v-if="current && canEditSeries(current)"
            color="error"
            variant="ghost"
            label="删除系列"
            @click="requestDelete"
          />
          <span v-else />
          <div class="flex gap-2">
            <UButton color="neutral" variant="outline" label="关闭" @click="void (editorOpen = false)" />
            <ActionFeedbackButton
              :status="saveStatus"
              idle-label="保存"
              pending-label="保存中"
              success-label="已保存"
              :disabled="!form.name.trim() || !form.slug.trim()"
              @click="save"
            />
          </div>
        </div>
      </template>
    </USlideover>

    <UModal
      v-model:open="deleteOpen"
      title="删除系列？"
      description="系列内文章会保留，并自动移出该系列。"
      :ui="{ content: 'w-full max-w-md bg-default' }"
    >
      <template #body>
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" label="取消" :disabled="deleting" @click="void (deleteOpen = false)" />
          <UButton color="error" label="删除系列" :loading="deleting" @click="removeSeries" />
        </div>
      </template>
    </UModal>
  </div>
</template>

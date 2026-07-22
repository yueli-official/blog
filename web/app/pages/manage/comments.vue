<script setup lang="ts">
import {
  createCollectionRouteQueryCodec,
  createJsonCollectionQueryPolicy,
  type CollectionControl,
  type CollectionControlValue,
  type CollectionPanelMessages,
  type CollectionWorkflow
} from '@yueli/ui/collection'
import { CollectionPanel } from '@yueli/ui/collection/pattern'
import { useVueCollectionWorkflow } from '@yueli/ui/collection/vue'
import { createVueRouterCollectionQuerySync } from '@yueli/ui/collection/vue-router'
import { PageHeader } from '@yueli/ui/dashboard/pattern'
import { createPlatformNotifier } from '@platform/ui/feedback'
import type { CommentAdminView, MyComments } from '~/types'

// Author moderation console: comments on my posts, filterable by status. Anonymous
// comments arrive as 待审核 (pending); approving makes them public. Auth-gated;
// the list needs the author's Bearer (BFF-injected), so client-only.
definePageMeta({ layout: 'manage', middleware: 'auth' })
useSeoMeta({ title: '评论 · 控制台' })

const { call } = useApi()
const { isOwner } = useMe()
const toast = createPlatformNotifier(useToast())
const router = useRouter()

type CommentStatus = '2' | '1' | '3' | '4' | '0'
interface CommentCollectionQuery {
  q: string
  status: CommentStatus
  page: number
  size: number
}

const statuses = ['2', '1', '3', '4', '0'] as const
const pageSizes = [20, 50, 100] as const
const defaultQuery: CommentCollectionQuery = {
  q: '',
  status: '2',
  page: 1,
  size: 20
}
const queryPolicy = createJsonCollectionQueryPolicy<CommentCollectionQuery>()
const searchInput = ref('')
const sync = createVueRouterCollectionQuerySync({
  router,
  codec: createCollectionRouteQueryCodec({
    q: { kind: 'string', default: defaultQuery.q, maxLength: 200 },
    status: { kind: 'enum', values: statuses, default: defaultQuery.status },
    page: { kind: 'positive-integer', default: defaultQuery.page },
    size: {
      kind: 'positive-integer',
      values: pageSizes,
      default: defaultQuery.size
    }
  })
})

async function load(nextQuery: Readonly<CommentCollectionQuery>, activeWorkflow: CollectionWorkflow<CommentAdminView, string, CommentCollectionQuery>) {
  const token = activeWorkflow.beginLoad()
  try {
    const result = await call<MyComments>('/api/v1/comments/mine', {
      query: {
        status: Number(nextQuery.status),
        keyword: nextQuery.q || undefined,
        page: nextQuery.page,
        size: nextQuery.size
      }
    })
    const lastPage = Math.max(1, Math.ceil(result.total / nextQuery.size))
    if (nextQuery.page > lastPage) {
      activeWorkflow.setQuery({ ...nextQuery, page: lastPage })
      return
    }
    activeWorkflow.resolveLoad(token, {
      items: result.items,
      total: result.total
    })
  } catch {
    activeWorkflow.rejectLoad(token, {
      key: 'blog.comments.collection.load_failed'
    })
  }
}

const {
  snapshot: collection,
  workflow,
  reload
} = useVueCollectionWorkflow({
  initialQuery: defaultQuery,
  queryPolicy,
  keyOf: (comment: CommentAdminView) => comment.id,
  querySync: sync,
  dataQueryKey: (query) => JSON.stringify(query),
  load
})
const query = computed(() => collection.value.query)
function updateQuery(patch: Partial<CommentCollectionQuery>, resetPage = true) {
  workflow.setQuery({
    ...query.value,
    ...patch,
    ...(resetPage ? { page: 1 } : {})
  })
}
const q = computed(() => query.value.q)
const status = computed({
  get: () => query.value.status,
  set: (value: CommentStatus) => updateQuery({ status: value })
})
const page = computed({
  get: () => query.value.page,
  set: (value: number) => updateQuery({ page: value }, false)
})
const size = computed({
  get: () => query.value.size,
  set: (value: number) => updateQuery({ size: value })
})

let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(searchInput, (value) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => updateQuery({ q: value.trim() }), 300)
})
searchInput.value = collection.value.query.q
watch(q, (value) => {
  if (searchInput.value !== value) searchInput.value = value
})

const busy = ref('')
const showDelete = ref(false)
const deleteTarget = ref<CommentAdminView | null>(null)
const mounted = ref(false)
onMounted(() => {
  mounted.value = true
})
onScopeDispose(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
const pending = computed(() => collection.value.loadState === 'loading' || collection.value.loadState === 'refreshing')
const showSkeleton = useMinimumLoading(computed(() => !mounted.value || pending.value))
const items = computed(() => collection.value.items)
const total = computed(() => collection.value.total)

const selectedIds = computed(() => (collection.value.selection.mode === 'keys' ? collection.value.selection.keys : []))
const selectionCount = computed(() => collection.value.selection.count)
const isPageSelected = computed(() => collection.value.isPageSelected)
const isPageIndeterminate = computed(() => collection.value.isPageIndeterminate)
const isSelected = (id: string) => workflow.isSelected(id)
const toggleOne = (id: string, selected?: boolean) => {
  if (selected === undefined || selected !== workflow.isSelected(id)) workflow.toggleKey(id)
}
const togglePage = (selected?: boolean | 'indeterminate') => workflow.togglePage(selected === true)
const clearSelection = () => workflow.clearSelection()
function replaceSelection(ids: readonly string[]) {
  workflow.clearSelection()
  for (const id of ids) workflow.toggleKey(id)
}

async function setStatus(id: string, nextStatus: number) {
  busy.value = id
  try {
    await call(`/api/v1/comments/${id}`, {
      method: 'PATCH',
      body: { status: nextStatus }
    })
    if (workflow.isSelected(id)) workflow.toggleKey(id)
    await reload()
  } catch (e: any) {
    toast.add({
      title: '操作失败',
      description: e?.data?.message,
      color: 'error'
    })
  } finally {
    busy.value = ''
  }
}

async function remove(id: string) {
  busy.value = id
  try {
    await call(`/api/v1/comments/${id}`, { method: 'DELETE' })
    if (workflow.isSelected(id)) workflow.toggleKey(id)
    await reload()
  } catch (e: any) {
    toast.add({
      title: '删除失败',
      description: e?.data?.message,
      color: 'error'
    })
  } finally {
    busy.value = ''
  }
}

const batchAction = ref<'' | '1' | '3' | '4'>('')
const batchRunning = ref(false)
const batchResult = ref<{ success: number; failed: number } | null>(null)
watch([q, status, page, size], () => {
  batchAction.value = ''
  batchResult.value = null
})
async function applyBatch() {
  if (!batchAction.value || !selectedIds.value.length || batchRunning.value) return
  const ids = [...selectedIds.value]
  batchRunning.value = true
  try {
    const results = await Promise.allSettled(
      ids.map((id) =>
        call(`/api/v1/comments/${id}`, {
          method: 'PATCH',
          body: { status: Number(batchAction.value) }
        })
      )
    )
    const failedIds = ids.filter((_, index) => results[index]?.status === 'rejected')
    batchResult.value = {
      success: ids.length - failedIds.length,
      failed: failedIds.length
    }
    if (failedIds.length) replaceSelection(failedIds)
    else clearSelection()
    batchAction.value = ''
    await reload()
  } finally {
    batchRunning.value = false
  }
}

function askRemove(comment: CommentAdminView) {
  deleteTarget.value = comment
  showDelete.value = true
}

async function confirmRemove() {
  if (!deleteTarget.value) return
  const id = deleteTarget.value.id
  await remove(id)
  showDelete.value = false
  if (deleteTarget.value?.id === id) deleteTarget.value = null
}

const statusMeta: Record<
  number,
  {
    label: string
    color: 'warning' | 'success' | 'error' | 'neutral'
    icon: string
  }
> = {
  1: { label: '已通过', color: 'success', icon: 'i-tabler-circle-check' },
  2: { label: '待审核', color: 'warning', icon: 'i-tabler-clock' },
  3: { label: '垃圾', color: 'error', icon: 'i-tabler-alert-triangle' },
  4: { label: '回收站', color: 'neutral', icon: 'i-tabler-trash' }
}
const meta = (s: number) => statusMeta[s] || statusMeta[2]!
function authorInitial(name: string) {
  return (name || '?').charAt(0).toUpperCase()
}

const statusItems = [
  { label: '待审核', value: '2' },
  { label: '已通过', value: '1' },
  { label: '垃圾', value: '3' },
  { label: '回收站', value: '4' },
  { label: '全部状态', value: '0' }
]
const collectionControls = computed<CollectionControl[]>(() => [
  {
    kind: 'select',
    id: 'status',
    label: '评论状态',
    value: status.value,
    options: statusItems,
    icon: 'i-tabler-filter',
    class: 'w-32'
  }
])
const collectionMessages: CollectionPanelMessages = {
  searchPlaceholder: '搜索评论内容、评论者或文章…',
  searchAction: '搜索',
  filtersAction: '筛选',
  activeFilters: (count) => `筛选（${count}）`,
  clearFilters: '清除筛选',
  selectPage: '选择当前页评论',
  selectItem: (label) => `选择评论：${label}`,
  bulkRegion: '评论批量操作',
  selected: (count) => `已选择 ${count} 条评论`,
  selectAllResults: '选择全部结果',
  clearSelection: '取消选择',
  emptyTitle: '没有匹配的评论',
  emptyDescription: '请调整搜索内容或评论状态后重试。',
  errorTitle: '评论加载失败',
  retry: '重新加载',
  showing: (first, last, count) => `显示 ${first}–${last}，共 ${count} 条`,
  pageSize: '每页',
  pageSizeControl: '每页评论数量',
  pageSizeOption: (value) => `${value} 条`
}
function changeCollectionControl(id: string, value: CollectionControlValue) {
  if (id === 'status' && statuses.includes(value as CommentStatus)) status.value = value as CommentStatus
}
function submitCollectionSearch(value: string) {
  if (searchTimer) clearTimeout(searchTimer)
  searchInput.value = value
  updateQuery({ q: value.trim() })
}
function clearActiveFilters() {
  status.value = defaultQuery.status
}
const commentKey = (comment: CommentAdminView) => comment.id
const commentLabel = (comment: CommentAdminView) => `${comment.authorName || '匿名用户'}的评论`
</script>

<template>
  <div>
    <PageHeader title="评论管理">
      <template #subtitle>
        <span v-if="isOwner">全站评论审核 · 你是站长,可处理所有作者文章下的评论</span>
        <span v-else>审核你文章下的评论 · 匿名评论默认待审,通过后公开</span>
      </template>
    </PageHeader>

    <CollectionPanel
      v-model:search="searchInput"
      :items="items"
      :item-key="commentKey"
      :item-label="commentLabel"
      :controls="collectionControls"
      :messages="collectionMessages"
      :state="collection.issue ? 'error' : showSkeleton ? 'loading' : 'ready'"
      :error-message="collection.issue?.key"
      :total="total"
      :page="page"
      :page-size="size"
      :page-sizes="pageSizes"
      :active-filter-count="status === defaultQuery.status ? 0 : 1"
      selectable
      :selection-count="selectionCount"
      :page-selected="isPageSelected"
      :page-indeterminate="isPageIndeterminate"
      :is-selected="isSelected"
      label="评论列表"
      @search="submitCollectionSearch"
      @control-change="changeCollectionControl"
      @clear-filters="clearActiveFilters"
      @retry="reload"
      @toggle-page="togglePage"
      @toggle-item="toggleOne"
      @clear-selection="clearSelection"
      @page-change="
        (value) => {
          page = value
        }
      "
      @page-size-change="
        (value) => {
          size = value
        }
      "
    >
      <template #columns>
        <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3">
          <span>评论、评论者与文章</span>
          <span class="hidden w-40 text-right sm:block">状态与操作</span>
        </div>
      </template>

      <template #bulk-actions>
        <USelect
          v-model="batchAction"
          :items="[
            { label: '通过', value: '1' },
            { label: '标记垃圾', value: '3' },
            { label: '移入回收站', value: '4' }
          ]"
          value-key="value"
          placeholder="批量操作"
          size="xs"
          class="w-28"
        />
        <UButton label="应用" color="primary" variant="soft" size="xs" :loading="batchRunning" :disabled="!batchAction" @click="applyBatch" />
      </template>

      <template #item="{ item: c }">
        <div class="grid min-w-0 grid-cols-1 items-start gap-3 sm:grid-cols-[minmax(0,1fr)_auto]">
          <div class="flex min-w-0 gap-3">
            <UAvatar :text="authorInitial(c.authorName)" size="sm" class="mt-0.5 hidden shrink-0 sm:flex" />
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2 text-sm">
                <span class="font-medium text-highlighted">{{ c.authorName }}</span>
                <UBadge v-if="c.userId" label="会员" color="primary" variant="subtle" size="sm" />
                <span v-if="c.parentId" class="text-xs text-dimmed">· 回复</span>
                <span class="text-dimmed">·</span>
                <ClientOnly
                  ><span class="text-muted">{{ rel(c.createdAt) }}</span
                  ><template #fallback><span /></template
                ></ClientOnly>
              </div>
              <p class="mt-1 line-clamp-3 whitespace-pre-wrap text-sm leading-relaxed text-default">
                {{ c.content }}
              </p>
              <p class="mt-1 flex flex-wrap items-center gap-1 text-xs text-muted">
                <UIcon name="i-tabler-article" class="size-3.5" />
                <NuxtLink :to="`/posts/${c.postSlug}`" class="hover:text-primary hover:underline">{{ c.postTitle || c.postSlug }}</NuxtLink>
                <template v-if="c.authorEmail">
                  <span class="text-dimmed">·</span>
                  <span class="break-all">{{ c.authorEmail }}</span>
                </template>
              </p>
            </div>
          </div>
          <div class="flex w-full flex-col items-start gap-2 sm:w-40 sm:items-end">
            <UBadge :color="meta(c.status).color" :icon="meta(c.status).icon" :label="meta(c.status).label" variant="subtle" size="sm" />
            <div class="flex flex-wrap justify-start gap-1 sm:justify-end">
              <UButton v-if="c.status !== 1" label="通过" icon="i-tabler-check" size="xs" color="success" variant="soft" :loading="busy === c.id" @click="setStatus(c.id, 1)" />
              <UButton v-if="c.status !== 3" label="垃圾" icon="i-tabler-alert-triangle" size="xs" color="warning" variant="soft" :loading="busy === c.id" @click="setStatus(c.id, 3)" />
              <UButton icon="i-tabler-trash" size="xs" color="error" variant="ghost" square :loading="busy === c.id" :aria-label="`删除 ${c.authorName || '匿名用户'} 的评论`" @click="askRemove(c)" />
            </div>
          </div>
        </div>
      </template>
    </CollectionPanel>

    <UAlert
      v-if="batchResult"
      class="mt-3"
      :color="batchResult.failed ? 'warning' : 'success'"
      variant="subtle"
      :icon="batchResult.failed ? 'i-tabler-alert-triangle' : 'i-tabler-circle-check'"
      :title="`已处理 ${batchResult.success} 条评论`"
      :description="batchResult.failed ? `${batchResult.failed} 条评论处理失败，已保留选中。` : '批量操作已完成。'"
      close
      @update:open="batchResult = null"
    />

    <UModal v-model:open="showDelete" title="删除评论" :description="`确定删除「${deleteTarget?.authorName || '匿名用户'}」的这条评论?此操作不可撤销。`" :ui="{ footer: 'justify-end' }">
      <template #footer="{ close }">
        <UButton label="取消" color="neutral" variant="outline" @click="close" />
        <UButton label="删除" icon="i-tabler-trash" color="error" :loading="busy === deleteTarget?.id" @click="confirmRemove" />
      </template>
    </UModal>
  </div>
</template>

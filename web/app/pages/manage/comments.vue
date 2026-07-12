<script setup lang="ts">
import { createPlatformNotifier } from '@platform/ui/feedback'
import { ManageHeader, SkeletonList, ManageEmpty, ManageTabs, ManagePagination, ManageCollectionToolbar, ManageCollectionDock, ManagePageSelection } from '@platform/manage/components'
import { manageCollectionQueryFingerprint, serializeManageCollectionQuery, type ManageCollectionDefinition } from '@platform/manage/collection'
import { useManageCollectionState } from '@platform/manage/use-manage-collection-state'
import { useManageSelection } from '@platform/manage/use-manage-selection'
import type { CommentAdminView, MyComments } from '~/types'

// Author moderation console: comments on my posts, filterable by status. Anonymous
// comments arrive as 待审核 (pending); approving makes them public. Auth-gated;
// the list needs the author's Bearer (BFF-injected), so client-only.
definePageMeta({ layout: 'manage', middleware: 'auth' })
useSeoMeta({ title: '评论 · 控制台' })

const { call } = useApi()
const { isOwner } = useMe()
const toast = createPlatformNotifier(useToast())
const route = useRoute()
const router = useRouter()

// string keys for the shared ManageTabs (v-model is a string); converted to the
// numeric status the API expects. 0 = 全部.
const tabs = [
  { key: '2', label: '待审核' },
  { key: '1', label: '已通过' },
  { key: '3', label: '垃圾' },
  { key: '4', label: '回收站' },
  { key: '0', label: '全部' }
]
const collectionDefinition = {
  resourceKind: 'blog-comment',
  statuses: ['2', '1', '3', '4', '0'],
  views: ['list'],
  sortKeys: ['createdAt'],
  pageSizes: [20, 50, 100],
  defaultStatus: '2', defaultView: 'list', defaultSort: 'createdAt', defaultDirection: 'desc', defaultPageSize: 20,
  pagination: 'server', selection: 'page', filters: []
} as const satisfies ManageCollectionDefinition
const { searchInput, q, status, page, size, state: collectionState } = useManageCollectionState({
  definition: collectionDefinition,
  routeQuery: computed(() => route.query),
  replaceQuery: query => router.replace({ query })
})

const items = ref<CommentAdminView[]>([])
const total = ref(0)
const loading = ref(true)
const busy = ref('')
const showDelete = ref(false)
const deleteTarget = ref<CommentAdminView | null>(null)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size.value)))
const pageSizeItems = [20, 50, 100].map(value => ({ label: `${value}/页`, value }))

async function load() {
  loading.value = true
  try {
    const r = await call<MyComments>('/api/v1/comments/mine', { query: { status: Number(status.value), keyword: q.value || undefined, page: page.value, size: size.value } })
    items.value = r.items
    total.value = r.total
  } catch (e: any) {
    toast.add({ title: '加载失败', description: e?.data?.message, color: 'error' })
  } finally {
    loading.value = false
  }
}

const mounted = ref(false)
onMounted(() => { mounted.value = true; load() })
const showSkeleton = useMinLoading(computed(() => !mounted.value || loading.value))
watch([q, status, page, size], load)

const selectionResetKey = computed(() => manageCollectionQueryFingerprint(serializeManageCollectionQuery(collectionState.value, collectionDefinition)))
const { selectedIds, selectionCount, isPageSelected, isPageIndeterminate, isSelected, toggleOne, togglePage, keepOnly, clear: clearSelection } = useManageSelection({
  visibleIds: computed(() => items.value.map(comment => comment.id)), filteredTotal: total, resetKey: selectionResetKey
})

async function setStatus(id: string, nextStatus: number) {
  busy.value = id
  try {
    await call(`/api/v1/comments/${id}`, { method: 'PATCH', body: { status: nextStatus } })
    await load()
  } catch (e: any) {
    toast.add({ title: '操作失败', description: e?.data?.message, color: 'error' })
  } finally {
    busy.value = ''
  }
}

async function remove(id: string) {
  busy.value = id
  try {
    await call(`/api/v1/comments/${id}`, { method: 'DELETE' })
    await load()
  } catch (e: any) {
    toast.add({ title: '删除失败', description: e?.data?.message, color: 'error' })
  } finally {
    busy.value = ''
  }
}

const batchAction = ref<'' | '1' | '3' | '4'>('')
const batchRunning = ref(false)
const batchResult = ref<{ success: number, failed: number } | null>(null)
async function applyBatch() {
  if (!batchAction.value || !selectedIds.value.length || batchRunning.value) return
  const ids = [...selectedIds.value]
  batchRunning.value = true
  try {
    const results = await Promise.allSettled(ids.map(id => call(`/api/v1/comments/${id}`, {
      method: 'PATCH', body: { status: Number(batchAction.value) }
    })))
    const failedIds = ids.filter((_, index) => results[index]?.status === 'rejected')
    batchResult.value = { success: ids.length - failedIds.length, failed: failedIds.length }
    if (failedIds.length) keepOnly(failedIds)
    else clearSelection()
    batchAction.value = ''
    await load()
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

const statusMeta: Record<number, { label: string, color: 'warning' | 'success' | 'error' | 'neutral', icon: string }> = {
  1: { label: '已通过', color: 'success', icon: 'i-tabler-circle-check' },
  2: { label: '待审核', color: 'warning', icon: 'i-tabler-clock' },
  3: { label: '垃圾', color: 'error', icon: 'i-tabler-alert-triangle' },
  4: { label: '回收站', color: 'neutral', icon: 'i-tabler-trash' }
}
const meta = (s: number) => statusMeta[s] || statusMeta[2]!
function authorInitial(name: string) { return (name || '?').charAt(0).toUpperCase() }
</script>

<template>
  <div>
    <ManageHeader title="评论管理">
      <template #subtitle>
        <span v-if="isOwner">全站评论审核 · 你是站长,可处理所有作者文章下的评论</span>
        <span v-else>审核你文章下的评论 · 匿名评论默认待审,通过后公开</span>
      </template>
    </ManageHeader>

    <ManageTabs v-model="status" :items="tabs" class="mb-5" />

    <ManageCollectionToolbar
      v-model:search="searchInput"
      search-placeholder="搜索评论内容、评论者或文章…"
      class="mb-5"
    />

    <SkeletonList v-if="showSkeleton" :rows="8" />

    <ManageEmpty v-else-if="!items.length" icon="i-tabler-message-2" text="这里没有评论" />

    <div v-else class="blog-manage-panel divide-y divide-default overflow-hidden rounded-xl">
      <div v-for="c in items" :key="c.id" class="blog-manage-row grid grid-cols-[auto_minmax(0,1fr)] gap-3 px-3 py-4 sm:px-4">
        <UCheckbox :model-value="isSelected(c.id)" class="mt-1" @update:model-value="toggleOne(c.id)" />
        <div class="flex min-w-0 gap-3">
          <UAvatar :text="authorInitial(c.authorName)" size="sm" class="mt-0.5 hidden shrink-0 sm:flex" />
          <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2 text-sm">
            <span class="font-medium text-highlighted">{{ c.authorName }}</span>
            <UBadge v-if="c.userId" label="会员" color="primary" variant="subtle" size="sm" />
            <UBadge v-if="status === '0'" :color="meta(c.status).color" :icon="meta(c.status).icon" :label="meta(c.status).label" variant="subtle" size="sm" />
            <span v-if="c.parentId" class="text-xs text-dimmed">· 回复</span>
            <span class="text-dimmed">·</span>
            <ClientOnly><span class="text-muted">{{ rel(c.createdAt) }}</span><template #fallback><span /></template></ClientOnly>
          </div>
          <p class="mt-1 whitespace-pre-wrap text-sm leading-relaxed text-default">{{ c.content }}</p>
          <p class="mt-1 flex flex-wrap items-center gap-1 text-xs text-muted">
            <UIcon name="i-tabler-article" class="size-3.5" />
            <NuxtLink :to="`/posts/${c.postSlug}`" class="hover:text-primary hover:underline">{{ c.postTitle || c.postSlug }}</NuxtLink>
            <template v-if="c.authorEmail"><span class="text-dimmed">·</span>{{ c.authorEmail }}</template>
          </p>
          <div class="mt-2 flex flex-wrap gap-1.5">
            <UButton v-if="c.status !== 1" label="通过" icon="i-tabler-check" size="xs" color="success" variant="soft" :loading="busy === c.id" @click="setStatus(c.id, 1)" />
            <UButton v-if="c.status !== 3" label="垃圾" icon="i-tabler-alert-triangle" size="xs" color="warning" variant="soft" :loading="busy === c.id" @click="setStatus(c.id, 3)" />
            <UButton label="删除" icon="i-tabler-trash" size="xs" color="error" variant="ghost" :loading="busy === c.id" @click="askRemove(c)" />
          </div>
          </div>
        </div>
      </div>
    </div>

    <ManageCollectionDock v-if="items.length" label="评论选择、批量审核与分页">
      <template #selection>
        <ManagePageSelection :model-value="isPageSelected" :indeterminate="isPageIndeterminate" @update:model-value="togglePage" />
        <template v-if="selectionCount">
          <span>已选 {{ selectionCount }}</span>
          <USelect
            v-model="batchAction"
            :items="[{ label: '通过', value: '1' }, { label: '标记垃圾', value: '3' }, { label: '移入回收站', value: '4' }]"
            value-key="value"
            placeholder="批量操作"
            size="sm"
            class="w-36"
          />
          <UButton label="应用" color="primary" variant="soft" size="sm" :loading="batchRunning" :disabled="!batchAction" @click="applyBatch" />
          <UButton label="清除" color="neutral" variant="ghost" size="sm" :disabled="batchRunning" @click="clearSelection(); batchResult = null" />
        </template>
        <span v-else class="text-xs">共 {{ total }} 条评论</span>
        <span v-if="batchResult" class="text-xs" :class="batchResult.failed ? 'text-warning' : 'text-success'">
          成功 {{ batchResult.success }}<template v-if="batchResult.failed">，失败 {{ batchResult.failed }}</template>
        </span>
      </template>
      <template #pagination>
        <USelect v-model="size" :items="pageSizeItems" value-key="value" size="sm" class="w-24" />
        <ManagePagination v-model="page" :total-pages="totalPages" class="!mt-0" />
      </template>
    </ManageCollectionDock>

    <UModal v-model:open="showDelete" title="删除评论" :description="`确定删除「${deleteTarget?.authorName || '匿名用户'}」的这条评论?此操作不可撤销。`" :ui="{ footer: 'justify-end' }">
      <template #footer="{ close }">
        <UButton label="取消" color="neutral" variant="outline" @click="close" />
        <UButton label="删除" icon="i-tabler-trash" color="error" :loading="busy === deleteTarget?.id" @click="confirmRemove" />
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { ManageHeader, SkeletonList, ManageEmpty, ManageTabs, ManagePagination, ManagePageFooter } from '@platform/ui/components'
import type { CommentAdminView, MyComments } from '~/types'

// Author moderation console: comments on my posts, filterable by status. Anonymous
// comments arrive as 待审核 (pending); approving makes them public. Auth-gated;
// the list needs the author's Bearer (BFF-injected), so client-only.
definePageMeta({ layout: 'manage', middleware: 'auth' })
useSeoMeta({ title: '评论 · 控制台' })

const { call } = useApi()
const { isOwner } = useMe()
const toast = useToast()

// string keys for the shared ManageTabs (v-model is a string); converted to the
// numeric status the API expects. 0 = 全部.
const tabs = [
  { key: '2', label: '待审核' },
  { key: '1', label: '已通过' },
  { key: '3', label: '垃圾' },
  { key: '4', label: '回收站' },
  { key: '0', label: '全部' }
]
const status = ref('2')
const page = ref(1)
const size = 20

const items = ref<CommentAdminView[]>([])
const total = ref(0)
const loading = ref(true)
const busy = ref('')
const showDelete = ref(false)
const deleteTarget = ref<CommentAdminView | null>(null)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)))

async function load() {
  loading.value = true
  try {
    const r = await call<MyComments>('/api/v1/comments/mine', { query: { status: Number(status.value), page: page.value, size } })
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
watch(status, () => { page.value = 1; load() })
watch(page, load)

async function setStatus(id: string, s: number, msg: string) {
  busy.value = id
  try {
    await call(`/api/v1/comments/${id}`, { method: 'PATCH', body: { status: s } })
    toast.add({ title: msg, color: 'success', icon: 'i-tabler-check' })
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
    toast.add({ title: '已删除', color: 'success', icon: 'i-tabler-trash' })
    await load()
  } catch (e: any) {
    toast.add({ title: '删除失败', description: e?.data?.message, color: 'error' })
  } finally {
    busy.value = ''
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

    <SkeletonList v-if="showSkeleton" :rows="8" />

    <ManageEmpty v-else-if="!items.length" icon="i-tabler-message-2" text="这里没有评论" />

    <div v-else class="blog-manage-panel divide-y divide-default overflow-hidden rounded-xl">
      <div v-for="c in items" :key="c.id" class="blog-manage-row flex gap-3 px-4 py-4">
        <UAvatar :text="authorInitial(c.authorName)" size="sm" class="mt-0.5 shrink-0" />
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2 text-sm">
            <span class="font-medium text-highlighted">{{ c.authorName }}</span>
            <UBadge v-if="c.userId" label="会员" color="primary" variant="subtle" size="sm" />
            <UBadge :color="meta(c.status).color" :icon="meta(c.status).icon" :label="meta(c.status).label" variant="subtle" size="sm" />
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
            <UButton v-if="c.status !== 1" label="通过" icon="i-tabler-check" size="xs" color="success" variant="soft" :loading="busy === c.id" @click="setStatus(c.id, 1, '已通过')" />
            <UButton v-if="c.status !== 3" label="垃圾" icon="i-tabler-alert-triangle" size="xs" color="warning" variant="soft" :loading="busy === c.id" @click="setStatus(c.id, 3, '已标记垃圾')" />
            <UButton label="删除" icon="i-tabler-trash" size="xs" color="error" variant="ghost" :loading="busy === c.id" @click="askRemove(c)" />
          </div>
        </div>
      </div>
    </div>

    <ManagePageFooter v-if="items.length">
      <template #left><span class="text-xs">共 {{ total }} 条评论</span></template>
      <template #right><ManagePagination v-model="page" :total-pages="totalPages" class="!mt-0" /></template>
    </ManagePageFooter>

    <UModal v-model:open="showDelete" title="删除评论" :description="`确定删除「${deleteTarget?.authorName || '匿名用户'}」的这条评论?此操作不可撤销。`" :ui="{ footer: 'justify-end' }">
      <template #footer="{ close }">
        <UButton label="取消" color="neutral" variant="outline" @click="close" />
        <UButton label="删除" icon="i-tabler-trash" color="error" :loading="busy === deleteTarget?.id" @click="confirmRemove" />
      </template>
    </UModal>
  </div>
</template>

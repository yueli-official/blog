<script setup lang="ts">
import { ManageHeader, ManageTabs, ManageEmpty, ManagePagination, ManagePageFooter, SkeletonList } from '@platform/manage/components'
import type { AdminAuthorList, AdminAuthorView } from '~/types'

// Authors: the blog's writers, split by status — 现有作者 (approved) / 待审核
// (applications). The site 站长 (owner) is config-set and shown read-only.
// Layout mirrors the taxonomy pages (clean list rows).
definePageMeta({ layout: 'manage', middleware: 'auth' })
useSeoMeta({ title: '作者 · 控制台' })
const { user } = useAuth()
const { isOwner, pending: mePending } = useMe()
const isSelf = (a: AdminAuthorView) => a.id === user.value?.sub
const { call } = useApi()
const toast = useToast()

const mounted = ref(false)
onMounted(() => { mounted.value = true })
const { data, pending: loading, refresh } = await useAsyncData(
  'admin-authors',
  () => call<AdminAuthorList>('/api/v1/admin/authors'),
  { server: false, default: () => ({ authors: [] as AdminAuthorView[] }) }
)
const showSkeleton = useMinLoading(computed(() => !mounted.value || mePending.value || loading.value))
const authors = computed(() => data.value?.authors ?? [])
const active = computed(() => authors.value.filter(a => a.status !== 'pending'))
const pending = computed(() => authors.value.filter(a => a.status === 'pending'))

const tab = ref<'active' | 'pending'>('active')
const tabs = computed(() => [
  { key: 'active', label: '现有作者', count: active.value.length },
  { key: 'pending', label: '待审核', count: pending.value.length }
])

// client-side paging over the active list (small roster; no server paging).
const page = ref(1)
const SIZE = 15
watch(tab, () => { page.value = 1 })
const totalPages = computed(() => Math.max(1, Math.ceil(active.value.length / SIZE)))
const activePaged = computed(() => active.value.slice((page.value - 1) * SIZE, page.value * SIZE))

const saving = ref('')
async function act(a: AdminAuthorView, fn: () => Promise<unknown>, okMsg: string) {
  saving.value = a.id
  try {
    await fn()
    toast.add({ title: okMsg, color: 'success', icon: 'i-tabler-check' })
    await refresh()
  } catch (e) {
    toast.add({ title: '操作失败', description: (e as Error)?.message, color: 'error', icon: 'i-tabler-alert-triangle' })
  } finally {
    saving.value = ''
  }
}
const approve = (a: AdminAuthorView) => act(a, () => call(`/api/v1/admin/authors/${a.id}/approve`, { method: 'POST', body: {} }), '已通过')
const remove = (a: AdminAuthorView) => act(a, () => call(`/api/v1/admin/authors/${a.id}`, { method: 'DELETE' }), a.status === 'pending' ? '已拒绝' : '已移除')
const nameOf = (a: AdminAuthorView) => a.displayName || a.id.slice(0, 8)

// Removing an existing author is destructive — confirm first. (Rejecting a
// pending application is low-risk and stays a direct action.)
const confirmRemove = ref<AdminAuthorView | null>(null)
const showRemove = computed({
  get: () => !!confirmRemove.value,
  set: v => { if (!v) confirmRemove.value = null }
})
async function doRemove() {
  const a = confirmRemove.value
  if (!a) return
  await remove(a)
  confirmRemove.value = null
}
</script>

<template>
  <div>
    <ManageHeader title="作者">
      <template #subtitle>管理本站作者。任何人发文需先「申请成为作者」,在此审批通过即成为作者。</template>
    </ManageHeader>

    <SkeletonList v-if="showSkeleton" :rows="6" />

    <div v-else-if="!isOwner" class="blog-manage-panel rounded-2xl border-dashed py-16 text-center text-muted">
      <UIcon name="i-tabler-lock" class="mx-auto size-8 text-muted" />
      <p class="mt-3 text-sm">仅站长可管理作者</p>
    </div>

    <template v-else>
      <ManageTabs v-model="tab" :items="tabs" class="mb-4" />

      <!-- 现有作者 -->
      <template v-if="tab === 'active'">
        <ManageEmpty v-if="!active.length" icon="i-tabler-users" text="还没有作者" />
        <div v-else class="blog-manage-panel divide-y divide-default overflow-hidden rounded-2xl">
          <div v-for="a in activePaged" :key="a.id" class="blog-manage-row group flex items-center gap-3 px-4 py-3">
            <UAvatar :text="nameOf(a).charAt(0).toUpperCase()" size="sm" class="shrink-0" />
            <div class="min-w-0 flex-1">
              <NuxtLink :to="`/author/${a.id}`" target="_blank" class="font-medium text-highlighted transition hover:text-primary">{{ nameOf(a) }}</NuxtLink>
              <p class="font-mono text-xs text-dimmed">{{ a.id.slice(0, 8) }}</p>
            </div>
            <UBadge :label="`${a.postCount} 篇`" color="neutral" variant="subtle" size="sm" class="shrink-0" />
            <UBadge v-if="a.owner" label="站长" color="primary" variant="subtle" icon="i-tabler-shield-check" class="shrink-0" />
            <UBadge v-if="isSelf(a)" label="你自己" color="neutral" variant="soft" class="shrink-0" />
            <UButton
              v-if="!a.owner && !isSelf(a)"
              icon="i-tabler-user-minus"
              color="error"
              variant="ghost"
              size="sm"
              square
              aria-label="移除作者"
              class="shrink-0 opacity-60 transition group-hover:opacity-100"
              :loading="saving === a.id"
              @click="() => { confirmRemove = a }"
            />
          </div>
        </div>
        <ManagePageFooter v-if="active.length">
          <template #left><span class="text-xs">共 {{ active.length }} 位作者</span></template>
          <template #right><ManagePagination v-model="page" :total-pages="totalPages" class="!mt-0" /></template>
        </ManagePageFooter>
      </template>

      <!-- 待审核 -->
      <template v-else>
        <ManageEmpty v-if="!pending.length" icon="i-tabler-inbox" text="没有待审核的申请" />
        <div v-else class="blog-manage-panel divide-y divide-default overflow-hidden rounded-2xl">
          <div v-for="a in pending" :key="a.id" class="blog-manage-row flex items-center gap-3 px-4 py-3">
            <UAvatar :text="nameOf(a).charAt(0).toUpperCase()" size="sm" class="shrink-0" />
            <div class="min-w-0 flex-1">
              <p class="font-medium text-highlighted">{{ nameOf(a) }}</p>
              <p class="font-mono text-xs text-dimmed">{{ a.id.slice(0, 8) }} · 申请成为作者</p>
            </div>
            <UButton label="通过" icon="i-tabler-check" size="sm" :loading="saving === a.id" @click="approve(a)" />
            <UButton label="拒绝" icon="i-tabler-x" size="sm" color="error" variant="outline" :loading="saving === a.id" @click="remove(a)" />
          </div>
        </div>
      </template>
    </template>

    <UModal
      v-model:open="showRemove"
      title="移除作者"
      :description="`确定移除作者「${confirmRemove ? nameOf(confirmRemove) : ''}」?移除后 TA 将无法再发文。此操作不可撤销。`"
      :ui="{ footer: 'justify-end' }"
    >
      <template #footer>
        <UButton label="取消" color="neutral" variant="outline" @click="() => { showRemove = false }" />
        <UButton label="移除" icon="i-tabler-user-minus" color="error" :loading="!!confirmRemove && saving === confirmRemove.id" @click="doRemove" />
      </template>
    </UModal>
  </div>
</template>

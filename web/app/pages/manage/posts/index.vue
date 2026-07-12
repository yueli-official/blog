<script setup lang="ts">
import {
  ManageActiveFilters,
  ManageCollectionDock,
  ManageCollectionToolbar,
  ManageHeader,
  ManageLifecycleTabs,
  ManagePagination,
  ManageQuickEditField,
  ManageRowShell,
  ManageViewToggle,
  ManageEmpty,
  SkeletonList
} from '@platform/manage/components'
import {
  manageCollectionQueryFingerprint,
  serializeManageCollectionQuery,
  type ManageCollectionDefinition
} from '@platform/manage/collection'
import { useManageCollectionState } from '@platform/manage/use-manage-collection-state'
import { useManageSelection } from '@platform/manage/use-manage-selection'
import type { PostView, MyPosts, ListTaxonomies, AdminAuthorList } from '~/types'

interface BatchResult {
  changed: number
  failures: Array<{ id: string, code: string, message: string }>
}

// Author console: my posts — status tabs + search + category/tag/author filters +
// list/grid view, server-side paginated, with always-on selection feeding a sticky
// footer (select-all + batch actions + pagination). Auth-gated; client-fetched
// (needs the author's BFF-injected Bearer).
definePageMeta({ layout: 'manage', middleware: 'auth' })
useSeoMeta({ title: '文章 · 控制台' })

const { call } = useApi()
const { isOwner, status: authorStatus, pending: mePending } = useMe()
const toast = useToast()
const route = useRoute()
const router = useRouter()

const ALL = '__all__' // USelect items can't carry an empty-string value
const collectionDefinition = {
  resourceKind: 'post',
  statuses: ['', 'published', 'draft', 'archived', 'issues', 'private'],
  views: ['list', 'grid'],
  sortKeys: ['updated'],
  pageSizes: [10, 15, 30, 50],
  defaultStatus: '',
  defaultView: 'list',
  defaultSort: 'updated',
  defaultDirection: 'desc',
  defaultPageSize: 15,
  pagination: 'server',
  selection: 'page',
  filters: ['category', 'tag', 'author', 'flag'],
  quickEditFields: ['title', 'slug', 'status'],
  bulkActions: ['publish', 'draft', 'archive', 'delete']
} as const satisfies ManageCollectionDefinition

const {
  status,
  searchInput,
  q,
  page,
  size,
  view: viewMode,
  state: collectionState,
  filterModel
} = useManageCollectionState({
  definition: collectionDefinition,
  routeQuery: () => route.query,
  replaceQuery: query => router.replace({ query })
})
const flag = filterModel('flag', 'all') // all | pinned | featured

// ── filters: category + tag + (admin) author ──────────────────────────────────
const categoryId = filterModel('category', ALL)
const tagId = filterModel('tag', ALL)
const authorFilter = filterModel('author', 'mine') // admin only
const taxonomyIds = computed(() => [categoryId.value, tagId.value].filter(v => v !== ALL))

const { data, pending, refresh } = await useAsyncData(
  'my-posts',
  () => call<MyPosts>('/api/v1/posts/mine', {
    query: {
      status: status.value || undefined,
      q: q.value.trim() || undefined,
      taxonomyIds: taxonomyIds.value.length ? taxonomyIds.value : undefined,
      pinned: flag.value === 'pinned' ? true : undefined,
      featured: flag.value === 'featured' ? true : undefined,
      all: authorFilter.value === 'all' ? true : undefined,
      authorId: ['mine', 'all'].includes(authorFilter.value) ? undefined : authorFilter.value,
      page: page.value,
      size: size.value
    }
  }),
  { server: false, default: () => ({ items: [] as PostView[], total: 0, page: 1, size: 15, counts: {} as Record<string, number> }), watch: [status, q, page, categoryId, tagId, authorFilter, flag, size] }
)

// Any query transition clears batch feedback; the shared state composable owns
// debounced search, page reset, URL canonicalization and browser history.
watch([status, q, categoryId, tagId, authorFilter, flag, size, page], () => {
  batchAction.value = undefined
  batchResult.value = undefined
})

// taxonomy + author option sources (authors fetched only for admins)
const { data: taxData } = await useAsyncData('manage-taxes', () => call<ListTaxonomies>('/api/v1/taxonomies'), { server: false, default: () => ({ items: [] }) })
const catOptions = computed(() => [{ label: '全部分类', value: ALL }, ...(taxData.value?.items ?? []).filter(t => t.taxonomy === 'category').map(t => ({ label: t.name, value: t.id }))])
const tagOptions = computed(() => [{ label: '全部标签', value: ALL }, ...(taxData.value?.items ?? []).filter(t => t.taxonomy === 'tag').map(t => ({ label: t.name, value: t.id }))])
const { data: authorData } = await useAsyncData('manage-author-roster', () => isOwner.value ? call<AdminAuthorList>('/api/v1/admin/authors') : Promise.resolve({ authors: [] }), { server: false, default: () => ({ authors: [] }) })
const authorOptions = computed(() => [
  { label: '我的文章', value: 'mine' },
  { label: '全部作者', value: 'all' },
  ...(authorData.value?.authors ?? []).map(a => ({ label: a.displayName || a.id.slice(0, 8), value: a.id }))
])
const authorName = (id: string) => (authorData.value?.authors ?? []).find(a => a.id === id)?.displayName || id.slice(0, 8)
const showAuthor = computed(() => isOwner.value && authorFilter.value !== 'mine')

const activeFilters = computed(() => [
  ...(categoryId.value !== ALL ? [{ key: 'category', label: `分类：${catOptions.value.find(item => item.value === categoryId.value)?.label || categoryId.value}` }] : []),
  ...(tagId.value !== ALL ? [{ key: 'tag', label: `标签：${tagOptions.value.find(item => item.value === tagId.value)?.label || tagId.value}` }] : []),
  ...(authorFilter.value !== 'mine' ? [{ key: 'author', label: `作者：${authorOptions.value.find(item => item.value === authorFilter.value)?.label || authorFilter.value}` }] : []),
  ...(flag.value !== 'all' ? [{ key: 'flag', label: flag.value === 'pinned' ? '已置顶' : '已精选' }] : [])
])
function removeActiveFilter(key: string) {
  if (key === 'category') categoryId.value = ALL
  if (key === 'tag') tagId.value = ALL
  if (key === 'author') authorFilter.value = 'mine'
  if (key === 'flag') flag.value = 'all'
}
function clearActiveFilters() {
  categoryId.value = ALL
  tagId.value = ALL
  authorFilter.value = 'mine'
  flag.value = 'all'
}

const pageSizeItems = [10, 15, 30, 50].map(n => ({ label: `${n}/页`, value: n }))
const flagItems = [
  { label: '全部', value: 'all' },
  { label: '置顶', value: 'pinned' },
  { label: '精选', value: 'featured' }
]

const items = computed<PostView[]>(() => data.value?.items ?? [])
const counts = computed<Record<string, number>>(() => data.value?.counts ?? {})
const totalPages = computed(() => Math.max(1, Math.ceil((data.value?.total ?? 0) / size.value)))
const tabs = computed(() => {
  const c = counts.value
  const t = [
    { key: '', label: '全部', count: c.all ?? 0 },
    { key: 'published', label: '已发布', count: c.published ?? 0 },
    { key: 'draft', label: '草稿', count: c.draft ?? 0 },
    { key: 'archived', label: '归档', count: c.archived ?? 0 },
    { key: 'issues', label: '待完善', count: c.issues ?? 0 }
  ]
  if (c.private) t.push({ key: 'private', label: '私密', count: c.private })
  return t
})

// ── write gate: only an approved (active) author — or the owner — can write.
// Becoming an author is done from the public site header (申请成为作者), not here.
const canWrite = computed(() => isOwner.value || authorStatus.value === 'active')

const quickStatusBusy = ref('')
const qualityTarget = ref<PostView>()
const statusBadge: Record<string, { label: string, color: 'success' | 'warning' | 'neutral' }> = {
  published: { label: '已发布', color: 'success' },
  draft: { label: '草稿', color: 'warning' },
  archived: { label: '已归档', color: 'neutral' },
  private: { label: '私密', color: 'neutral' }
}
function quickEditError(error: unknown) {
  const apiError = error as { data?: { code?: string, message?: string } }
  if (apiError.data?.code === 'blog.slug_taken') return 'slug 已被使用，请换一个'
  return apiError.data?.message || '保存失败，请重试'
}
async function quickPatch(post: PostView, body: { title?: string, slug?: string, status?: string }, field: 'title' | 'slug') {
  const res = await call<{ post: PostView }>(`/api/v1/posts/${post.id}`, { method: 'PATCH', body })
  Object.assign(post, res.post)
  return res.post[field]
}
async function quickStatus(post: PostView, next: 'published' | 'draft' | 'archived') {
  quickStatusBusy.value = post.id
  try {
    const res = await call<{ post: PostView }>(`/api/v1/posts/${post.id}`, { method: 'PATCH', body: { status: next } })
    Object.assign(post, res.post)
    await refresh()
  } catch (error) {
    const apiError = error as { data?: { code?: string, message?: string } }
    if (next === 'published' && apiError.data?.code === 'blog.invalid_state') {
      qualityTarget.value = post
      return
    }
    toast.add({ title: '状态更新失败', description: apiError.data?.message || '请重试', color: 'error' })
  } finally {
    quickStatusBusy.value = ''
  }
}
function postQuickActions(post: PostView) {
  return [[
    ...(post.status !== 'published' ? [{ label: '发布', icon: 'i-tabler-world-upload', onSelect: () => quickStatus(post, 'published') }] : []),
    ...(post.status !== 'draft' ? [{ label: '转为草稿', icon: 'i-tabler-file-pencil', onSelect: () => quickStatus(post, 'draft') }] : []),
    ...(post.status !== 'archived' ? [{ label: '归档', icon: 'i-tabler-archive', onSelect: () => quickStatus(post, 'archived') }] : [])
  ]]
}

// ── selection + batch (always-on; driven from the sticky footer) ──────────────
const selectionResetKey = computed(() => manageCollectionQueryFingerprint(serializeManageCollectionQuery(collectionState.value, collectionDefinition)))
const {
  selectedIds,
  selectionCount,
  isPageSelected,
  isPageIndeterminate,
  isSelected,
  toggleOne,
  togglePage,
  clear: clearSelection
} = useManageSelection({
  visibleIds: computed(() => items.value.map(item => item.id)),
  filteredTotal: computed(() => data.value?.total ?? 0),
  resetKey: selectionResetKey
})
const batchAction = ref<string | undefined>(undefined)
const batchBusy = ref(false)
const batchResult = ref<BatchResult | undefined>(undefined)
const batchItems = [
  { label: '发布', value: 'publish' },
  { label: '转草稿', value: 'draft' },
  { label: '归档', value: 'archive' },
  { label: '删除', value: 'delete' }
]
const showBatchConfirm = ref(false)
async function runBatch() {
  if (!batchAction.value || !selectedIds.value.length) return
  batchBusy.value = true
  try {
    const res = await call<BatchResult>('/api/v1/posts/batch', { method: 'POST', body: { ids: selectedIds.value, action: batchAction.value } })
    batchResult.value = res
    const failedIds = new Set(res.failures.map(item => item.id))
    selectedIds.value = selectedIds.value.filter(id => failedIds.has(id))
    if (!res.failures.length) clearSelection()
    batchAction.value = undefined
    showBatchConfirm.value = false
    await refresh()
  } catch (e: any) {
    toast.add({ title: '批量操作失败', description: e?.data?.message || '请重试', color: 'error' })
  } finally {
    batchBusy.value = false
  }
}
// delete is destructive → confirm first; everything else applies straight away.
function applyBatch() {
  if (!batchAction.value || !selectedIds.value.length) return
  if (batchAction.value === 'delete') { showBatchConfirm.value = true; return }
  runBatch()
}

const mounted = ref(false)
onMounted(() => { mounted.value = true })
// gateLoading: still resolving whether the caller can write (useMe) → show a
// full skeleton, not a flash of "你还不是作者". showSkeleton: the list itself is
// (re)loading (filter / page change) → skeleton just the list, keep the toolbar.
const gateLoading = useMinLoading(computed(() => !mounted.value || mePending.value))
const showSkeleton = useMinLoading(computed(() => pending.value))

const showCreate = ref(false)
const title = ref('')
const creating = ref(false)
async function create() {
  if (!title.value.trim()) return
  creating.value = true
  try {
    const res = await call<{ post: PostView }>('/api/v1/posts', { method: 'POST', body: { title: title.value } })
    toast.add({ title: '已创建草稿', color: 'success', icon: 'i-tabler-check' })
    showCreate.value = false
    title.value = ''
    navigateTo(`/manage/posts/${res.post.slug}`)
  } catch (e: any) {
    toast.add({ title: '创建失败', description: e?.data?.message || '请重试', color: 'error' })
  } finally {
    creating.value = false
  }
}

const firstFailedPost = computed(() => {
  const failedID = batchResult.value?.failures[0]?.id
  return failedID ? items.value.find(item => item.id === failedID) : undefined
})

</script>

<template>
  <div>
    <ManageHeader title="文章">
      <template #subtitle>管理你的全部文章</template>
      <template #actions>
        <UButton v-if="canWrite" icon="i-tabler-plus" label="写新文章" @click="() => { showCreate = true }" />
        <UButton v-else-if="authorStatus === 'pending'" icon="i-tabler-clock" label="作者申请审核中" color="neutral" variant="subtle" disabled />
      </template>
    </ManageHeader>

    <SkeletonList v-if="gateLoading" :rows="8" />

    <ManageEmpty
      v-else-if="!canWrite && !items.length"
      :icon="authorStatus === 'pending' ? 'i-tabler-clock' : 'i-tabler-user-edit'"
      :text="authorStatus === 'pending' ? '作者申请审核中,通过后即可写文章' : '你还不是作者 —— 在站点右上角头像菜单「申请成为作者」'"
    />

    <template v-else-if="canWrite || items.length">
      <ManageLifecycleTabs v-model="status" :items="tabs" class="mb-4" />

      <!-- filter bar: search + category + tag + author + page-size + view switch -->
      <ManageCollectionToolbar
        v-model:search="searchInput"
        search-placeholder="搜索标题 / slug…"
        :filter-count="activeFilters.length"
        class="mb-3"
      >
        <template #filters>
          <USelectMenu v-model="categoryId" :items="catOptions" value-key="value" icon="i-tabler-folder" size="sm" class="w-full sm:w-36" :search-input="{ placeholder: '搜索分类…' }" />
          <USelectMenu v-model="tagId" :items="tagOptions" value-key="value" icon="i-tabler-hash" size="sm" class="w-full sm:w-36" :search-input="{ placeholder: '搜索标签…' }" />
          <USelectMenu v-if="isOwner" v-model="authorFilter" :items="authorOptions" value-key="value" icon="i-tabler-user" size="sm" class="w-full sm:w-36" :search-input="{ placeholder: '搜索作者…' }" />
          <USelect v-model="flag" :items="flagItems" icon="i-tabler-flag" size="sm" class="w-full sm:w-28" />
        </template>
        <template #actions>
          <ManageViewToggle
            v-model="viewMode"
            :items="[
              { key: 'list', label: '列表视图', icon: 'i-tabler-list' },
              { key: 'grid', label: '网格视图', icon: 'i-tabler-layout-grid' }
            ]"
          />
        </template>
      </ManageCollectionToolbar>

      <ManageActiveFilters
        :items="activeFilters"
        class="mb-4 px-1"
        @remove="removeActiveFilter"
        @clear="clearActiveFilters"
      />

      <SkeletonList v-if="showSkeleton" :rows="8" />

      <ManageEmpty v-else-if="!items.length" icon="i-tabler-article" :text="(q || taxonomyIds.length || authorFilter !== 'mine' || flag !== 'all') ? '没有匹配的文章' : '这个状态下还没有文章'" />

      <!-- list view -->
      <div v-else-if="viewMode === 'list'" class="blog-manage-panel overflow-hidden rounded-xl">
        <ManageRowShell
          v-for="p in items"
          :key="p.id"
          :selected="isSelected(p.id)"
          :selection-label="`选择文章：${p.title || '无标题'}`"
          @select="toggleOne(p.id)"
        >
          <template #media>
            <div class="size-12 shrink-0 overflow-hidden rounded-lg bg-elevated">
              <img v-if="p.coverUrl" :src="p.coverUrl" :alt="p.title" class="size-full object-cover" >
              <div v-else class="blog-cover-placeholder blog-cover-placeholder--tiny grid size-full place-items-center bg-gradient-to-br from-primary/10 to-transparent"><UIcon name="i-tabler-feather" class="blog-cover-icon size-5 text-primary/30" /></div>
            </div>
          </template>
          <div class="min-w-0">
            <ManageQuickEditField
              :value="p.title"
              label="文章标题"
              placeholder="(无标题)"
              :validate="value => value.trim() ? undefined : '标题不能为空'"
              :error-message="quickEditError"
              :save="value => quickPatch(p, { title: value.trim() }, 'title')"
            />
            <div class="mt-0.5 flex min-w-0 items-center gap-2 text-xs text-muted">
              <span v-if="showAuthor" class="inline-flex items-center gap-1 text-primary"><UIcon name="i-tabler-user" class="size-3" />{{ authorName(p.authorId) }}</span>
              <span v-if="showAuthor" class="text-dimmed">·</span>
              <ManageQuickEditField
                :value="p.slug"
                label="文章 slug"
                placeholder="url-slug"
                monospace
                :validate="value => value.trim() ? undefined : 'slug 不能为空'"
                :error-message="quickEditError"
                :save="value => quickPatch(p, { slug: value.trim() }, 'slug')"
              />
              <span class="text-dimmed">·</span>
              <ClientOnly><span class="shrink-0">{{ rel(p.publishedAt || p.createdAt) }}</span><template #fallback>…</template></ClientOnly>
            </div>
          </div>
          <template #meta>
            <UBadge :label="statusBadge[p.status]?.label || p.status" :color="statusBadge[p.status]?.color || 'neutral'" variant="soft" size="sm" />
          </template>
          <template #actions>
            <UTooltip text="编辑文章">
              <UButton :to="`/manage/posts/${p.slug}`" icon="i-tabler-edit" color="neutral" variant="ghost" size="sm" square :aria-label="`编辑文章：${p.title || '无标题'}`" />
            </UTooltip>
            <UDropdownMenu :items="postQuickActions(p)">
              <UButton icon="i-tabler-dots" color="neutral" variant="ghost" size="sm" square :loading="quickStatusBusy === p.id" :aria-label="`更多文章操作：${p.title || '无标题'}`" />
            </UDropdownMenu>
          </template>
        </ManageRowShell>
      </div>

      <!-- grid view -->
      <div v-else class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
        <div
          v-for="p in items"
          :key="p.id"
          class="blog-manage-card group relative flex cursor-pointer flex-col overflow-hidden rounded-xl"
          :class="isSelected(p.id) ? 'ring-2 ring-primary' : ''"
          @click="navigateTo(`/manage/posts/${p.slug}`)"
        >
          <div class="relative aspect-[16/10] overflow-hidden bg-elevated">
            <img v-if="p.coverUrl" :src="p.coverUrl" :alt="p.title" class="size-full object-cover" >
            <div v-else class="blog-cover-placeholder grid size-full place-items-center bg-gradient-to-br from-primary/10 to-transparent"><UIcon name="i-tabler-feather" class="blog-cover-icon size-7 text-primary/30" /></div>
            <UCheckbox class="absolute left-2 top-2 rounded-md bg-default/85 p-1 backdrop-blur" :model-value="isSelected(p.id)" :aria-label="`选择文章：${p.title || '无标题'}`" @click.stop @update:model-value="toggleOne(p.id)" />
          </div>
          <div class="min-w-0 p-3">
            <h3 class="truncate text-sm font-medium text-highlighted">{{ p.title || '(无标题)' }}</h3>
            <p class="mt-1 truncate text-xs text-dimmed"><ClientOnly>{{ rel(p.publishedAt || p.createdAt) }}<template #fallback>…</template></ClientOnly></p>
          </div>
        </div>
      </div>

      <!-- viewport-fixed collection dock: selection + batch + pagination -->
      <ManageCollectionDock v-if="items.length" label="文章批量操作与分页">
        <template #selection>
          <UCheckbox :model-value="isPageSelected" :indeterminate="isPageIndeterminate" aria-label="选择当前页" @update:model-value="togglePage" />
          <div v-if="batchResult" class="flex flex-wrap items-center gap-2 rounded-lg bg-elevated px-2.5 py-1.5">
            <UIcon :name="batchResult.failures.length ? 'i-tabler-alert-triangle' : 'i-tabler-circle-check'" :class="batchResult.failures.length ? 'text-warning' : 'text-success'" />
            <span class="text-xs text-default">
              已处理 {{ batchResult.changed }} 篇<span v-if="batchResult.failures.length">，{{ batchResult.failures.length }} 篇待完善</span>
            </span>
            <UButton v-if="firstFailedPost" :to="`/manage/posts/${firstFailedPost.slug}`" label="去完善" color="warning" variant="link" size="xs" />
            <UButton icon="i-tabler-x" color="neutral" variant="ghost" size="xs" square aria-label="关闭批量结果" @click="batchResult = undefined" />
          </div>
          <template v-if="selectionCount">
            <span class="text-sm text-default">已选 {{ selectionCount }}</span>
            <span class="h-4 w-px bg-default" />
            <USelect v-model="batchAction" :items="batchItems" placeholder="批量操作" size="sm" class="w-28" />
            <UButton size="sm" color="primary" variant="soft" :disabled="!batchAction" :loading="batchBusy" @click="applyBatch">应用</UButton>
            <UButton size="sm" color="neutral" variant="ghost" @click="clearSelection">取消</UButton>
          </template>
          <span v-else class="text-xs">共 {{ data?.total ?? 0 }} 篇</span>
        </template>
        <template #pagination>
          <USelect v-model="size" :items="pageSizeItems" size="sm" class="w-20" />
          <ManagePagination v-model="page" :total-pages="totalPages" class="!mt-0" />
        </template>
      </ManageCollectionDock>
    </template>

    <UModal v-model:open="showBatchConfirm" title="删除文章" :description="`确定删除选中的 ${selectedIds.length} 篇文章?此操作不可撤销。`" :ui="{ footer: 'justify-end' }">
      <template #footer>
        <UButton label="取消" color="neutral" variant="outline" @click="() => { showBatchConfirm = false }" />
        <UButton label="删除" icon="i-tabler-trash" color="error" :loading="batchBusy" @click="runBatch" />
      </template>
    </UModal>

    <USlideover :open="!!qualityTarget" title="发布前待完善" description="只展示需要处理的问题；完成后可直接返回列表发布。" @update:open="open => { if (!open) qualityTarget = undefined }">
      <template #body>
        <div v-if="qualityTarget" class="space-y-4">
          <div class="rounded-xl border border-warning/30 bg-warning/5 p-4">
            <p class="font-medium text-highlighted">{{ qualityTarget.title || '(无标题)' }}</p>
            <ul class="mt-3 space-y-2 text-sm text-muted">
              <li class="flex items-center gap-2"><UIcon :name="qualityTarget.title.trim() ? 'i-tabler-circle-check' : 'i-tabler-alert-circle'" :class="qualityTarget.title.trim() ? 'text-success' : 'text-warning'" />标题</li>
              <li class="flex items-center gap-2"><UIcon :name="qualityTarget.content.trim() ? 'i-tabler-circle-check' : 'i-tabler-alert-circle'" :class="qualityTarget.content.trim() ? 'text-success' : 'text-warning'" />正文</li>
            </ul>
          </div>
          <UButton :to="`/manage/posts/${qualityTarget.slug}`" label="打开完整编辑器" icon="i-tabler-edit" block />
        </div>
      </template>
    </USlideover>

    <UModal v-model:open="showCreate" title="写新文章" :ui="{ footer: 'justify-end' }">
      <template #body>
        <UFormField label="标题" required>
          <UInput v-model="title" placeholder="给文章起个标题" class="w-full" autofocus @keyup.enter="create" />
        </UFormField>
      </template>
      <template #footer="{ close }">
        <UButton label="取消" color="neutral" variant="outline" @click="close" />
        <UButton label="创建并编辑" icon="i-tabler-arrow-right" trailing :loading="creating" :disabled="!title.trim()" @click="create" />
      </template>
    </UModal>
  </div>
</template>

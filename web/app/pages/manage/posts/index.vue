<script setup lang="ts">
import { ManageHeader, SkeletonList, ManageEmpty, ManageTabs, ManagePagination, ManagePageFooter } from '@platform/manage/components'
import type { PostView, MyPosts, ListTaxonomies, AdminAuthorList } from '~/types'

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

// honor a ?status= deep-link (e.g. from the dashboard's 草稿/已发布 cards)
const status = ref((route.query.status as string) || '')
const q = ref('')
const page = ref(1)
const size = ref(15)
const viewMode = ref<'list' | 'grid'>('list')
const flag = ref('all') // all | 'pinned' | 'featured' (USelect can't carry an empty value)

// ── filters: category + tag + (admin) author ──────────────────────────────────
const ALL = '__all__' // USelect items can't carry an empty-string value
const categoryId = ref(ALL)
const tagId = ref(ALL)
const authorFilter = ref<'mine' | 'all' | string>('mine') // admin only
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
  { server: false, default: () => ({ items: [] as PostView[], total: 0, page: 1, size: 15, counts: {} as Record<string, number> }), watch: [status, page, categoryId, tagId, authorFilter, flag, size] }
)

// debounced search resets paging; any filter / page-size change also resets it;
// any reload clears the current selection.
let searchTimer: ReturnType<typeof setTimeout>
watch(q, () => { clearTimeout(searchTimer); searchTimer = setTimeout(() => { page.value = 1; refresh() }, 300) })
watch([status, categoryId, tagId, authorFilter, flag, size], () => { page.value = 1 })
watch([status, categoryId, tagId, authorFilter, flag, size, page], () => { selectedIds.value = []; batchAction.value = undefined })

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
    { key: 'archived', label: '归档', count: c.archived ?? 0 }
  ]
  if (c.private) t.push({ key: 'private', label: '私密', count: c.private })
  return t
})

// ── write gate: only an approved (active) author — or the owner — can write.
// Becoming an author is done from the public site header (申请成为作者), not here.
const canWrite = computed(() => isOwner.value || authorStatus.value === 'active')

// ── selection + batch (always-on; driven from the sticky footer) ──────────────
const selectedIds = ref<string[]>([])
const batchAction = ref<string | undefined>(undefined)
const batchBusy = ref(false)
const batchItems = [
  { label: '发布', value: 'publish' },
  { label: '转草稿', value: 'draft' },
  { label: '归档', value: 'archive' },
  { label: '删除', value: 'delete' }
]
const isAllSelected = computed(() => items.value.length > 0 && items.value.every(p => selectedIds.value.includes(p.id)))
const isIndeterminate = computed(() => selectedIds.value.length > 0 && !isAllSelected.value)
function toggleSelectAll() { selectedIds.value = isAllSelected.value ? [] : items.value.map(p => p.id) }
function toggleSelect(id: string) {
  selectedIds.value = selectedIds.value.includes(id) ? selectedIds.value.filter(x => x !== id) : [...selectedIds.value, id]
}
const showBatchConfirm = ref(false)
async function runBatch() {
  if (!batchAction.value || !selectedIds.value.length) return
  batchBusy.value = true
  try {
    const res = await call<{ changed: number }>('/api/v1/posts/batch', { method: 'POST', body: { ids: selectedIds.value, action: batchAction.value } })
    toast.add({ title: `已处理 ${res.changed} 篇`, color: 'success', icon: 'i-tabler-check' })
    selectedIds.value = []
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
      <ManageTabs v-model="status" :items="tabs" class="mb-3" />

      <!-- filter bar: search + category + tag + author + page-size + view switch -->
      <div class="blog-manage-panel mb-4 flex flex-wrap items-center gap-2 rounded-lg p-3">
        <UInput v-model="q" icon="i-tabler-search" placeholder="搜索标题 / slug…" size="sm" class="w-full sm:w-52" />
        <USelectMenu v-model="categoryId" :items="catOptions" value-key="value" icon="i-tabler-folder" size="sm" class="w-36" :search-input="{ placeholder: '搜索分类…' }" />
        <USelectMenu v-model="tagId" :items="tagOptions" value-key="value" icon="i-tabler-hash" size="sm" class="w-36" :search-input="{ placeholder: '搜索标签…' }" />
        <USelectMenu v-if="isOwner" v-model="authorFilter" :items="authorOptions" value-key="value" icon="i-tabler-user" size="sm" class="w-36" :search-input="{ placeholder: '搜索作者…' }" />
        <USelect v-model="flag" :items="flagItems" icon="i-tabler-flag" size="sm" class="w-28" />
        <div class="ml-auto flex items-center gap-0.5 rounded-lg bg-default/80 p-0.5 ring-1 ring-default">
          <UButton :variant="viewMode === 'list' ? 'soft' : 'ghost'" :color="viewMode === 'list' ? 'primary' : 'neutral'" size="xs" icon="i-tabler-list" square aria-label="列表视图" @click="() => { viewMode = 'list' }" />
          <UButton :variant="viewMode === 'grid' ? 'soft' : 'ghost'" :color="viewMode === 'grid' ? 'primary' : 'neutral'" size="xs" icon="i-tabler-layout-grid" square aria-label="网格视图" @click="() => { viewMode = 'grid' }" />
        </div>
      </div>

      <SkeletonList v-if="showSkeleton" :rows="8" />

      <ManageEmpty v-else-if="!items.length" icon="i-tabler-article" :text="(q || taxonomyIds.length || authorFilter !== 'mine' || flag !== 'all') ? '没有匹配的文章' : '这个状态下还没有文章'" />

      <!-- list view -->
      <div v-else-if="viewMode === 'list'" class="blog-manage-panel overflow-hidden rounded-xl">
        <div
          v-for="p in items"
          :key="p.id"
          class="blog-manage-row group flex cursor-pointer items-center gap-3 border-b border-default px-4 py-3.5 last:border-b-0"
          :class="selectedIds.includes(p.id) ? 'bg-primary/10' : ''"
          @click="navigateTo(`/manage/posts/${p.slug}`)"
        >
          <button type="button" class="shrink-0" aria-label="选择" @click.stop="toggleSelect(p.id)">
            <UIcon :name="selectedIds.includes(p.id) ? 'i-tabler-square-check-filled' : 'i-tabler-square'" class="size-5" :class="selectedIds.includes(p.id) ? 'text-primary' : 'text-dimmed transition hover:text-muted'" />
          </button>
          <div class="size-12 shrink-0 overflow-hidden rounded-lg bg-elevated">
            <img v-if="p.coverUrl" :src="p.coverUrl" :alt="p.title" class="size-full object-cover" >
            <div v-else class="blog-cover-placeholder blog-cover-placeholder--tiny grid size-full place-items-center bg-gradient-to-br from-primary/10 to-transparent"><UIcon name="i-tabler-feather" class="blog-cover-icon size-5 text-primary/30" /></div>
          </div>
          <div class="min-w-0 flex-1">
            <h3 class="truncate font-medium text-highlighted">{{ p.title || '(无标题)' }}</h3>
            <p class="mt-0.5 flex items-center gap-2 truncate text-xs text-muted">
              <span v-if="showAuthor" class="inline-flex items-center gap-1 text-primary"><UIcon name="i-tabler-user" class="size-3" />{{ authorName(p.authorId) }}</span>
              <span v-if="showAuthor" class="text-dimmed">·</span>
              <span class="font-mono">/{{ p.slug }}</span>
              <span class="text-dimmed">·</span>
              <ClientOnly>{{ rel(p.publishedAt || p.createdAt) }}<template #fallback>…</template></ClientOnly>
            </p>
          </div>
          <UIcon name="i-tabler-chevron-right" class="size-4 shrink-0 text-dimmed transition group-hover:translate-x-0.5 group-hover:text-muted" />
        </div>
      </div>

      <!-- grid view -->
      <div v-else class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
        <div
          v-for="p in items"
          :key="p.id"
          class="blog-manage-card group relative flex cursor-pointer flex-col overflow-hidden rounded-xl"
          :class="selectedIds.includes(p.id) ? 'ring-2 ring-primary' : ''"
          @click="navigateTo(`/manage/posts/${p.slug}`)"
        >
          <div class="relative aspect-[16/10] overflow-hidden bg-elevated">
            <img v-if="p.coverUrl" :src="p.coverUrl" :alt="p.title" class="size-full object-cover" >
            <div v-else class="blog-cover-placeholder grid size-full place-items-center bg-gradient-to-br from-primary/10 to-transparent"><UIcon name="i-tabler-feather" class="blog-cover-icon size-7 text-primary/30" /></div>
            <button type="button" class="absolute left-2 top-2 grid size-6 place-items-center rounded-md bg-default/85 backdrop-blur" aria-label="选择" @click.stop="toggleSelect(p.id)">
              <UIcon :name="selectedIds.includes(p.id) ? 'i-tabler-square-check-filled' : 'i-tabler-square'" class="size-4" :class="selectedIds.includes(p.id) ? 'text-primary' : 'text-dimmed'" />
            </button>
          </div>
          <div class="min-w-0 p-3">
            <h3 class="truncate text-sm font-medium text-highlighted">{{ p.title || '(无标题)' }}</h3>
            <p class="mt-1 truncate text-xs text-dimmed"><ClientOnly>{{ rel(p.publishedAt || p.createdAt) }}<template #fallback>…</template></ClientOnly></p>
          </div>
        </div>
      </div>

      <!-- sticky footer: select-all + batch + pagination -->
      <ManagePageFooter v-if="items.length">
        <template #left>
          <UCheckbox :model-value="isAllSelected" :indeterminate="isIndeterminate" aria-label="全选" @update:model-value="toggleSelectAll" />
          <template v-if="selectedIds.length">
            <span class="text-sm text-default">已选 {{ selectedIds.length }}</span>
            <span class="h-4 w-px bg-default" />
            <USelect v-model="batchAction" :items="batchItems" placeholder="批量操作" size="sm" class="w-28" />
            <UButton size="sm" color="primary" variant="soft" :disabled="!batchAction" :loading="batchBusy" @click="applyBatch">应用</UButton>
            <UButton size="sm" color="neutral" variant="ghost" @click="selectedIds = []">取消</UButton>
          </template>
          <span v-else class="text-xs">共 {{ data?.total ?? 0 }} 篇</span>
        </template>
        <template #right>
          <USelect v-model="size" :items="pageSizeItems" size="sm" class="w-20" />
          <ManagePagination v-model="page" :total-pages="totalPages" class="!mt-0" />
        </template>
      </ManagePageFooter>
    </template>

    <UModal v-model:open="showBatchConfirm" title="删除文章" :description="`确定删除选中的 ${selectedIds.length} 篇文章?此操作不可撤销。`" :ui="{ footer: 'justify-end' }">
      <template #footer>
        <UButton label="取消" color="neutral" variant="outline" @click="() => { showBatchConfirm = false }" />
        <UButton label="删除" icon="i-tabler-trash" color="error" :loading="batchBusy" @click="runBatch" />
      </template>
    </UModal>

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

<script setup lang="ts">
import {
  ActionFeedbackButton,
  ManageCollectionDock,
  ManageCollectionToolbar,
  ManageEmpty,
  ManageHeader,
  ManagePagination,
  SkeletonList
} from '@platform/manage/components'
import type { ManageCollectionDefinition } from '@platform/manage/collection'
import { useManageCollectionState } from '@platform/manage/use-manage-collection-state'
import { useActionFeedback } from '@platform/manage/use-action-feedback'
import { useMinLoading } from '@platform/ui/use-min-loading'
import type { ListTaxonomies, TaxonomyView } from '~/types'

const props = defineProps<{ kind: 'category' | 'tag' }>()
const isCategory = computed(() => props.kind === 'category')
const label = computed(() => isCategory.value ? '分类' : '标签')
const { isOwner } = useMe()
const { call } = useApi()
const route = useRoute()
const router = useRouter()
const ROOT = '__root__'
const ALL = '__all__'

const collectionDefinition = {
  resourceKind: `blog-${props.kind}`,
  statuses: [ALL],
  views: ['list'],
  sortKeys: ['postCount', 'name', 'slug'],
  pageSizes: [15, 30, 60, 120],
  defaultStatus: ALL,
  defaultView: 'list',
  defaultSort: props.kind === 'category' ? 'name' : 'postCount',
  defaultDirection: props.kind === 'category' ? 'asc' : 'desc',
  defaultPageSize: 30,
  pagination: 'client',
  selection: 'page',
  filters: []
} as const satisfies ManageCollectionDefinition
const { searchInput, q, sort, direction, page, size } = useManageCollectionState({
  definition: collectionDefinition,
  routeQuery: computed(() => route.query),
  replaceQuery: query => router.replace({ query })
})

const mounted = ref(false)
onMounted(() => { mounted.value = true })
const { data, pending, refresh, error } = await useAsyncData(
  `gov-${props.kind}`,
  () => call<ListTaxonomies>('/api/v1/taxonomies', { query: { taxonomy: props.kind } }),
  { server: false, default: () => ({ items: [] as TaxonomyView[] }) }
)
const all = computed<TaxonomyView[]>(() => data.value?.items ?? [])
const flat = computed(() => !isCategory.value || Boolean(q.value.trim()))
const comparator = (a: TaxonomyView, b: TaxonomyView) => {
  const multiplier = direction.value === 'asc' ? 1 : -1
  if (sort.value === 'name') return a.name.localeCompare(b.name, 'zh-CN') * multiplier
  if (sort.value === 'slug') return a.slug.localeCompare(b.slug) * multiplier
  return ((a.postCount || 0) - (b.postCount || 0) || a.name.localeCompare(b.name, 'zh-CN')) * multiplier
}
const rows = computed<{ tax: TaxonomyView, depth: number }[]>(() => {
  const keyword = q.value.trim().toLowerCase()
  if (flat.value) {
    const filtered = keyword
      ? all.value.filter(item => `${item.name} ${item.slug} ${item.description || ''}`.toLowerCase().includes(keyword))
      : all.value
    return [...filtered].sort(comparator).map(tax => ({ tax, depth: 0 }))
  }

  const byParent = new Map<string, TaxonomyView[]>()
  for (const item of all.value) {
    const parent = item.parentId || ''
    if (!byParent.has(parent)) byParent.set(parent, [])
    byParent.get(parent)!.push(item)
  }
  const result: { tax: TaxonomyView, depth: number }[] = []
  const walk = (parent: string, depth: number) => {
    for (const item of [...(byParent.get(parent) || [])].sort(comparator)) {
      result.push({ tax: item, depth })
      walk(item.id, depth + 1)
    }
  }
  walk('', 0)
  return result
})
const totalPages = computed(() => flat.value ? Math.max(1, Math.ceil(rows.value.length / size.value)) : 1)
const pagedRows = computed(() => flat.value ? rows.value.slice((page.value - 1) * size.value, page.value * size.value) : rows.value)
const showSkeleton = useMinLoading(computed(() => !mounted.value || pending.value))
const sortItems = [
  { label: '按文章数', value: 'postCount' },
  { label: '按名称', value: 'name' },
  { label: '按 Slug', value: 'slug' }
]
const pageSizeItems = [15, 30, 60, 120].map(value => ({ label: `${value}/页`, value }))

watch(totalPages, (lastPage) => {
  if (page.value > lastPage) page.value = lastPage
}, { flush: 'sync' })

const panel = ref(false)
const current = ref<TaxonomyView | null>(null)
const mergeTarget = ref('')
const operationBusy = ref<'' | 'merge' | 'delete'>('')
const operationError = ref('')
const saveError = ref('')
const confirmingDelete = ref(false)
const form = reactive({ name: '', slug: '', description: '', parentId: ROOT })
const slugTouched = ref(false)
const { status: saveStatus, pending: markSaving, success: markSaved, reset: resetSave } = useActionFeedback()
const parentItems = computed(() => [
  { label: '顶级分类', value: ROOT },
  ...all.value
    .filter(item => item.id !== current.value?.id)
    .map(item => ({ label: item.name, value: item.id }))
])
const mergeTargets = computed(() => all.value
  .filter(item => item.id !== current.value?.id)
  .map(item => ({ label: `${item.name} · ${item.postCount || 0} 篇文章`, value: item.id })))

function clientSlug(value: string) {
  return value.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '')
}

watch(() => form.name, (name) => {
  if (!current.value && !slugTouched.value) form.slug = clientSlug(name)
})

function resetPanelState() {
  resetSave()
  mergeTarget.value = ''
  operationBusy.value = ''
  operationError.value = ''
  saveError.value = ''
  confirmingDelete.value = false
}

function openCreate() {
  resetPanelState()
  current.value = null
  Object.assign(form, { name: '', slug: '', description: '', parentId: ROOT })
  slugTouched.value = false
  panel.value = true
}

function openEdit(item: TaxonomyView) {
  resetPanelState()
  current.value = item
  Object.assign(form, {
    name: item.name,
    slug: item.slug,
    description: item.description || '',
    parentId: item.parentId || ROOT
  })
  slugTouched.value = true
  panel.value = true
}

function toggleDirection() {
  direction.value = direction.value === 'asc' ? 'desc' : 'asc'
}

async function save() {
  if (!form.name.trim()) return
  markSaving()
  saveError.value = ''
  try {
    const body: Record<string, unknown> = {
      name: form.name.trim(),
      slug: form.slug.trim() || undefined,
      description: form.description.trim()
    }
    if (!current.value) body.taxonomy = props.kind
    if (isCategory.value) body.parentId = form.parentId === ROOT ? '' : form.parentId

    const response = current.value
      ? await call<{ taxonomy: TaxonomyView }>(`/api/v1/taxonomies/${current.value.id}`, { method: 'PATCH', body })
      : await call<{ taxonomy: TaxonomyView }>('/api/v1/taxonomies', { method: 'POST', body })
    current.value = response.taxonomy
    Object.assign(form, {
      name: response.taxonomy.name,
      slug: response.taxonomy.slug,
      description: response.taxonomy.description || '',
      parentId: response.taxonomy.parentId || ROOT
    })
    await refresh()
    markSaved()
  } catch (err: any) {
    resetSave()
    saveError.value = err?.data?.message || '保存失败；中文名请确认已手动填写 slug'
  }
}

async function mergeCurrent() {
  if (!current.value || !mergeTarget.value || operationBusy.value) return
  operationBusy.value = 'merge'
  operationError.value = ''
  try {
    await call(`/api/v1/taxonomies/${current.value.id}/merge`, { method: 'POST', body: { targetId: mergeTarget.value } })
    panel.value = false
    await refresh()
  } catch (err: any) {
    operationError.value = err?.data?.message || '合并失败，请重试'
  } finally {
    operationBusy.value = ''
  }
}

async function deleteCurrent() {
  if (!current.value || operationBusy.value) return
  operationBusy.value = 'delete'
  operationError.value = ''
  try {
    await call(`/api/v1/taxonomies/${current.value.id}`, { method: 'DELETE' })
    panel.value = false
    await refresh()
  } catch (err: any) {
    operationError.value = err?.data?.message || '可能仍有子分类'
  } finally {
    operationBusy.value = ''
  }
}

function armDelete() { confirmingDelete.value = true }
function cancelDelete() { confirmingDelete.value = false }
</script>

<template>
  <div class="space-y-5">
    <ManageHeader :title="label">
      <template #subtitle>{{ isCategory ? '维护文章分类层级和公开路径。' : '维护文章标签，合并重复词并保持检索清晰。' }}</template>
      <template #actions>
        <UButton v-if="isOwner" icon="i-tabler-plus" :label="`新建${label}`" @click="openCreate" />
      </template>
    </ManageHeader>

    <SkeletonList v-if="showSkeleton" :rows="6" />
    <UAlert v-else-if="error" color="error" icon="i-tabler-alert-circle" title="加载失败" :description="error.message" />
    <div v-else-if="!isOwner" class="blog-manage-panel rounded-2xl border-dashed py-16 text-center text-muted">
      <UIcon name="i-tabler-lock" class="mx-auto size-8" />
      <p class="mt-2 text-sm">仅站长可治理全站{{ label }}。</p>
    </div>

    <template v-else>
      <ManageCollectionToolbar v-model:search="searchInput" :search-placeholder="`搜索${label}名称、slug 或描述…`">
        <template #filters>
          <USelectMenu v-model="sort" :items="sortItems" value-key="value" icon="i-tabler-arrows-sort" size="sm" />
          <UButton
            :icon="direction === 'asc' ? 'i-tabler-sort-ascending' : 'i-tabler-sort-descending'"
            :label="direction === 'asc' ? '升序' : '降序'"
            color="neutral"
            variant="outline"
            size="sm"
            @click="toggleDirection"
          />
        </template>
      </ManageCollectionToolbar>

      <ManageEmpty v-if="!rows.length" :icon="isCategory ? 'i-tabler-folders' : 'i-tabler-hash'" :text="q ? `没有匹配的${label}` : `还没有${label}`" />

      <div v-else class="blog-manage-panel divide-y divide-default overflow-hidden rounded-2xl">
        <button
          v-for="row in pagedRows"
          :key="row.tax.id"
          type="button"
          class="blog-manage-row group grid w-full grid-cols-[minmax(0,1fr)_2.75rem] items-center gap-3 px-3 py-3 text-left focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-primary sm:px-4 lg:grid-cols-[minmax(14rem,1fr)_8rem_2.75rem]"
          @click="openEdit(row.tax)"
        >
          <span class="flex min-w-0 items-center gap-3" :style="isCategory ? { paddingLeft: `${Math.min(row.depth, 4) * 22}px` } : undefined">
            <UIcon v-if="isCategory && row.depth > 0" name="i-tabler-corner-down-right" class="size-4 shrink-0 text-dimmed" />
            <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-elevated text-dimmed transition group-hover:bg-primary/10 group-hover:text-primary">
              <UIcon :name="isCategory ? 'i-tabler-folder' : 'i-tabler-hash'" class="size-4" />
            </span>
            <span class="min-w-0">
              <span class="line-clamp-1 text-sm font-medium text-highlighted">{{ row.tax.name }}</span>
              <span class="mt-0.5 block line-clamp-1 font-mono text-xs text-muted">/{{ row.tax.slug }}</span>
              <span v-if="row.tax.description" class="mt-1 block line-clamp-1 text-xs text-muted">{{ row.tax.description }}</span>
            </span>
          </span>
          <span class="col-start-1 pl-13 text-xs text-muted lg:col-start-auto lg:pl-0 lg:text-right">
            <span class="font-semibold text-highlighted">{{ row.tax.postCount || 0 }}</span> 篇文章
          </span>
          <span class="row-start-1 col-start-2 grid size-11 place-items-center text-muted lg:row-auto lg:col-start-auto" aria-hidden="true">
            <UIcon name="i-tabler-pencil" class="size-4" />
          </span>
        </button>
      </div>

      <ManageCollectionDock v-if="pagedRows.length" :label="`${label}统计与分页`">
        <template #selection>
          <span>共 {{ rows.length }} 个{{ label }}</span>
          <span v-if="q" class="text-xs text-muted">全部 {{ all.length }} 个</span>
        </template>
        <template #pagination>
          <template v-if="flat">
            <USelect v-model="size" :items="pageSizeItems" value-key="value" size="sm" class="w-24" />
            <ManagePagination v-model="page" :total-pages="totalPages" class="!mt-0" />
          </template>
        </template>
      </ManageCollectionDock>
    </template>

    <USlideover v-model:open="panel" :title="current ? `编辑${label}` : `新建${label}`">
      <template #body>
        <div class="space-y-5">
          <div class="space-y-4">
            <UAlert v-if="saveError" color="error" variant="subtle" icon="i-tabler-alert-circle" title="保存失败" :description="saveError" />
            <UFormField label="名称" required>
              <UInput v-model="form.name" :placeholder="`${label}名称`" class="w-full" autofocus />
            </UFormField>
            <UFormField label="Slug" :help="current ? '改动会影响公开链接。' : '英文名自动生成；中文名请手动填写。'">
              <UInput v-model="form.slug" placeholder="例如 tech" class="w-full" @input="slugTouched = true" />
            </UFormField>
            <UFormField v-if="isCategory" label="父分类">
              <USelectMenu v-model="form.parentId" :items="parentItems" value-key="value" placeholder="选择父分类" :search-input="{ placeholder: '搜索分类…' }" class="w-full" />
            </UFormField>
            <UFormField label="描述">
              <UTextarea v-model="form.description" :rows="3" class="w-full" />
            </UFormField>
            <ActionFeedbackButton block :status="saveStatus" idle-label="保存" pending-label="保存中" success-label="已保存" :disabled="!form.name.trim()" @click="save" />
          </div>

          <template v-if="current">
            <USeparator />
            <section aria-labelledby="blog-taxonomy-management" class="space-y-4">
              <div>
                <h3 id="blog-taxonomy-management" class="text-sm font-medium text-highlighted">管理{{ label }}</h3>
                <p class="mt-1 text-xs text-muted">合并会迁移文章关联；删除有子分类时会被后端拒绝。</p>
              </div>
              <UAlert v-if="operationError" color="error" variant="subtle" icon="i-tabler-alert-circle" title="操作失败" :description="operationError" />
              <div class="rounded-xl border border-default bg-elevated/35 p-3">
                <UFormField :label="`合并到其他${label}`" :help="`当前关联 ${current.postCount || 0} 篇文章`">
                  <div class="grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto]">
                    <USelectMenu v-model="mergeTarget" :items="mergeTargets" value-key="value" placeholder="选择目标…" :search-input="{ placeholder: `搜索目标${label}…` }" class="w-full" />
                    <UButton label="合并" icon="i-tabler-arrows-join" color="warning" variant="soft" :disabled="!mergeTarget" :loading="operationBusy === 'merge'" @click="mergeCurrent" />
                  </div>
                </UFormField>
              </div>
              <div class="rounded-xl border border-error/25 bg-error/5 p-3">
                <div v-if="!confirmingDelete" class="flex items-center justify-between gap-3">
                  <div class="min-w-0">
                    <p class="text-sm font-medium text-highlighted">删除{{ label }}</p>
                    <p class="mt-1 text-xs text-muted">文章会解除关联；有子分类时后端会拒绝。</p>
                  </div>
                  <UButton label="删除" icon="i-tabler-trash" color="error" variant="soft" size="sm" @click="armDelete" />
                </div>
                <div v-else>
                  <p class="text-sm text-highlighted">确定删除「{{ current.name }}」？此操作不可恢复。</p>
                  <div class="mt-3 flex justify-end gap-2">
                    <UButton label="取消" color="neutral" variant="ghost" size="sm" @click="cancelDelete" />
                    <UButton label="确认删除" icon="i-tabler-trash" color="error" size="sm" :loading="operationBusy === 'delete'" @click="deleteCurrent" />
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

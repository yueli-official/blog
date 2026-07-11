<script setup lang="ts">
import { ManageHeader, SkeletonList, ManageEmpty, ManagePageFooter, ManagePagination } from '@platform/ui/components'
import type { ListTaxonomies, TaxonomyView } from '~/types'

// Governance for ONE taxonomy kind (category or tag). Clean clickable list (a
// tree for categories) → a right-side slideover handles create + edit + the
// rarer merge / (destructive) delete, so a row is just "click to manage" instead
// of a strip of equal-weight icons. Admin-gated. Both /manage/categories and
// /manage/tags render this.
const props = defineProps<{ kind: 'category' | 'tag' }>()
const isCategory = computed(() => props.kind === 'category')
const label = computed(() => (isCategory.value ? '分类' : '标签'))

const { isOwner } = useMe()
const { call } = useApi()
const toast = useToast()
const ROOT = '__root__' // USelect can't carry an empty-string value

const mounted = ref(false)
onMounted(() => { mounted.value = true })

const { data, pending, refresh } = await useAsyncData(
  `gov-${props.kind}`,
  () => call<ListTaxonomies>('/api/v1/taxonomies', { query: { taxonomy: props.kind } }),
  { server: false, default: () => ({ items: [] as TaxonomyView[] }) }
)
const all = computed<TaxonomyView[]>(() => data.value?.items ?? [])
const showSkeleton = useMinLoading(computed(() => !mounted.value || pending.value))

// search (filter by name or slug)
const q = ref('')
const filtered = computed(() => {
  const kw = q.value.trim().toLowerCase()
  return kw ? all.value.filter(t => t.name.toLowerCase().includes(kw) || t.slug.toLowerCase().includes(kw)) : all.value
})
// Flat list (tags, or any kind while searching) vs the category tree. A flat
// list can run long (hundreds of tags), so it paginates client-side; the tree
// stays whole (slicing would orphan children, and category trees are small).
const flat = computed(() => !isCategory.value || !!q.value.trim())
const rows = computed<{ tax: TaxonomyView, depth: number }[]>(() => {
  if (flat.value) return [...filtered.value].sort((a, b) => b.postCount - a.postCount).map(t => ({ tax: t, depth: 0 }))
  const byParent = new Map<string, TaxonomyView[]>()
  for (const c of all.value) {
    const k = c.parentId || ''
    if (!byParent.has(k)) byParent.set(k, [])
    byParent.get(k)!.push(c)
  }
  const out: { tax: TaxonomyView, depth: number }[] = []
  const walk = (parent: string, depth: number) => {
    for (const c of byParent.get(parent) || []) { out.push({ tax: c, depth }); walk(c.id, depth + 1) }
  }
  walk('', 0)
  return out
})

// client-side pagination over the flat list (search stays instant over the full set).
const PAGE_SIZE = 30
const page = ref(1)
const totalPages = computed(() => Math.max(1, Math.ceil(rows.value.length / PAGE_SIZE)))
const pagedRows = computed(() => flat.value ? rows.value.slice((page.value - 1) * PAGE_SIZE, page.value * PAGE_SIZE) : rows.value)
watch([q, () => props.kind], () => { page.value = 1 })
watch(totalPages, (tp) => { if (page.value > tp) page.value = tp })

function parentItems(excludeId?: string) {
  return [{ label: '（顶级分类）', value: ROOT }, ...all.value.filter(c => c.id !== excludeId).map(c => ({ label: c.name, value: c.id }))]
}

// ── one slideover for create + edit ─────────────────────────────────────────────
const panel = ref(false)
const current = ref<TaxonomyView | null>(null) // null = create mode
const isEdit = computed(() => !!current.value)
const form = reactive({ name: '', slug: '', description: '', parentId: ROOT })
const slugTouched = ref(false)
const saving = ref(false)
// slug auto-suggest from ascii names (create only); Chinese names need a manual slug.
function clientSlug(s: string) { return s.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '') }
watch(() => form.name, (n) => { if (!isEdit.value && !slugTouched.value) form.slug = clientSlug(n) })

function resetSecondary() { confirmingDelete.value = false; mergeTarget.value = '' }
function openCreate() {
  current.value = null
  form.name = ''; form.slug = ''; form.description = ''; form.parentId = ROOT
  slugTouched.value = false
  resetSecondary()
  panel.value = true
}
function openEdit(t: TaxonomyView) {
  current.value = t
  form.name = t.name; form.slug = t.slug; form.description = t.description || ''; form.parentId = t.parentId || ROOT
  slugTouched.value = true // don't auto-rewrite an existing slug while typing the name
  resetSecondary()
  panel.value = true
}
async function save() {
  if (!form.name.trim()) return
  saving.value = true
  try {
    if (current.value) {
      const body: Record<string, unknown> = { name: form.name.trim(), slug: form.slug.trim(), description: form.description }
      if (isCategory.value) body.parentId = form.parentId === ROOT ? '' : form.parentId
      await call(`/api/v1/taxonomies/${current.value.id}`, { method: 'PATCH', body })
      toast.add({ title: '已更新', color: 'success', icon: 'i-tabler-check' })
    } else {
      const body: Record<string, unknown> = { name: form.name.trim(), taxonomy: props.kind, slug: form.slug.trim() || undefined, description: form.description }
      if (isCategory.value && form.parentId !== ROOT) body.parentId = form.parentId
      await call('/api/v1/taxonomies', { method: 'POST', body })
      toast.add({ title: `已创建${label.value}`, color: 'success', icon: 'i-tabler-check' })
    }
    panel.value = false
    await refresh()
  } catch (e: any) {
    toast.add({ title: current.value ? '更新失败' : '创建失败', description: e?.data?.message || '中文名需手动填写 slug', color: 'error' })
  } finally {
    saving.value = false
  }
}

// ── merge (inside the panel, edit mode) ─────────────────────────────────────────
const mergeTarget = ref('')
const mergingBusy = ref(false)
const mergeTargets = computed(() => current.value
  ? all.value.filter(t => t.id !== current.value!.id).map(t => ({ label: t.name, value: t.id }))
  : [])
async function doMerge() {
  if (!current.value || !mergeTarget.value) return
  mergingBusy.value = true
  try {
    await call(`/api/v1/taxonomies/${current.value.id}/merge`, { method: 'POST', body: { targetId: mergeTarget.value } })
    toast.add({ title: '已合并', color: 'success', icon: 'i-tabler-check' })
    panel.value = false
    await refresh()
  } catch (e: any) {
    toast.add({ title: '合并失败', description: e?.data?.message || '请重试', color: 'error' })
  } finally {
    mergingBusy.value = false
  }
}

// ── delete (inside the panel, inline confirm — destructive) ─────────────────────
const confirmingDelete = ref(false)
const deletingBusy = ref(false)
async function doDelete() {
  if (!current.value) return
  deletingBusy.value = true
  try {
    await call(`/api/v1/taxonomies/${current.value.id}`, { method: 'DELETE' })
    toast.add({ title: `已删除「${current.value.name}」`, color: 'success', icon: 'i-tabler-check' })
    panel.value = false
    await refresh()
  } catch (e: any) {
    toast.add({ title: '删除失败', description: e?.data?.message || '可能仍有子分类', color: 'error' })
  } finally {
    deletingBusy.value = false
  }
}

const mergeHelp = computed(() => `本${label.value}下的文章改挂到目标,然后本${label.value}被删除。`)
</script>

<template>
  <div>
    <ManageHeader :title="label">
      <template #subtitle>
        共 {{ all.length }} 个{{ label }}{{ isCategory ? ',点任一行编辑名称 / slug / 父级,或合并、删除' : ',点任一行编辑名称 / slug,或合并、删除' }}
      </template>
      <template #actions>
        <UButton v-if="isOwner" icon="i-tabler-plus" :label="`新建${label}`" @click="openCreate" />
      </template>
    </ManageHeader>

    <SkeletonList v-if="showSkeleton" :rows="6" />

    <div v-else-if="!isOwner" class="blog-manage-panel rounded-2xl border-dashed py-16 text-center text-muted">
      <UIcon name="i-tabler-lock" class="mx-auto size-8" />
      <p class="mt-2 text-sm">仅站长可治理全站{{ label }}。</p>
    </div>

    <template v-else>
      <UInput v-model="q" icon="i-tabler-search" :placeholder="`搜索${label}名称 / slug…`" size="sm" class="mb-4 w-full sm:w-72" />

      <ManageEmpty v-if="!rows.length" :icon="isCategory ? 'i-tabler-folders' : 'i-tabler-hash'" :text="q ? `没有匹配「${q}」的${label}` : `还没有${label},点右上角新建`" />

      <div v-else class="blog-manage-panel divide-y divide-default overflow-hidden rounded-2xl">
        <button
          v-for="row in pagedRows"
          :key="row.tax.id"
          type="button"
          class="blog-manage-row group flex w-full items-center gap-3 px-4 py-3 text-left"
          @click="openEdit(row.tax)"
        >
          <div class="flex min-w-0 flex-1 items-center gap-2.5" :style="isCategory ? { paddingLeft: row.depth * 22 + 'px' } : {}">
            <UIcon v-if="isCategory && row.depth > 0" name="i-tabler-corner-down-right" class="size-4 shrink-0 text-dimmed/50" />
            <span class="grid size-7 shrink-0 place-items-center rounded-lg bg-elevated text-dimmed transition group-hover:bg-primary/10 group-hover:text-primary">
              <UIcon :name="isCategory ? 'i-tabler-folder' : 'i-tabler-hash'" class="size-4" />
            </span>
            <span class="truncate text-sm font-medium text-highlighted">{{ row.tax.name }}</span>
            <span class="shrink-0 font-mono text-xs text-dimmed">/{{ row.tax.slug }}</span>
          </div>
          <UBadge :label="`${row.tax.postCount} 篇`" color="neutral" variant="subtle" size="sm" class="shrink-0" />
          <UIcon name="i-tabler-chevron-right" class="size-4 shrink-0 text-dimmed transition group-hover:translate-x-0.5 group-hover:text-muted" />
        </button>
      </div>

      <ManagePageFooter v-if="rows.length">
        <template #left>
          <span class="text-xs">共 {{ rows.length }} 个{{ label }}</span>
        </template>
        <template #right>
          <ManagePagination v-if="flat && totalPages > 1" v-model="page" :total-pages="totalPages" class="!mt-0" />
        </template>
      </ManagePageFooter>

      <!-- create + edit slideover (right) -->
      <USlideover v-model:open="panel" :title="isEdit ? `编辑${label}` : `新建${label}`">
        <template #body>
          <div class="space-y-5">
            <div class="space-y-4">
              <UFormField label="名称" required>
                <UInput v-model="form.name" :placeholder="`${label}名称`" class="w-full" autofocus @keyup.enter="save" />
              </UFormField>
              <UFormField label="Slug" :help="isEdit ? '改动会影响该链接。' : 'URL 标识。英文名自动生成;中文名请手动填写(如 tech)。'">
                <UInput v-model="form.slug" placeholder="例如 tech" class="w-full" @input="slugTouched = true" />
              </UFormField>
              <UFormField v-if="isCategory" label="父分类">
                <USelectMenu
                  v-model="form.parentId"
                  :items="parentItems(current?.id)"
                  value-key="value"
                  placeholder="选择父分类"
                  :search-input="{ placeholder: '搜索分类…' }"
                  class="w-full"
                />
              </UFormField>
              <UFormField label="描述">
                <UTextarea v-model="form.description" :rows="3" class="w-full" />
              </UFormField>
            </div>

            <!-- management (edit only): merge + destructive delete -->
            <template v-if="isEdit">
              <USeparator />
              <div class="space-y-4">
                <p class="text-xs font-semibold uppercase tracking-wide text-muted">管理</p>

                <UFormField :label="`合并到其他${label}`" :help="mergeHelp">
                  <div class="flex gap-2">
                    <USelectMenu v-model="mergeTarget" :items="mergeTargets" value-key="value" placeholder="选择目标…" class="flex-1" :search-input="{ placeholder: '搜索…' }" />
                    <UButton label="合并" icon="i-tabler-arrows-join" color="warning" variant="soft" :disabled="!mergeTarget" :loading="mergingBusy" @click="doMerge" />
                  </div>
                </UFormField>

                <div class="rounded-xl border border-error/25 bg-error/5 p-3">
                  <div v-if="!confirmingDelete" class="flex items-center justify-between gap-3">
                    <div class="min-w-0">
                      <p class="text-sm font-medium text-highlighted">删除{{ label }}</p>
                      <p class="mt-0.5 text-xs text-muted">文章会解除关联,但不会被删除。</p>
                    </div>
                    <UButton label="删除" icon="i-tabler-trash" color="error" variant="soft" class="shrink-0" @click="() => { confirmingDelete = true }" />
                  </div>
                  <div v-else>
                    <p class="text-sm text-highlighted">确定删除「{{ current?.name }}」?此操作不可撤销。</p>
                    <div class="mt-3 flex justify-end gap-2">
                      <UButton label="取消" color="neutral" variant="ghost" size="sm" @click="() => { confirmingDelete = false }" />
                      <UButton label="确认删除" icon="i-tabler-trash" color="error" size="sm" :loading="deletingBusy" @click="doDelete" />
                    </div>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </template>
        <template #footer>
          <div class="flex w-full justify-end gap-2">
            <UButton label="取消" color="neutral" variant="outline" @click="() => { panel = false }" />
            <UButton :label="isEdit ? '保存' : `创建${label}`" icon="i-tabler-check" :loading="saving" :disabled="!form.name.trim()" @click="save" />
          </div>
        </template>
      </USlideover>
    </template>
  </div>
</template>

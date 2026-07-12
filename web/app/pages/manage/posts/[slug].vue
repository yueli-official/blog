<script setup lang="ts">
import { ActionFeedbackButton } from '@platform/manage/components'
import { useActionFeedback } from '@platform/manage/use-action-feedback'
import type { PostDetail, ListTaxonomies, TaxonomyView, ListSeries } from '~/types'

// Post editor (author): title + editable slug + rich content, with sidebar panels
// for cover / taxonomies / series / editorial flags, and a collapsible SEO block.
// One primary 保存 persists title+slug+excerpt+content; the sidebar panels apply
// on their own (distinct endpoints). Auth-gated; loaded by slug with the author's
// Bearer so own drafts are visible.
definePageMeta({ layout: 'manage', middleware: 'auth' })

const route = useRoute()
const slug = route.params.slug as string
const { call } = useApi()
const { isOwner } = useMe()
const { uploadCover, uploadImage } = useUpload()
const toast = useToast()

const editorComp = ref<{ markSaved: () => void } | null>(null)
async function uploadInlineImage(file: File): Promise<string> {
  const { url } = await uploadImage(file)
  return url
}

const { data, pending, refresh } = await useAsyncData(
  `edit-${slug}`,
  () => call<PostDetail>(`/api/v1/posts/${slug}`),
  { server: false }
)
const post = computed(() => data.value?.post)
const postId = computed(() => post.value?.id || '')

const mounted = ref(false)
onMounted(() => { mounted.value = true })

// ── editable title / slug / excerpt / content ─────────────────────────────────
const form = reactive({ title: '', slug: '', excerpt: '', content: '' })
// SeoMeta after `form` is declared: its title getter reads form.title, and the
// reactive effect evaluates synchronously during setup — placing it above the
// declaration hits form's temporal dead zone (Cannot access 'form' before init).
useSeoMeta({ title: () => `${form.title || '未命名文章'} · 控制台` })
const slugTouched = ref(false)
watch(data, (d) => {
  if (!d?.post) return
  form.title = d.post.title
  form.slug = d.post.slug
  form.excerpt = d.post.excerpt
  form.content = d.post.content
}, { immediate: true })

// One 保存 persists everything. Each domain is its own endpoint (post core /
// taxonomies / series / flags / seo); all are idempotent, so re-saving is safe.
const { status: saveStatus, pending: markSaving, success: markSaved, reset: resetSave } = useActionFeedback()
async function save() {
  if (!postId.value) return
  markSaving()
  try {
    await call(`/api/v1/posts/${postId.value}`, {
      method: 'PATCH',
      body: { title: form.title, slug: form.slug, excerpt: form.excerpt, content: form.content }
    })
    await call(`/api/v1/posts/${postId.value}/taxonomies`, { method: 'PUT', body: { taxonomyIds: selected.value } })
    await call(`/api/v1/posts/${postId.value}/series`, {
      method: 'PUT',
      body: { seriesId: seriesId.value === NO_SERIES ? '' : seriesId.value, seriesOrder: Number(seriesOrder.value) || 0 }
    })
    if (isOwner.value) {
      await call(`/api/v1/posts/${postId.value}/flags`, { method: 'PUT', body: { pinned: pinned.value, featured: featured.value } })
    }
    await call(`/api/v1/posts/${postId.value}/seo`, { method: 'PUT', body: { ...seo } })
    markSaved()
    editorComp.value?.markSaved()
    await refresh()
  } catch (e: any) {
    resetSave()
    toast.add({ title: '保存失败', description: e?.data?.message || '请重试(slug 可能重复)', color: 'error' })
  }
}

// ── SEO (collapsible advanced block) ──────────────────────────────────────────
const showSeo = ref(false)
const seo = reactive({ metaTitle: '', metaDesc: '', ogTitle: '', ogImage: '', canonicalUrl: '', robots: '' })
watch(data, (d) => { if (d?.seo) Object.assign(seo, d.seo) }, { immediate: true })

// ── cover image ──────────────────────────────────────────────────────────────
const coverInput = ref<HTMLInputElement>()
const coverPct = ref(-1)
async function onPickCover(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  coverPct.value = 0
  try {
    await uploadCover(postId.value, file, (pct) => { coverPct.value = pct })
    await refresh()
  } catch (err: any) {
    toast.add({ title: '封面上传失败', description: err?.message || '请重试', color: 'error' })
  } finally {
    coverPct.value = -1
  }
}

// ── taxonomies (M2): category tree (curated) + tags (folksonomy) ──────────────
const { data: taxes, refresh: refreshTaxes } = await useAsyncData(
  'all-taxes',
  () => call<ListTaxonomies>('/api/v1/taxonomies'),
  { server: false, default: () => ({ items: [] as TaxonomyView[] }) }
)
const selected = ref<string[]>([])
watch(data, (d) => { if (d?.taxonomies) selected.value = d.taxonomies.map(t => t.id) }, { immediate: true })
function toggleTax(id: string) {
  selected.value = selected.value.includes(id) ? selected.value.filter(x => x !== id) : [...selected.value, id]
}
const categories = computed(() => (taxes.value?.items || []).filter(t => t.taxonomy === 'category'))
const tags = computed(() => (taxes.value?.items || []).filter(t => t.taxonomy === 'tag'))
const categoryTree = computed(() => {
  const byParent = new Map<string, TaxonomyView[]>()
  for (const c of categories.value) {
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
const selectedTags = computed(() => tags.value.filter(t => selected.value.includes(t.id)))

// create via the shared modal (supports a custom slug); auto-select on create.
const showCatModal = ref(false)
const showTagModal = ref(false)
const tagModalName = ref('')
function onTaxCreated(t: TaxonomyView) {
  refreshTaxes()
  if (!selected.value.includes(t.id)) selected.value = [...selected.value, t.id]
}
// tag quick-add: an existing name selects it; a new name opens the create modal
// (prefilled) so the author can set a slug.
const newTagName = ref('')
function onTagEnter() {
  const name = newTagName.value.trim()
  if (!name) return
  const existing = tags.value.find(t => t.name === name)
  if (existing) {
    if (!selected.value.includes(existing.id)) selected.value = [...selected.value, existing.id]
  } else {
    tagModalName.value = name
    showTagModal.value = true
  }
  newTagName.value = ''
}

// ── series (M3) ───────────────────────────────────────────────────────────────
const NO_SERIES = '__none__'
const { data: seriesData, refresh: refreshSeries } = await useAsyncData(
  'all-series',
  () => call<ListSeries>('/api/v1/series'),
  { server: false, default: () => ({ items: [] }) }
)
const seriesId = ref(NO_SERIES)
const seriesOrder = ref(0)
watch(data, (d) => {
  seriesId.value = d?.series?.id || NO_SERIES
  seriesOrder.value = d?.post?.seriesOrder ?? 0
}, { immediate: true })
const seriesItems = computed(() => [
  { label: '（无系列）', value: NO_SERIES },
  ...(seriesData.value?.items || []).map(s => ({ label: s.name, value: s.id }))
])
const newSeriesName = ref('')
const creatingSeries = ref(false)
async function createSeries() {
  const name = newSeriesName.value.trim()
  if (!name) return
  creatingSeries.value = true
  try {
    const res = await call<{ series: { id: string } }>('/api/v1/series', { method: 'POST', body: { name } })
    newSeriesName.value = ''
    await refreshSeries()
    seriesId.value = res.series.id
  } catch (e: any) {
    toast.add({ title: '创建系列失败', description: e?.data?.message || '请重试', color: 'error' })
  } finally {
    creatingSeries.value = false
  }
}

// ── editorial flags (M4, superadmin): pinned / featured ───────────────────────
const pinned = ref(false)
const featured = ref(false)
watch(data, (d) => {
  pinned.value = d?.post?.pinned ?? false
  featured.value = d?.post?.featured ?? false
}, { immediate: true })

// ── lifecycle: publish / draft / archive / delete ────────────────────────────
const busy = ref('')
async function setStatus(status: string) {
  busy.value = status
  try {
    await call(`/api/v1/posts/${postId.value}`, { method: 'PATCH', body: { status } })
    await refresh()
  } catch (e: any) {
    toast.add({ title: '操作失败', description: e?.data?.message || '请检查发布条件(标题+正文非空)', color: 'error' })
  } finally {
    busy.value = ''
  }
}
const showDelete = ref(false)
const deleting = ref(false)
async function del() {
  deleting.value = true
  try {
    await call(`/api/v1/posts/${postId.value}`, { method: 'DELETE' })
    navigateTo('/manage/posts')
  } catch (e: any) {
    toast.add({ title: '删除失败', description: e?.data?.message || '请重试', color: 'error' })
    deleting.value = false
  }
}

const statusMeta: Record<string, { label: string, color: 'neutral' | 'success' | 'warning', icon: string }> = {
  draft: { label: '草稿', color: 'neutral', icon: 'i-tabler-pencil' },
  published: { label: '已发布', color: 'success', icon: 'i-tabler-circle-check' },
  private: { label: '私密', color: 'warning', icon: 'i-tabler-lock' },
  archived: { label: '已归档', color: 'warning', icon: 'i-tabler-archive' }
}
const sm = computed(() => statusMeta[post.value?.status || 'draft'] || statusMeta.draft!)

// ── immersive shell: settings drawer + preview + keyboard shortcuts ───────────
// Writing fills the screen; the low-frequency settings (cover / taxonomies /
// series / flags / excerpt / SEO) live in a slide-over, summoned by ⚙ or ⌘/Ctrl+,.
const settingsOpen = ref(false)
function preview() { if (post.value) window.open(`/posts/${post.value.slug}`, '_blank') }
// Title is a borderless textarea that wraps + auto-grows (long titles shouldn't
// clip like a single-line input) — the immersive-editor title pattern.
const titleEl = ref<HTMLTextAreaElement>()
function autoGrowTitle() {
  const el = titleEl.value
  if (!el) return
  el.style.height = '0px'
  el.style.height = `${el.scrollHeight}px`
}
watch(() => form.title, () => nextTick(autoGrowTitle))
onMounted(() => nextTick(autoGrowTitle))
// Both meta (⌘ on macOS) and ctrl (Windows/Linux) so Save works everywhere.
// usingInput keeps them live while the title/editor has focus.
defineShortcuts({
  meta_s: { usingInput: true, handler: () => { save() } },
  ctrl_s: { usingInput: true, handler: () => { save() } },
  'meta_,': { usingInput: true, handler: () => { settingsOpen.value = !settingsOpen.value } },
  'ctrl_,': { usingInput: true, handler: () => { settingsOpen.value = !settingsOpen.value } },
  meta_shift_p: { usingInput: true, handler: () => preview() },
  ctrl_shift_p: { usingInput: true, handler: () => preview() }
})
</script>

<template>
  <div>
    <!-- top bar: back · status · ……(spacer) · preview · settings · publish · more · save -->
    <div class="mb-6 flex flex-wrap items-center gap-2">
      <UButton to="/manage/posts" icon="i-tabler-arrow-left" variant="link" color="neutral" label="文章列表" class="-ml-2" />
      <template v-if="post">
        <UBadge :color="sm.color" :icon="sm.icon" :label="sm.label" variant="subtle" />
        <div class="ml-auto flex items-center gap-1.5">
          <UTooltip text="前台预览 (⌘/Ctrl ⇧ P)">
            <UButton icon="i-tabler-eye" color="neutral" variant="ghost" square aria-label="前台预览" @click="preview" />
          </UTooltip>
          <UTooltip text="文章设置 (⌘/Ctrl ,)">
            <UButton icon="i-tabler-adjustments-horizontal" color="neutral" variant="ghost" square aria-label="文章设置" @click="void (settingsOpen = true)" />
          </UTooltip>
          <UButton
            v-if="post.status !== 'published'"
            label="发布" icon="i-tabler-rocket" color="primary" variant="soft"
            :loading="busy === 'published'" @click="setStatus('published')"
          />
          <UDropdownMenu
            :items="[[
              { label: '删除', icon: 'i-tabler-trash', color: 'error', onSelect: () => { showDelete = true } }
            ]]"
          >
            <UButton icon="i-tabler-dots-vertical" color="neutral" variant="ghost" square />
          </UDropdownMenu>
          <ActionFeedbackButton :status="saveStatus" idle-label="保存" pending-label="保存中" success-label="已保存" @click="save" />
        </div>
      </template>
    </div>

    <div v-if="!mounted || (pending && !post)" class="py-16 text-center text-muted">
      <UIcon name="i-tabler-loader-2" class="size-6 animate-spin" />
    </div>

    <!-- immersive single-column writing area -->
    <div v-else-if="post" class="mx-auto max-w-2xl">
      <div class="mb-4 space-y-2">
        <textarea
          ref="titleEl"
          v-model="form.title"
          rows="1"
          placeholder="未命名文章"
          class="font-display w-full resize-none overflow-hidden bg-transparent text-3xl font-bold leading-tight text-highlighted outline-none placeholder:text-dimmed"
          @input="autoGrowTitle"
        />
        <div class="flex items-center gap-1 text-xs text-muted">
          <UIcon name="i-tabler-link" class="size-3.5 shrink-0 text-dimmed" />
          <span class="shrink-0 font-mono text-dimmed">/posts/</span>
          <input
            v-model="form.slug"
            placeholder="url-slug"
            class="min-w-0 flex-1 bg-transparent font-mono text-dimmed outline-none transition placeholder:text-dimmed focus:text-default"
            @input="slugTouched = true"
          >
        </div>
      </div>

      <ContentEditor
        ref="editorComp"
        v-model="form.content"
        :image-uploader="uploadInlineImage"
        :draft-entity-id="postId"
        :has-initial-content="!!post?.content"
      />
    </div>

    <!-- settings slide-over: the low-frequency post metadata -->
    <USlideover v-model:open="settingsOpen" title="文章设置" description="封面、分类、系列、摘要与 SEO — 改完点保存生效">
      <template #body>
        <div v-if="post" class="space-y-7">
          <!-- cover -->
          <section class="space-y-3">
            <h3 class="text-sm font-semibold text-highlighted">封面</h3>
            <div class="blog-manage-subtle relative aspect-[16/9] w-full overflow-hidden rounded-xl">
              <img v-if="post.coverUrl" :src="post.coverUrl" alt="cover" class="size-full object-cover" >
              <div v-else class="grid size-full place-items-center text-muted"><UIcon name="i-tabler-photo" class="size-8" /></div>
              <div v-if="coverPct >= 0" class="absolute inset-0 grid place-items-center bg-default/70 backdrop-blur-sm">
                <div class="w-3/4">
                  <div class="h-2 overflow-hidden rounded-full bg-accented"><div class="h-full rounded-full bg-primary transition-all" :style="{ width: coverPct + '%' }" /></div>
                  <p class="mt-1 text-center text-xs text-muted">{{ coverPct }}%</p>
                </div>
              </div>
            </div>
            <UButton icon="i-tabler-upload" :label="post.coverUrl ? '更换封面' : '上传封面'" color="neutral" variant="outline" block :disabled="coverPct >= 0" @click="coverInput?.click()" />
            <input ref="coverInput" type="file" accept="image/*" class="hidden" @change="onPickCover" >
          </section>

          <!-- taxonomies -->
          <section class="space-y-4">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold text-highlighted">分类 / 标签</h3>
              <UButton v-if="isOwner" label="新建分类" icon="i-tabler-plus" size="xs" color="neutral" variant="ghost" @click="void (showCatModal = true)" />
            </div>
            <div>
              <p class="mb-1.5 text-xs font-medium text-muted">分类</p>
              <div v-if="categoryTree.length" class="space-y-0.5">
                <button
                  v-for="row in categoryTree"
                  :key="row.tax.id"
                  type="button"
                  class="blog-manage-row flex w-full items-center gap-2 rounded-md px-1.5 py-1 text-left text-sm"
                  :style="{ paddingLeft: 6 + row.depth * 16 + 'px' }"
                  @click="toggleTax(row.tax.id)"
                >
                  <UIcon :name="selected.includes(row.tax.id) ? 'i-tabler-square-check-filled' : 'i-tabler-square'" class="size-4 shrink-0" :class="selected.includes(row.tax.id) ? 'text-primary' : 'text-dimmed'" />
                  <span :class="selected.includes(row.tax.id) ? 'text-highlighted' : 'text-default'">{{ row.tax.name }}</span>
                </button>
              </div>
              <p v-else class="text-xs text-dimmed">还没有分类。</p>
            </div>
            <div class="border-t border-default pt-3">
              <p class="mb-1.5 text-xs font-medium text-muted">标签</p>
              <div v-if="selectedTags.length" class="mb-2 flex flex-wrap gap-1.5">
                <UButton v-for="t in selectedTags" :key="t.id" size="xs" color="primary" variant="soft" :label="t.name" trailing-icon="i-tabler-x" @click="toggleTax(t.id)" />
              </div>
              <UInput v-model="newTagName" size="sm" placeholder="输入标签名回车;新标签可设 slug" class="w-full" @keyup.enter="onTagEnter" />
              <div v-if="tags.length" class="mt-2 flex flex-wrap gap-1">
                <button
                  v-for="t in tags"
                  :key="t.id"
                  type="button"
                  class="rounded-full px-2 py-0.5 text-xs transition"
                  :class="selected.includes(t.id) ? 'bg-primary/15 text-primary' : 'blog-manage-subtle text-muted hover:text-primary'"
                  @click="toggleTax(t.id)"
                >#{{ t.name }}</button>
              </div>
            </div>
          </section>

          <!-- series -->
          <section class="space-y-3">
            <h3 class="text-sm font-semibold text-highlighted">系列</h3>
            <UFormField label="归属系列">
              <USelectMenu
                v-model="seriesId"
                :items="seriesItems"
                value-key="value"
                placeholder="选择系列"
                :search-input="{ placeholder: '搜索系列…' }"
                class="w-full"
              />
            </UFormField>
            <UFormField v-if="seriesId !== NO_SERIES" label="连载序号" help="数字越小越靠前"><UInput v-model="seriesOrder" type="number" class="w-full" /></UFormField>
            <div class="flex gap-1.5">
              <UInput v-model="newSeriesName" size="sm" placeholder="新建系列名" class="flex-1" @keyup.enter="createSeries" />
              <UButton icon="i-tabler-plus" size="sm" color="neutral" variant="outline" :loading="creatingSeries" @click="createSeries" />
            </div>
          </section>

          <!-- editorial flags (owner) -->
          <section v-if="isOwner" class="space-y-3">
            <h3 class="text-sm font-semibold text-highlighted">运营</h3>
            <div class="flex items-center justify-between">
              <span class="flex items-center gap-2 text-sm text-default"><UIcon name="i-tabler-pin" class="size-4 text-muted" />置顶</span>
              <USwitch v-model="pinned" />
            </div>
            <div class="flex items-center justify-between">
              <span class="flex items-center gap-2 text-sm text-default"><UIcon name="i-tabler-sparkles" class="size-4 text-muted" />首页精选</span>
              <USwitch v-model="featured" />
            </div>
            <p class="text-xs text-dimmed">置顶在列表靠前;精选进首页焦点。</p>
          </section>

          <!-- excerpt -->
          <section class="space-y-3">
            <h3 class="text-sm font-semibold text-highlighted">摘要</h3>
            <UTextarea v-model="form.excerpt" :rows="3" class="w-full" placeholder="列表与分享展示;留空则自动取正文开头" />
          </section>

          <!-- lifecycle -->
          <section class="space-y-3 border-t border-default pt-5">
            <h3 class="text-sm font-semibold text-highlighted">生命周期</h3>
            <div class="grid gap-2 sm:grid-cols-2">
              <UButton
                v-if="post.status !== 'draft'"
                label="转回草稿"
                icon="i-tabler-pencil"
                color="neutral"
                variant="outline"
                :loading="busy === 'draft'"
                block
                @click="setStatus('draft')"
              />
              <UButton
                v-if="post.status !== 'archived'"
                label="归档"
                icon="i-tabler-archive"
                color="warning"
                variant="outline"
                :loading="busy === 'archived'"
                block
                @click="setStatus('archived')"
              />
            </div>
            <p class="text-xs text-dimmed">这些是低频状态变更,放在设置里避免误触。</p>
          </section>

          <!-- SEO (collapsible) -->
          <section class="space-y-3">
            <button type="button" class="flex w-full items-center justify-between text-sm font-semibold text-highlighted" @click="showSeo = !showSeo">
              <span class="flex items-center gap-2"><UIcon name="i-tabler-seo" class="size-4 text-muted" />高级 · SEO</span>
              <UIcon name="i-tabler-chevron-down" class="size-4 text-muted transition" :class="showSeo ? 'rotate-180' : ''" />
            </button>
            <div v-if="showSeo" class="space-y-4">
              <UFormField label="Meta 标题"><UInput v-model="seo.metaTitle" class="w-full" /></UFormField>
              <UFormField label="OG 标题"><UInput v-model="seo.ogTitle" class="w-full" /></UFormField>
              <UFormField label="Meta 描述"><UTextarea v-model="seo.metaDesc" :rows="2" class="w-full" /></UFormField>
              <UFormField label="Canonical URL"><UInput v-model="seo.canonicalUrl" class="w-full" placeholder="https://…" /></UFormField>
              <UFormField label="OG 图片 URL"><UInput v-model="seo.ogImage" class="w-full" placeholder="https://…" /></UFormField>
              <UFormField label="Robots"><UInput v-model="seo.robots" class="w-full" placeholder="index, follow" /></UFormField>
            </div>
          </section>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton label="完成" color="neutral" variant="outline" @click="void (settingsOpen = false)" />
          <ActionFeedbackButton :status="saveStatus" idle-label="保存" pending-label="保存中" success-label="已保存" @click="save" />
        </div>
      </template>
    </USlideover>

    <UModal v-model:open="showDelete" title="删除文章" :description="`确定删除「${post?.title || '无标题'}」?此操作不可撤销。`" :ui="{ footer: 'justify-end' }">
      <template #footer="{ close }">
        <UButton label="取消" color="neutral" variant="outline" @click="close" />
        <UButton label="删除" icon="i-tabler-trash" color="error" :loading="deleting" @click="del" />
      </template>
    </UModal>

    <TaxonomyCreateModal kind="category" v-model:open="showCatModal" :categories="categories" @created="onTaxCreated" />
    <TaxonomyCreateModal kind="tag" v-model:open="showTagModal" :default-name="tagModalName" @created="onTaxCreated" />
  </div>
</template>

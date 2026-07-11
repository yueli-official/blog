<script setup lang="ts">
import type { TaxonomyView } from '~/types'

// Reusable "create a category/tag" modal: name + slug (WP-style auto-suggest from
// ascii names; Chinese names need a manual slug) + description + parent (category).
// Emits the created taxonomy so the caller can auto-select / refresh.
const props = defineProps<{ kind: 'category' | 'tag', categories?: TaxonomyView[], defaultName?: string }>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ created: [TaxonomyView] }>()

const { call } = useApi()
const toast = useToast()
const ROOT = '__root__'
const label = computed(() => (props.kind === 'category' ? '分类' : '标签'))

const form = reactive({ name: '', slug: '', description: '', parentId: ROOT })
const slugTouched = ref(false)
const busy = ref(false)
function clientSlug(s: string) { return s.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '') }
watch(() => form.name, (n) => { if (!slugTouched.value) form.slug = clientSlug(n) })
watch(open, (v) => {
  if (!v) return
  form.name = props.defaultName?.trim() || ''
  form.slug = clientSlug(form.name)
  form.description = ''
  form.parentId = ROOT
  slugTouched.value = false
})
const parentItems = computed(() => [{ label: '（顶级分类）', value: ROOT }, ...(props.categories ?? []).map(c => ({ label: c.name, value: c.id }))])

async function submit() {
  if (!form.name.trim()) return
  busy.value = true
  try {
    const body: Record<string, unknown> = {
      name: form.name.trim(), taxonomy: props.kind,
      slug: form.slug.trim() || undefined, description: form.description
    }
    if (props.kind === 'category' && form.parentId !== ROOT) body.parentId = form.parentId
    const res = await call<{ taxonomy: TaxonomyView }>('/api/v1/taxonomies', { method: 'POST', body })
    toast.add({ title: `已创建${label.value}`, color: 'success', icon: 'i-tabler-check' })
    emit('created', res.taxonomy)
    open.value = false
  } catch (e: any) {
    toast.add({ title: '创建失败', description: e?.data?.message || '中文名需手动填写 slug', color: 'error' })
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" :title="`新建${label}`">
    <template #body>
      <div class="space-y-4">
        <UFormField label="名称" required>
          <UInput v-model="form.name" :placeholder="`${label}名称`" class="w-full" autofocus @keyup.enter="submit" />
        </UFormField>
        <UFormField label="Slug" help="URL 标识。英文名自动生成;中文名请手动填写(如 tech)。">
          <UInput v-model="form.slug" placeholder="例如 tech" class="w-full" @input="slugTouched = true" />
        </UFormField>
        <UFormField v-if="kind === 'category'" label="父分类">
          <USelectMenu
            v-model="form.parentId"
            :items="parentItems"
            value-key="value"
            placeholder="选择父分类"
            :search-input="{ placeholder: '搜索分类…' }"
            class="w-full"
          />
        </UFormField>
        <UFormField label="描述">
          <UTextarea v-model="form.description" :rows="2" class="w-full" />
        </UFormField>
      </div>
    </template>
    <template #footer="{ close }">
      <div class="flex justify-end gap-2">
        <UButton label="取消" color="neutral" variant="outline" @click="close" />
        <UButton :label="`创建${label}`" icon="i-tabler-check" :loading="busy" :disabled="!form.name.trim()" @click="submit" />
      </div>
    </template>
  </UModal>
</template>

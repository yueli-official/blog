<script setup lang="ts">
import type { ListTaxonomies, TaxonomyView } from '~/types'

// Tags index (M2): a searchable directory. 热度 mode is a size-encoded cloud;
// A–Z mode groups tags by pinyin initial with a letter-jump rail.
definePageMeta({ width: 'full' })
const { call } = useApi()
const { data } = await useAsyncData(
  'tag-cloud',
  () => call<ListTaxonomies>('/api/v1/taxonomies', { query: { taxonomy: 'tag' } })
)
const all = computed<TaxonomyView[]>(() => data.value?.items ?? [])

const q = ref('')
const sort = ref<'hot' | 'az'>('hot')
const filtered = computed(() => {
  const kw = q.value.trim().toLowerCase()
  return kw ? all.value.filter(t => t.name.toLowerCase().includes(kw)) : all.value
})

// 热度: size-encoded cloud (count → one of five type sizes)
const byHot = computed(() => [...filtered.value].sort((a, b) => b.postCount - a.postCount))
const maxCount = computed(() => Math.max(1, ...all.value.map(t => t.postCount)))
function sizeClass(count: number): string {
  const r = count / maxCount.value
  if (r > 0.8) return 'text-2xl font-semibold'
  if (r > 0.6) return 'text-xl font-medium'
  if (r > 0.4) return 'text-lg'
  if (r > 0.2) return 'text-base'
  return 'text-sm'
}

// A–Z: pinyin initial via Intl.Collator boundary chars (no dependency; consistent
// across SSR/client since both use ICU). zh[i] is the first hanzi of letter[i].
const collator = new Intl.Collator('zh-CN')
const LETTERS = 'ABCDEFGHJKLMNOPQRSTWXYZ'.split('')
const ANCHORS = '阿八嚓哒妸发旮哈讥咔垃痳拏哦妑七然撒塌挖夕压匝'.split('')
function initial(name: string): string {
  const ch = name.charAt(0)
  if (/[a-z]/i.test(ch)) return ch.toUpperCase()
  if (!/[一-龥]/.test(ch)) return '#'
  for (let i = LETTERS.length - 1; i >= 0; i--) {
    if (collator.compare(ch, ANCHORS[i]!) >= 0) return LETTERS[i]!
  }
  return '#'
}
const grouped = computed(() => {
  const map = new Map<string, TaxonomyView[]>()
  for (const t of [...filtered.value].sort((a, b) => collator.compare(a.name, b.name))) {
    const k = initial(t.name)
    ;(map.get(k) ?? map.set(k, []).get(k)!).push(t)
  }
  return [...map.entries()].sort((a, b) => (a[0] === '#' ? 1 : b[0] === '#' ? -1 : a[0] < b[0] ? -1 : 1))
})
function jump(letter: string) {
  document.getElementById(`tg-${letter}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

useSeoMeta({ title: '标签 · 博客' })
</script>

<template>
  <div>
    <PageHero
      eyebrow="Tags"
      title="标签"
      :subtitle="all.length ? `共 ${all.length} 个标签 · 按热度或拼音浏览。` : undefined"
    >
      <ListToolbar v-if="all.length" v-model:q="q" v-model:sort="sort" placeholder="搜索标签" />
    </PageHero>

    <div v-if="!all.length" class="rounded-2xl border border-dashed border-default py-24 text-center">
      <div class="mx-auto grid size-14 place-items-center rounded-2xl bg-primary/10 text-primary"><UIcon name="i-tabler-hash" class="size-7" /></div>
      <p class="mt-4 font-medium text-highlighted">还没有标签</p>
    </div>

    <div v-else-if="!filtered.length" class="py-20 text-center text-muted">
      <p class="text-sm">没有匹配「{{ q }}」的标签</p>
    </div>

    <!-- 热度: size-encoded cloud -->
    <div v-else-if="sort === 'hot'" class="rounded-2xl border border-default p-6 sm:p-8">
      <div class="flex flex-wrap items-baseline gap-x-5 gap-y-4">
        <NuxtLink
          v-for="t in byHot"
          :key="t.id"
          :to="`/tags/${t.slug}`"
          class="group inline-flex items-baseline gap-1 text-muted transition hover:text-primary"
          :class="sizeClass(t.postCount)"
        >
          <span class="text-primary/40 transition group-hover:text-primary/70">#</span>{{ t.name }}
          <span class="text-xs font-normal text-dimmed">{{ t.postCount }}</span>
        </NuxtLink>
      </div>
    </div>

    <!-- A–Z: grouped directory + letter rail -->
    <div v-else>
      <div class="mb-5 flex flex-wrap gap-1">
        <button
          v-for="[letter] in grouped"
          :key="letter"
          type="button"
          class="grid size-7 place-items-center rounded-md text-xs font-semibold text-muted transition hover:bg-primary/10 hover:text-primary"
          @click="jump(letter)"
        >{{ letter }}</button>
      </div>
      <div class="space-y-6">
        <section v-for="[letter, items] in grouped" :id="`tg-${letter}`" :key="letter" class="scroll-mt-20">
          <div class="mb-2 flex items-center gap-3">
            <span class="font-display text-lg font-bold text-primary">{{ letter }}</span>
            <span class="h-px flex-1 bg-default" />
          </div>
          <div class="flex flex-wrap gap-2">
            <NuxtLink
              v-for="t in items"
              :key="t.id"
              :to="`/tags/${t.slug}`"
              class="inline-flex items-baseline gap-1 rounded-full border border-default px-3 py-1 text-sm text-default transition hover:border-primary/40 hover:text-primary"
            >
              <span class="text-dimmed">#</span>{{ t.name }}
              <span class="text-xs text-dimmed">{{ t.postCount }}</span>
            </NuxtLink>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

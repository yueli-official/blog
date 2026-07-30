<script setup lang="ts">
import SkeletonCards from '~/components/SkeletonCards.vue'
import type { HomeConfigResponse, ListPosts, ListTaxonomies, ListSeries, PostView, SeriesView } from '~/types'

// Magazine home (reworked layout): slim masthead → composed hero (one lead +
// a recommended-reading column) → static series grid → latest grid + sidebar.
// Style unchanged (same tokens/colors/fonts); only the composition changed.
definePageMeta({ width: 'full' })
const { call } = useApi()
const taxonomy = ref('')
const page = ref(1)
const size = 12

const { data: cats } = await useAsyncData('home-categories', () => call<ListTaxonomies>('/api/v1/taxonomies', { query: { taxonomy: 'category' } }))
const { data: tagData } = await useAsyncData('home-tags', () => call<ListTaxonomies>('/api/v1/taxonomies', { query: { taxonomy: 'tag' } }))
const { data: homeConfigData } = await useAsyncData('home-config', () => call<HomeConfigResponse>('/api/v1/home'))
if (!homeConfigData.value?.config) {
  throw createError({ statusCode: 500, statusMessage: '博客站点配置尚未初始化' })
}
const { data: featuredData } = await useAsyncData('home-featured', () => call<ListPosts>('/api/v1/posts', { query: { featured: true, size: 8 } }))
const { data: seriesData } = await useAsyncData('home-series', () => call<ListSeries>('/api/v1/series'))
const { data, pending } = await useAsyncData(
  'home-posts',
  () => call<ListPosts>('/api/v1/posts', { query: { taxonomy: taxonomy.value || undefined, page: page.value, size } }),
  { watch: [taxonomy, page] }
)
// discovery widgets (sidebar): 热门 by views, 随机 refreshable.
const { data: popularData } = await useAsyncData('home-popular', () => call<ListPosts>('/api/v1/posts', { query: { sort: 'popular', size: 10 } }))
const { data: randomData, refresh: refreshRandom, pending: randomPending } = await useAsyncData('home-random', () => call<ListPosts>('/api/v1/posts', { query: { sort: 'random', size: 5 } }))
const popular = computed<PostView[]>(() => popularData.value?.items ?? [])
const random = computed<PostView[]>(() => randomData.value?.items ?? [])
const homeConfig = computed(() => homeConfigData.value!.config)
useSeoMeta({
  title: () => homeConfig.value.siteTitle,
  description: () => homeConfig.value.siteDescription,
})

function setCat(v: string) { taxonomy.value = v; page.value = 1 }
const featured = computed<PostView[]>(() => featuredData.value?.items ?? [])
// home shows the richest series (by post count); the rest live on /series via
// the section's "more →" link, so the 2-col grid never grows unbounded.
const HOME_SERIES_LIMIT = 4
const series = computed<SeriesView[]>(() => [...(seriesData.value?.items ?? [])].sort((a, b) => b.postCount - a.postCount).slice(0, HOME_SERIES_LIMIT))
const tags = computed(() => [...(tagData.value?.items ?? [])].sort((a, b) => b.postCount - a.postCount).slice(0, 18))
const posts = computed<PostView[]>(() => data.value?.items ?? [])
const showSkeleton = useMinimumLoading(pending)
const totalPages = computed(() => Math.max(1, Math.ceil((data.value?.total ?? 0) / size)))
const totalPublished = computed(() => data.value?.total ?? posts.value.length)
function readMin(content?: string) { return Math.max(1, Math.ceil((content?.length ?? 0) / 400)) }

// hero composition: featured posts rotate through the lead slot via the dots
// below it; the recommended column is the popular list (minus the featured
// already in the dots) — a fixed curation, independent of the category filter.
const activeFeatured = ref(0)
const lead = computed<PostView | undefined>(() => featured.value[activeFeatured.value] ?? featured.value[0])
const secondary = computed<PostView[]>(() => {
  const exclude = new Set<string>(featured.value.map(f => f.id))
  return popular.value.filter(p => !exclude.has(p.id)).slice(0, 5)
})
// homepage category filter shows the top categories by post count; the full
// list lives on /category (linked after the pills) so many categories don't
// overflow the row.
const HOME_CAT_LIMIT = 8
const homeCats = computed(() => [...(cats.value?.items ?? [])].filter(c => !c.parentId).sort((a, b) => b.postCount - a.postCount).slice(0, HOME_CAT_LIMIT))

// auto-advance the featured lead every 5s (loops); pause on hover, restart on
// manual dot click. Client-only timer — SSR/first paint show featured[0].
let featTimer: ReturnType<typeof setInterval> | null = null
function stopFeat() { if (featTimer) { clearInterval(featTimer); featTimer = null } }
function startFeat() {
  stopFeat()
  if (featured.value.length <= 1) return
  featTimer = setInterval(() => { activeFeatured.value = (activeFeatured.value + 1) % featured.value.length }, 5000)
}
function pickFeat(i: number) { activeFeatured.value = i; startFeat() }
onMounted(startFeat)
onBeforeUnmount(stopFeat)
</script>

<template>
  <div class="space-y-14 sm:space-y-20">
    <!-- ░ masthead (slim) ░ -->
    <header class="grid gap-6 border-b border-default pb-7 lg:grid-cols-[minmax(0,1fr)_320px]">
      <div>
        <div class="flex items-center gap-2 text-primary">
          <span class="h-px w-6 bg-primary/40" />
          <span class="text-xs font-semibold uppercase tracking-[0.16em]">{{ homeConfig.eyebrow }}</span>
        </div>
        <h1 class="font-display mt-3 max-w-3xl text-4xl font-bold leading-[1.05] tracking-tight text-highlighted sm:text-5xl">{{ homeConfig.title }}</h1>
        <p class="mt-4 max-w-2xl text-base leading-7 text-muted">{{ homeConfig.subtitle }}</p>
      </div>
      <div class="grid grid-cols-3 gap-3 text-sm lg:self-end">
        <NuxtLink to="/archive" class="rounded-lg border border-default px-4 py-3 transition hover:border-primary/40 hover:text-primary">
          <span class="block text-xs text-muted">归档</span>
          <span class="font-medium text-default">时间线</span>
        </NuxtLink>
        <NuxtLink to="/series" class="rounded-lg border border-default px-4 py-3 transition hover:border-primary/40 hover:text-primary">
          <span class="block text-xs text-muted">专题</span>
          <span class="font-medium text-default">{{ series.length }}</span>
        </NuxtLink>
        <NuxtLink to="/category" class="rounded-lg border border-default px-4 py-3 transition hover:border-primary/40 hover:text-primary">
          <span class="block text-xs text-muted">文章</span>
          <span class="font-medium text-default">{{ totalPublished }}</span>
        </NuxtLink>
      </div>
    </header>

    <!-- ░ hero: composed front (lead + recommended column) ░ -->
    <section v-if="featured.length" class="grid gap-6 md:h-[450px] md:grid-cols-[minmax(0,1.7fr)_minmax(0,1fr)] md:gap-8">
      <!-- lead + dot switcher -->
      <div v-if="lead" class="flex flex-col gap-4 md:h-full" @mouseenter="stopFeat" @mouseleave="startFeat">
        <NuxtLink
          :to="`/posts/${lead.slug}`"
          class="blog-article-card group flex flex-1 flex-col overflow-hidden rounded-lg border border-default transition hover:border-primary/40 hover:shadow-lg md:min-h-0"
        >
          <Transition
            mode="out-in"
            enter-active-class="transition-opacity duration-200 ease-out motion-reduce:transition-none"
            leave-active-class="transition-opacity duration-200 ease-in motion-reduce:transition-none"
            enter-from-class="opacity-0"
            leave-to-class="opacity-0"
          >
            <div :key="lead.id" class="flex flex-1 flex-col md:min-h-0">
              <div class="relative aspect-[16/9] overflow-hidden md:aspect-auto md:min-h-0 md:flex-1">
                <img v-if="lead.coverUrl" :src="coverThumbUrl(lead)" :alt="lead.title" class="size-full object-cover transition duration-700 group-hover:scale-[1.03]" >
                <div v-else class="blog-cover-placeholder relative size-full bg-gradient-to-br from-primary/15 via-primary/5 to-elevated">
                  <UIcon name="i-tabler-feather" class="blog-cover-icon absolute -bottom-8 -right-6 size-48 text-primary/10" />
                </div>
                <span class="absolute left-4 top-4 inline-flex items-center gap-1 rounded-full bg-default/85 px-3 py-1 text-xs font-semibold text-primary shadow-sm backdrop-blur">
                  <UIcon name="i-tabler-sparkles" class="size-3.5" />精选
                </span>
              </div>
              <div class="flex flex-col p-5 sm:p-6 md:flex-none">
                <h2 class="font-display line-clamp-1 text-xl font-bold leading-tight text-highlighted transition group-hover:text-primary sm:text-2xl">{{ lead.title }}</h2>
                <p class="mt-2 line-clamp-2 min-h-[2lh] text-sm text-muted">{{ lead.excerpt }}</p>
                <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted">
                  <ClientOnly><span>{{ rel(lead.publishedAt || lead.createdAt) }}</span><template #fallback><span /></template></ClientOnly>
                  <span class="flex items-center gap-1"><UIcon name="i-tabler-clock" class="size-3.5" />{{ readMin(lead.content) }} 分钟</span>
                  <span class="flex items-center gap-1"><UIcon name="i-tabler-eye" class="size-3.5" />{{ lead.viewCount ?? 0 }}</span>
                </div>
              </div>
            </div>
          </Transition>
        </NuxtLink>
        <!-- dot switcher (no arrows): click a dot to swap the lead -->
        <div v-if="featured.length > 1" class="flex items-center gap-1 pl-1">
          <button
            v-for="(f, i) in featured"
            :key="f.id"
            type="button"
            :aria-label="`精选第 ${i + 1} 篇`"
            :aria-current="i === activeFeatured"
            class="group grid size-6 place-items-center rounded-full"
            @click="pickFeat(i)"
          >
            <span
              aria-hidden="true"
              class="h-2 rounded-full transition-all duration-300"
              :class="i === activeFeatured ? 'w-6 bg-primary' : 'w-2 bg-primary/40 group-hover:bg-primary/60'"
            />
          </button>
        </div>
      </div>

      <!-- recommended reading -->
      <div class="flex flex-col md:h-full">
        <h2 class="font-display mb-1 flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.14em] text-muted">
          <span class="h-px w-5 bg-primary/50" />推荐阅读
        </h2>
        <ul class="flex flex-1 flex-col divide-y divide-default">
          <li v-for="(p, i) in secondary" :key="p.id" class="md:flex-1">
            <NuxtLink :to="`/posts/${p.slug}`" class="group flex items-center gap-4 py-3 md:h-full">
              <span class="font-display w-7 shrink-0 text-lg font-bold tabular-nums text-primary">{{ String(i + 1).padStart(2, '0') }}</span>
              <div class="min-w-0">
                <h3 class="font-display line-clamp-2 font-semibold leading-snug text-highlighted transition group-hover:text-primary">{{ p.title }}</h3>
                <div class="mt-1.5 flex items-center gap-3 text-xs text-muted">
                  <ClientOnly><span>{{ rel(p.publishedAt || p.createdAt) }}</span><template #fallback><span /></template></ClientOnly>
                  <span class="flex items-center gap-1"><UIcon name="i-tabler-clock" class="size-3.5" />{{ readMin(p.content) }} 分钟</span>
                </div>
              </div>
            </NuxtLink>
          </li>
        </ul>
      </div>
    </section>

    <!-- ░ content: series + latest (left) · discovery rail (right) ░ -->
    <section class="space-y-12 lg:grid lg:gap-10 lg:space-y-0 lg:grid-cols-[minmax(0,1fr)_280px]">
      <!-- left column: series + latest stacked -->
      <div class="min-w-0 space-y-14 sm:space-y-16">
        <!-- series -->
        <div v-if="series.length">
          <div class="mb-5 flex items-center justify-between">
            <h2 class="font-display flex items-center gap-2 text-xl font-semibold text-highlighted">
              <UIcon name="i-tabler-stack-2" class="size-5 text-primary" />专题
            </h2>
            <NuxtLink to="/series" class="group inline-flex items-center gap-0.5 text-xs font-medium text-dimmed transition hover:text-primary">全部系列<UIcon name="i-tabler-chevron-right" class="size-3.5 transition group-hover:translate-x-0.5" /></NuxtLink>
          </div>
          <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
            <article v-for="s in series" :key="s.id" class="flex min-w-0 gap-4 rounded-lg border border-default p-4 transition hover:border-primary/40 hover:shadow-md">
              <NuxtLink :to="`/series/${s.slug}`" :aria-label="`查看专题：${s.name}`" class="relative aspect-square w-24 shrink-0 overflow-hidden rounded-md sm:w-28">
                <img v-if="s.coverUrl" :src="s.coverUrl" :alt="s.name" class="size-full object-cover" >
                <div v-else class="blog-cover-placeholder grid size-full place-items-center bg-gradient-to-br from-primary/20 to-primary/5"><UIcon name="i-tabler-stack-2" class="blog-cover-icon size-9 text-primary/50" /></div>
              </NuxtLink>
              <div class="flex min-w-0 flex-1 flex-col">
                <NuxtLink :to="`/series/${s.slug}`" class="font-display line-clamp-1 font-semibold text-highlighted transition hover:text-primary">{{ s.name }}</NuxtLink>
                <p class="mt-0.5 text-xs text-muted">{{ s.postCount }} 篇连载</p>
                <ul class="mt-2 space-y-1.5">
                  <li v-for="rp in (s.recentPosts || []).slice(0, 2)" :key="rp.id">
                    <NuxtLink :to="`/posts/${rp.slug}`" class="flex items-center gap-1.5 text-sm text-muted transition hover:text-primary">
                      <UIcon name="i-tabler-point-filled" class="size-3 shrink-0 text-primary/40" /><span class="min-w-0 line-clamp-1">{{ rp.title }}</span>
                    </NuxtLink>
                  </li>
                  <li v-if="!(s.recentPosts || []).length" class="text-sm text-dimmed">暂无文章</li>
                </ul>
              </div>
            </article>
          </div>
        </div>

        <!-- latest -->
        <div>
          <div class="mb-5 flex items-center justify-between gap-4">
            <h2 class="font-display text-xl font-semibold text-highlighted">最新</h2>
            <NuxtLink to="/category" class="group inline-flex items-center gap-0.5 text-xs font-medium text-dimmed transition hover:text-primary">全部分类<UIcon name="i-tabler-chevron-right" class="size-3.5 transition group-hover:translate-x-0.5" /></NuxtLink>
          </div>
          <div v-if="cats?.items?.length" class="mb-6 flex flex-wrap items-center gap-2">
            <button class="rounded-full px-3.5 py-1.5 text-sm font-medium transition" :class="taxonomy === '' ? 'bg-primary text-inverted' : 'bg-elevated text-muted hover:text-default'" @click="setCat('')">全部</button>
            <button v-for="c in homeCats" :key="c.id" class="rounded-full px-3.5 py-1.5 text-sm font-medium transition" :class="taxonomy === c.slug ? 'bg-primary text-inverted' : 'bg-elevated text-muted hover:text-default'" @click="setCat(c.slug)">{{ c.name }}</button>
          </div>
          <SkeletonCards v-if="showSkeleton" :count="6" />
          <div v-else-if="!posts.length" class="rounded-lg border border-dashed border-default py-24 text-center">
            <div class="mx-auto grid size-14 place-items-center rounded-lg bg-primary/10 text-primary"><UIcon name="i-tabler-feather" class="size-7" /></div>
            <p class="mt-4 font-medium text-highlighted">这里还没有更多文章</p>
          </div>
          <template v-else>
            <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 sm:gap-6 md:grid-cols-3 lg:grid-cols-2 xl:grid-cols-3">
              <NuxtLink v-for="p in posts" :key="p.id" :to="`/posts/${p.slug}`" class="blog-article-card group flex min-w-0 gap-4 overflow-hidden rounded-lg border border-default transition hover:border-primary/40 hover:shadow-md sm:flex-col sm:gap-0">
                <!-- cover: small thumb (row) on mobile, full 16:9 (column) on sm+ -->
                <div class="relative w-28 shrink-0 self-stretch overflow-hidden bg-elevated sm:aspect-[16/9] sm:w-auto sm:self-auto">
                  <img v-if="p.coverUrl" :src="coverThumbUrl(p)" :alt="p.title" class="size-full object-cover transition duration-500 group-hover:scale-105" >
                  <div v-else class="blog-cover-placeholder grid size-full place-items-center bg-gradient-to-br from-primary/10 to-transparent">
                    <UIcon name="i-tabler-feather" class="blog-cover-icon size-10 text-primary/25" />
                  </div>
                  <span v-if="p.pinned" class="absolute left-2 top-2 flex items-center gap-1 rounded-full bg-default/90 px-2 py-0.5 text-[11px] font-semibold text-primary backdrop-blur sm:left-3 sm:top-3 sm:px-2.5 sm:text-xs"><UIcon name="i-tabler-pin" class="size-3" />置顶</span>
                </div>
                <div class="flex min-w-0 flex-1 flex-col py-3 pr-4 sm:p-5">
                  <h3 class="font-display line-clamp-2 text-base font-semibold leading-snug text-highlighted transition group-hover:text-primary sm:text-lg">{{ p.title }}</h3>
                  <p v-if="p.excerpt" class="mt-2 line-clamp-2 flex-1 text-sm text-muted max-sm:hidden">{{ p.excerpt }}</p>
                  <div class="mt-2 flex items-center gap-3 text-xs text-muted sm:mt-4">
                    <ClientOnly><span>{{ rel(p.publishedAt || p.createdAt) }}</span><template #fallback><span /></template></ClientOnly>
                    <span class="flex items-center gap-1"><UIcon name="i-tabler-clock" class="size-3.5" />{{ readMin(p.content) }} 分钟</span>
                  </div>
                </div>
              </NuxtLink>
            </div>
            <div v-if="totalPages > 1" class="mt-12 flex items-center justify-center gap-3">
              <UButton aria-label="上一页" icon="i-tabler-chevron-left" color="neutral" variant="outline" size="sm" :disabled="page <= 1" @click="() => { page -= 1 }" />
              <span class="text-sm text-muted">{{ page }} / {{ totalPages }}</span>
              <UButton aria-label="下一页" icon="i-tabler-chevron-right" color="neutral" variant="outline" size="sm" :disabled="page >= totalPages" @click="() => { page += 1 }" />
            </div>
          </template>
        </div>
      </div>

      <!-- right rail: discovery (sticky) -->
      <aside class="space-y-8 lg:sticky lg:top-20 lg:self-start">
        <PostMiniList title="热门" icon="i-tabler-flame" :posts="popular.slice(0, 5)" />
        <PostMiniList title="随机阅读" icon="i-tabler-arrows-shuffle" :posts="random" :on-refresh="refreshRandom" :refreshing="randomPending" />
        <div v-if="tags.length">
          <h3 class="mb-3 flex items-center gap-2 font-display font-semibold text-highlighted"><UIcon name="i-tabler-hash" class="size-4 text-primary" />标签</h3>
          <div class="flex flex-wrap gap-1.5">
            <NuxtLink v-for="t in tags" :key="t.id" :to="`/tags/${t.slug}`" class="rounded-full bg-elevated px-2.5 py-1 text-xs text-muted transition hover:text-primary">#{{ t.name }}</NuxtLink>
          </div>
          <NuxtLink to="/tags" class="mt-3 inline-block text-xs text-primary hover:underline">标签云 →</NuxtLink>
        </div>
      </aside>
    </section>
  </div>
</template>

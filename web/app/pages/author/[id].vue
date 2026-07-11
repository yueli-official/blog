<script setup lang="ts">
import { ManagePagination, SkeletonCards } from '@platform/ui/components'
import type { AuthorPage, PostView } from '~/types'

// Author page (M5): a profile hero (cover + avatar + bio + stats) over the
// author's published posts. Layout patterns harvested from the donor user page.
definePageMeta({ width: 'full' })
const route = useRoute()
const id = route.params.id as string
const { call } = useApi()
const page = ref(1)
const size = 12

const { data, error, pending } = await useAsyncData(
  () => `author-${id}-${page.value}`,
  () => call<AuthorPage>(`/api/v1/authors/${id}`, { query: { page: page.value, size } }),
  { watch: [page] }
)
if (error.value || !data.value?.author) {
  throw createError({ statusCode: 404, statusMessage: '作者不存在', fatal: true })
}

const author = computed(() => data.value!.author)
const posts = computed<PostView[]>(() => data.value?.posts ?? [])
const total = computed(() => data.value?.total ?? 0)
const totalViews = computed(() => data.value?.totalViews ?? 0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size)))
const showSkeleton = useMinLoading(pending)
const name = computed(() => author.value.displayName || author.value.id.slice(0, 8))
const initial = computed(() => name.value.charAt(0).toUpperCase())
const avatarSrc = useVerifiedImage(() => author.value.avatarUrl)
const roleLabel = '作者'
const bannerFailed = ref(false)
const bannerMounted = ref(false)
const showBanner = computed(() => bannerMounted.value && Boolean(author.value.bannerUrl) && !bannerFailed.value)
onMounted(() => { bannerMounted.value = true })
watch(() => author.value.bannerUrl, () => { bannerFailed.value = false })
// createdAt has no "now" component, so it's hydration-safe to render in SSR.
const joined = computed(() => {
  if (!author.value.createdAt) return ''
  return new Date(author.value.createdAt).toLocaleDateString('zh-CN', { year: 'numeric', month: 'long' })
})

useSeoMeta({
  title: () => `${name.value} · 博客`,
  description: () => author.value.bio || `${name.value} 的文章`
})
</script>

<template>
  <div v-if="author">
    <UButton to="/" icon="i-tabler-arrow-left" variant="link" color="neutral" label="返回首页" class="-ml-2 mb-6" />

    <!-- profile hero -->
    <section class="mb-10">
      <!-- cover banner -->
      <div class="relative h-44 overflow-hidden rounded-2xl sm:h-56">
        <img v-if="showBanner" :src="author.bannerUrl" alt="" class="size-full object-cover" @error="bannerFailed = true" >
        <div v-else class="blog-cover-placeholder blog-cover-placeholder--plain size-full bg-gradient-to-br from-primary/30 via-primary/10 to-elevated" />
      </div>

      <!-- avatar overlaps the cover; role badge sits opposite -->
      <div class="relative z-10 -mt-12 flex items-end justify-between px-1 sm:px-4">
        <UAvatar :src="avatarSrc" :text="initial" size="3xl" class="ring-4 ring-default shadow-lg" />
        <UBadge :label="roleLabel" color="primary" variant="subtle" size="lg" class="mb-2" />
      </div>

      <!-- identity + bio -->
      <div class="mt-4 px-1 sm:px-4">
        <h1 class="font-display text-2xl font-bold leading-tight text-highlighted sm:text-3xl">{{ name }}</h1>
        <p v-if="author.bio" class="mt-2 max-w-2xl text-sm leading-relaxed text-muted sm:text-base">{{ author.bio }}</p>

        <div class="mt-3 flex flex-wrap items-center gap-x-5 gap-y-1.5 text-sm text-muted">
          <span v-if="joined" class="flex items-center gap-1.5">
            <UIcon name="i-tabler-calendar" class="size-4 shrink-0" />{{ joined }} 加入
          </span>
        </div>

        <!-- social links -->
        <div v-if="author.socialLinks?.length" class="mt-4 flex flex-wrap gap-1.5">
          <UButton
            v-for="(link, i) in author.socialLinks"
            :key="i"
            :to="link.url"
            :icon="socialIcon(link)"
            :label="link.label"
            target="_blank"
            rel="noopener"
            color="neutral"
            variant="soft"
            size="xs"
          />
        </div>
      </div>

      <!-- stats strip -->
      <div class="mt-6 grid grid-cols-3 overflow-hidden rounded-2xl border border-default">
        <div class="flex flex-col items-center gap-0.5 py-4">
          <span class="font-display text-2xl font-bold text-primary">{{ total }}</span>
          <span class="text-xs text-muted">文章</span>
        </div>
        <div class="flex flex-col items-center gap-0.5 border-l border-default py-4">
          <span class="font-display text-2xl font-bold text-primary">{{ totalViews.toLocaleString() }}</span>
          <span class="text-xs text-muted">总阅读</span>
        </div>
        <div class="flex flex-col items-center justify-center gap-1 border-l border-default py-4">
          <UBadge :label="roleLabel" color="primary" variant="subtle" />
          <span class="text-xs text-muted">身份</span>
        </div>
      </div>
    </section>

    <!-- posts -->
    <h2 class="font-display mb-5 flex items-center gap-2 text-xl font-semibold text-highlighted">
      <UIcon name="i-tabler-article" class="size-5 text-primary" />全部文章
      <span class="text-sm font-normal text-muted">{{ total }}</span>
    </h2>

    <SkeletonCards v-if="showSkeleton" :count="6" />

    <div v-else-if="!posts.length" class="rounded-2xl border border-dashed border-default py-20 text-center text-muted">
      <UIcon name="i-tabler-feather" class="mx-auto size-8 text-primary/40" />
      <p class="mt-3 text-sm">这位作者还没有发表文章</p>
    </div>

    <div v-else class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <NuxtLink
        v-for="p in posts"
        :key="p.id"
        :to="`/posts/${p.slug}`"
        class="blog-article-card group flex flex-col overflow-hidden rounded-2xl border border-default transition hover:border-primary/40 hover:shadow-md"
      >
        <div class="relative aspect-[16/9] overflow-hidden bg-elevated">
          <img v-if="p.coverUrl" :src="coverThumbUrl(p)" :alt="p.title" class="size-full object-cover transition duration-500 group-hover:scale-105" >
          <div v-else class="blog-cover-placeholder grid size-full place-items-center bg-gradient-to-br from-primary/10 to-transparent"><UIcon name="i-tabler-feather" class="blog-cover-icon size-10 text-primary/30" /></div>
        </div>
        <div class="flex flex-1 flex-col p-5">
          <h3 class="font-display text-lg font-semibold leading-snug text-highlighted transition group-hover:text-primary">{{ p.title }}</h3>
          <p v-if="p.excerpt" class="mt-2 line-clamp-2 flex-1 text-sm text-muted">{{ p.excerpt }}</p>
          <div class="mt-4 flex items-center gap-3 text-xs text-muted">
            <ClientOnly><span>{{ rel(p.publishedAt || p.createdAt) }}</span><template #fallback><span /></template></ClientOnly>
            <span class="flex items-center gap-1"><UIcon name="i-tabler-eye" class="size-3.5" />{{ p.viewCount }}</span>
          </div>
        </div>
      </NuxtLink>
    </div>

    <ManagePagination v-model="page" :total-pages="totalPages" class="mt-10" />
  </div>
</template>

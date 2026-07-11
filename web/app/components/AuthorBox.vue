<script setup lang="ts">
import type { AuthorView } from '~/types'

// Author profile card (M5). `compact` renders the sidebar-rail variant; the
// default is the full header used on the author page. displayName/avatar fall
// back to the identity id when the author hasn't filled the profile in.
const props = defineProps<{ author: AuthorView, compact?: boolean }>()

const name = computed(() => props.author.displayName || props.author.id.slice(0, 8))
const initial = computed(() => name.value.charAt(0).toUpperCase())
const avatarSrc = useVerifiedImage(() => props.author.avatarUrl)
const roleLabel = '作者'
const bannerFailed = ref(false)
const bannerMounted = ref(false)
const showBanner = computed(() => bannerMounted.value && !props.compact && Boolean(props.author.bannerUrl) && !bannerFailed.value)
onMounted(() => { bannerMounted.value = true })
watch(() => props.author.bannerUrl, () => { bannerFailed.value = false })
// socialIcon is auto-imported from app/utils/social.ts (shared with the author page).
</script>

<template>
  <div class="overflow-hidden rounded-2xl border border-default bg-elevated/30">
    <!-- banner -->
    <div v-if="!compact" class="relative h-24">
      <img v-if="showBanner" :src="author.bannerUrl" alt="" class="size-full object-cover" @error="bannerFailed = true" >
      <div v-else class="blog-cover-placeholder blog-cover-placeholder--plain size-full bg-gradient-to-br from-primary/30 via-primary/10 to-elevated" />
    </div>

    <div class="px-5 pb-5">
      <!-- avatar overlaps the banner -->
      <div class="flex items-end justify-between" :class="compact ? 'pt-5' : '-mt-8'">
        <UAvatar
          :src="avatarSrc"
          :text="initial"
          :size="compact ? 'lg' : 'xl'"
          class="ring-4 ring-default"
        />
        <UBadge :label="roleLabel" color="primary" variant="subtle" size="sm" class="mb-1" />
      </div>

      <div class="mt-3">
        <NuxtLink :to="`/author/${author.id}`" class="font-display text-lg font-semibold text-highlighted transition hover:text-primary">
          {{ name }}
        </NuxtLink>
        <p v-if="typeof author.postCount === 'number'" class="text-xs text-muted">{{ author.postCount }} 篇文章</p>
      </div>

      <p v-if="author.bio" class="mt-2 text-sm leading-relaxed text-muted" :class="compact ? 'line-clamp-3' : ''">
        {{ author.bio }}
      </p>

      <div v-if="author.socialLinks?.length" class="mt-4 flex flex-wrap gap-1.5">
        <UButton
          v-for="(link, i) in author.socialLinks"
          :key="i"
          :to="link.url"
          :icon="socialIcon(link)"
          :label="compact ? undefined : link.label"
          target="_blank"
          rel="noopener"
          color="neutral"
          variant="soft"
          size="xs"
          :square="compact"
        />
      </div>
    </div>
  </div>
</template>

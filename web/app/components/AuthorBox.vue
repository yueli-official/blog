<script setup lang="ts">
import type { AuthorView } from "~/types";

// Sidebar and profile variants have different information density; keeping the
// variants explicit prevents the article rail from inheriting profile-page chrome.
const props = withDefaults(
  defineProps<{
    author: AuthorView;
    variant?: "sidebar" | "profile";
  }>(),
  { variant: "profile" },
);

const name = computed(
  () => props.author.displayName || props.author.id.slice(0, 8),
);
const initial = computed(() => name.value.charAt(0).toUpperCase());
const avatarSrc = useVerifiedImage(() => props.author.avatarUrl);
const bannerSrc = useVerifiedImage(() => props.author.bannerUrl);
const sidebar = computed(() => props.variant === "sidebar");
const roleLabel = computed(() =>
  props.author.role === "contributor" ? "撰稿人" : "",
);
const hasBanner = computed(() => Boolean(bannerSrc.value));
const profileURL = computed(() => `/author/${props.author.id}`);
// socialIcon is auto-imported from app/utils/social.ts (shared with the author page).
</script>

<template>
  <section
    v-if="sidebar"
    class="overflow-hidden rounded-xl border border-default bg-default"
    aria-label="文章作者"
    data-article-author
    data-author-layout="cover"
  >
    <NuxtLink
      :to="profileURL"
      data-author-banner
      :aria-label="`查看 ${name} 的主页`"
      class="group relative block aspect-[3/1] overflow-hidden bg-elevated"
    >
      <img
        v-if="hasBanner"
        :src="bannerSrc"
        alt=""
        class="size-full object-cover transition duration-500 group-hover:scale-[1.02]"
      />
      <div
        v-else
        class="blog-cover-placeholder blog-cover-placeholder--plain size-full bg-gradient-to-br from-primary/25 via-primary/10 to-elevated"
      />
      <div
        class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent"
        data-author-cover-shade
      />
      <div class="absolute inset-x-0 bottom-0 flex min-w-0 items-end gap-3 p-4">
        <UAvatar
          :src="avatarSrc"
          :text="initial"
          alt=""
          size="lg"
          class="size-12 shrink-0 shadow-sm ring-[3px] ring-white/90"
        />
        <div class="min-w-0 flex-1">
          <h2 class="truncate font-display text-lg font-semibold text-white">
            {{ name }}
          </h2>
          <p v-if="author.handle" class="truncate text-xs text-white/75">
            @{{ author.handle }}
          </p>
        </div>
        <UBadge
          v-if="roleLabel"
          :label="roleLabel"
          color="neutral"
          variant="soft"
          size="sm"
          class="shrink-0 bg-default/90 text-highlighted"
        />
      </div>
    </NuxtLink>

    <div class="p-4">
      <p v-if="author.bio" class="line-clamp-2 text-sm leading-6 text-muted">
        {{ author.bio }}
      </p>

      <div
        v-if="
          typeof author.postCount === 'number' || author.socialLinks?.length
        "
        class="flex min-h-8 items-center justify-between gap-3"
        :class="author.bio ? 'mt-3' : ''"
      >
        <span
          v-if="typeof author.postCount === 'number'"
          class="text-xs text-muted"
        >
          {{ author.postCount }} 篇文章
        </span>
        <div
          v-if="author.socialLinks?.length"
          class="ml-auto flex flex-wrap justify-end gap-1"
        >
          <UTooltip
            v-for="(link, i) in author.socialLinks"
            :key="`${link.url}-${i}`"
            :text="link.label"
          >
            <UButton
              :to="link.url"
              :icon="socialIcon(link)"
              :aria-label="`打开 ${link.label}`"
              target="_blank"
              rel="noopener"
              color="neutral"
              variant="ghost"
              size="xs"
              square
              class="inline-flex size-8 items-center justify-center"
            />
          </UTooltip>
        </div>
      </div>
    </div>
  </section>

  <section
    v-else
    class="overflow-hidden rounded-2xl border border-default bg-default"
  >
    <div class="relative aspect-[4/1] overflow-hidden bg-elevated">
      <img
        v-if="hasBanner"
        :src="bannerSrc"
        alt=""
        class="size-full object-cover"
      />
      <div
        v-else
        class="blog-cover-placeholder blog-cover-placeholder--plain size-full bg-gradient-to-br from-primary/30 via-primary/10 to-elevated"
      />
      <div
        class="absolute inset-0 bg-gradient-to-t from-black/70 via-black/10 to-transparent"
      />
    </div>

    <div class="px-5 pb-5">
      <div class="-mt-8 flex items-end justify-between">
        <UAvatar
          :src="avatarSrc"
          :text="initial"
          alt=""
          size="xl"
          class="ring-4 ring-default"
        />
        <UBadge
          v-if="roleLabel"
          :label="roleLabel"
          color="primary"
          variant="subtle"
          size="sm"
          class="mb-1"
        />
      </div>

      <div class="mt-3">
        <NuxtLink
          :to="profileURL"
          class="font-display text-lg font-semibold text-highlighted transition hover:text-primary"
        >
          {{ name }}
        </NuxtLink>
        <p
          v-if="typeof author.postCount === 'number'"
          class="text-xs text-muted"
        >
          {{ author.postCount }} 篇文章
        </p>
      </div>

      <p v-if="author.bio" class="mt-2 text-sm leading-relaxed text-muted">
        {{ author.bio }}
      </p>

      <div v-if="author.socialLinks?.length" class="mt-4 flex flex-wrap gap-1">
        <UTooltip
          v-for="(link, i) in author.socialLinks"
          :key="`${link.url}-${i}`"
          :text="link.label"
        >
          <UButton
            :to="link.url"
            :icon="socialIcon(link)"
            :aria-label="`打开 ${link.label}`"
            target="_blank"
            rel="noopener"
            color="neutral"
            variant="soft"
            size="xs"
            square
            class="inline-flex size-8 items-center justify-center"
          />
        </UTooltip>
      </div>
    </div>
  </section>
</template>

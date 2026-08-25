<script setup lang="ts">
import type { CommentView, CommentList } from "~/types";

// Reader-facing comment area: a two-level thread list + a compose box. Fetched
// client-side (member state matters and comments aren't SEO-critical).
const props = defineProps<{ slug: string; commentStatus: number }>();

const { call } = useApi();

const items = ref<CommentView[]>([]);
const total = ref(0);
const loading = ref(true);
const replyTo = ref<string | null>(null);

async function load() {
  loading.value = true;
  try {
    const r = await call<CommentList>(`/api/v1/posts/${props.slug}/comments`, {
      query: { page: 1, size: 100 },
    });
    items.value = r.items;
    total.value = r.total;
  } catch {
    // leave the list empty on error
  } finally {
    loading.value = false;
  }
}

function onSubmitted(pending: boolean) {
  replyTo.value = null;
  if (!pending) load(); // approved → reflect immediately
}

onMounted(load);

function authorInitial(name: string) {
  return (name || "?").charAt(0).toUpperCase();
}
</script>

<template>
  <section class="mt-16 border-t border-default pt-10">
    <h2
      class="font-display mb-6 flex items-center gap-2 text-xl font-semibold text-highlighted"
    >
      <UIcon name="i-tabler-messages" class="size-5 text-primary" />
      评论
      <span v-if="total" class="text-base font-normal text-muted">{{
        total
      }}</span>
    </h2>

    <div
      v-if="commentStatus === 0"
      class="rounded-xl border border-dashed border-default py-8 text-center text-sm text-muted"
    >
      <UIcon name="i-tabler-message-off" class="mx-auto mb-1 size-6" />
      本文已关闭评论
    </div>

    <template v-else>
      <CommentForm :slug="slug" class="mb-10" @submitted="onSubmitted" />

      <div v-if="loading" class="py-10 text-center text-muted">
        <UIcon name="i-tabler-loader-2" class="size-5 animate-spin" />
      </div>

      <div
        v-else-if="!items.length"
        class="rounded-xl border border-dashed border-default py-10 text-center"
      >
        <UIcon name="i-tabler-message-2" class="mx-auto size-7 text-muted" />
        <p class="mt-2 text-sm text-muted">还没有评论,来说第一句吧</p>
      </div>

      <ul v-else class="space-y-8">
        <li v-for="c in items" :key="c.id">
          <div class="flex gap-3">
            <UAvatar
              :src="c.avatarUrl"
              :text="authorInitial(c.authorName)"
              alt=""
              size="sm"
              class="shrink-0"
            />
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2 text-sm">
                <span class="font-medium text-highlighted">{{
                  c.authorName
                }}</span>
                <UBadge
                  v-if="c.isAnonymous"
                  label="匿名用户"
                  color="neutral"
                  variant="subtle"
                  size="sm"
                />
                <span class="text-dimmed">·</span>
                <ClientOnly
                  ><span class="text-muted">{{ rel(c.createdAt) }}</span
                  ><template #fallback><span /></template
                ></ClientOnly>
              </div>
              <p
                class="mt-1 whitespace-pre-wrap text-sm leading-relaxed text-default"
              >
                {{ c.content }}
              </p>
              <UButton
                :icon="
                  replyTo === c.id ? 'i-tabler-x' : 'i-tabler-corner-down-right'
                "
                :label="replyTo === c.id ? '取消' : '回复'"
                size="xs"
                variant="ghost"
                color="neutral"
                class="-ml-2 mt-1"
                @click="
                  () => {
                    replyTo = replyTo === c.id ? null : c.id;
                  }
                "
              />

              <CommentForm
                v-if="replyTo === c.id"
                :slug="slug"
                :parent-id="c.id"
                compact
                class="mt-3"
                @submitted="onSubmitted"
              />

              <ul
                v-if="c.replies?.length"
                class="mt-4 space-y-4 border-l-2 border-default pl-4"
              >
                <li v-for="r in c.replies" :key="r.id" class="flex gap-2.5">
                  <UAvatar
                    :src="r.avatarUrl"
                    :text="authorInitial(r.authorName)"
                    alt=""
                    size="2xs"
                    class="mt-0.5 shrink-0"
                  />
                  <div class="min-w-0 flex-1">
                    <div class="flex items-center gap-2 text-sm">
                      <span class="font-medium text-highlighted">{{
                        r.authorName
                      }}</span>
                      <UBadge
                        v-if="r.isAnonymous"
                        label="匿名用户"
                        color="neutral"
                        variant="subtle"
                        size="sm"
                      />
                      <span class="text-dimmed">·</span>
                      <ClientOnly
                        ><span class="text-muted">{{ rel(r.createdAt) }}</span
                        ><template #fallback><span /></template
                      ></ClientOnly>
                    </div>
                    <p
                      class="mt-1 whitespace-pre-wrap text-sm leading-relaxed text-default"
                    >
                      {{ r.content }}
                    </p>
                  </div>
                </li>
              </ul>
            </div>
          </div>
        </li>
      </ul>
    </template>
  </section>
</template>

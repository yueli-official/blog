<script setup lang="ts">
import ShareBar from "~/components/ShareBar.vue";

const props = withDefaults(
  defineProps<{
    layout?: "rail" | "inline";
    liked: boolean;
    bookmarked: boolean;
    pendingAction?: "like" | "bookmark" | null;
    title: string;
  }>(),
  {
    layout: "inline",
    pendingAction: null,
  },
);

const emit = defineEmits<{
  like: [];
  bookmark: [];
}>();

const tooltipSide = computed(() =>
  props.layout === "rail" ? "left" : "top",
);
const shareSide = computed(() =>
  props.layout === "rail" ? "right" : "top",
);
</script>

<template>
  <nav
    aria-label="文章操作"
    class="items-center gap-2"
    :class="
      layout === 'rail'
        ? 'sticky top-28 flex flex-col'
        : 'flex flex-row'
    "
  >
    <UTooltip
      :text="liked ? '取消点赞' : '点赞'"
      :content="{ side: tooltipSide, sideOffset: 8 }"
    >
      <UButton
        :icon="liked ? 'i-tabler-heart-filled' : 'i-tabler-heart'"
        :color="liked ? 'primary' : 'neutral'"
        variant="soft"
        size="md"
        square
        class="size-11 justify-center rounded-full"
        :aria-label="liked ? '取消点赞' : '点赞'"
        :aria-pressed="liked"
        :loading="pendingAction === 'like'"
        :disabled="Boolean(pendingAction) && pendingAction !== 'like'"
        @click="emit('like')"
      />
    </UTooltip>

    <UPopover
      :content="{ side: shareSide, align: 'center', sideOffset: 10 }"
      :arrow="true"
    >
      <UButton
        icon="i-tabler-share-3"
        color="neutral"
        variant="soft"
        size="md"
        square
        class="size-11 justify-center rounded-full"
        aria-label="分享文章"
        title="分享文章"
      />
      <template #content>
        <div class="p-2">
          <ShareBar :title="title" />
        </div>
      </template>
    </UPopover>

    <UTooltip
      :text="bookmarked ? '取消收藏' : '收藏'"
      :content="{ side: tooltipSide, sideOffset: 8 }"
    >
      <UButton
        :icon="
          bookmarked ? 'i-tabler-bookmark-filled' : 'i-tabler-bookmark'
        "
        :color="bookmarked ? 'primary' : 'neutral'"
        variant="soft"
        size="md"
        square
        class="size-11 justify-center rounded-full"
        :aria-label="bookmarked ? '取消收藏' : '收藏'"
        :aria-pressed="bookmarked"
        :loading="pendingAction === 'bookmark'"
        :disabled="Boolean(pendingAction) && pendingAction !== 'bookmark'"
        @click="emit('bookmark')"
      />
    </UTooltip>
  </nav>
</template>

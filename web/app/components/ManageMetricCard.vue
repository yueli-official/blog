<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    label: string;
    value: string | number;
    detail: string;
    icon: string;
    tone?: "blue" | "blue-soft" | "blue-deep" | "neutral";
    to?: string;
  }>(),
  { tone: "blue", to: "" },
);

const root = computed(() =>
  props.to ? resolveComponent("NuxtLink") : "article",
);
const toneClass = computed(
  () =>
    ({
      blue: "text-primary",
      "blue-soft": "text-info",
      "blue-deep": "text-primary",
      neutral: "text-muted",
    })[props.tone],
);
</script>

<template>
  <component
    :is="root"
    :to="to || undefined"
    class="relative grid min-w-0 grid-cols-[2.625rem_minmax(0,1fr)] gap-3 overflow-hidden rounded-xl bg-default p-4 shadow-sm transition-colors duration-150 hover:bg-elevated"
    :class="toneClass"
  >
    <div class="grid size-10 place-items-center rounded-xl bg-current/10">
      <UIcon :name="icon" class="size-5" />
    </div>
    <div class="min-w-0">
      <div class="text-xs text-muted">{{ label }}</div>
      <div
        class="mt-0.5 text-2xl font-bold leading-tight tracking-[-0.03em] text-highlighted"
      >
        {{ value }}
      </div>
      <div class="mt-0.5 truncate text-xs text-dimmed">{{ detail }}</div>
    </div>
  </component>
</template>

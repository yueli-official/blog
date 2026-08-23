<script setup lang="ts">
import { TabbedSurface } from "@yueli/ui/admin";

interface ManageTabItem {
  label: string;
  value: string;
  icon?: string;
  badge?: string | number;
}

const active = defineModel<string>({ required: true });
defineProps<{
  items: readonly ManageTabItem[];
  navigationLabel: string;
}>();
</script>

<template>
  <TabbedSurface
    v-model="active"
    :items="items"
    :navigation-label="navigationLabel"
  >
    <slot />
  </TabbedSurface>
</template>

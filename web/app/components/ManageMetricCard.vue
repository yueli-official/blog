<script setup lang="ts">
import { AdminMetricCard } from "@yueli/ui/admin";
const props = withDefaults(defineProps<{
  label: string;
  value: string | number;
  detail: string;
  icon: string;
  tone?: "blue" | "blue-soft" | "blue-deep" | "neutral";
  to?: string;
}>(), { tone: "blue" });
const semanticTone = computed(() => props.tone === "neutral" ? "neutral" : props.tone === "blue-soft" ? "info" : "primary");
</script>

<template>
  <AdminMetricCard v-bind="props" :tone="semanticTone" />
</template>

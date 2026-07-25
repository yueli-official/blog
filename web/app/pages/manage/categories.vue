<script setup lang="ts">
// Category governance: capability-gated curated category tree.
definePageMeta({ layout: 'manage', middleware: 'auth' })
useSeoMeta({ title: '分类 · 控制台' })
</script>

<template>
  <TaxonomyManager kind="category" />
</template>

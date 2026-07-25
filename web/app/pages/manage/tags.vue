<script setup lang="ts">
// Tag governance: capability-gated flat tag folksonomy.
definePageMeta({ layout: 'manage', middleware: 'auth' })
useSeoMeta({ title: '标签 · 控制台' })
</script>

<template>
  <TaxonomyManager kind="tag" />
</template>

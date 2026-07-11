<script setup lang="ts">
// Legacy taxonomy archive route. M2 split categories (/category/[slug]) from
// tags (/tags/[slug]); this path now redirects to the category view (post
// filtering is slug-based and identical) so old links keep working.
const route = useRoute()
await navigateTo(`/category/${route.params.slug}`, { redirectCode: 301 })
</script>

<template>
  <div />
</template>

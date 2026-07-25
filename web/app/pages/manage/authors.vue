<script setup lang="ts">
definePageMeta({ layout: "manage", middleware: "auth" });
await navigateTo("/manage/authorization", { redirectCode: 301 });
</script>

<template>
  <div />
</template>

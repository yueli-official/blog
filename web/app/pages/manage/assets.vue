<script setup lang="ts">
import { AssetRegistrationSummary } from "@yueli/asset-nuxt/components";

definePageMeta({ layout: "manage", middleware: "auth" });
useSeoMeta({ title: "资源策略 · 控制台" });
</script>

<template>
  <div class="space-y-5">
    <ManagePageHeader title="资源策略" />
    <ClientOnly>
      <AssetRegistrationSummary
        expected-namespace="blog"
        :profile-order="['blog-cover', 'blog-post']"
      />
      <template #fallback>
        <SkeletonList :rows="4" />
      </template>
    </ClientOnly>
  </div>
</template>

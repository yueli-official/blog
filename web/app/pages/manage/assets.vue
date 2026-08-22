<script setup lang="ts">
import { AssetRegistrationSummary } from "@yueli/asset-nuxt/components";
import type { AssetVariantPreset } from "@yueli/asset-nuxt/registration";

definePageMeta({ layout: "manage", middleware: "auth" });
useSeoMeta({ title: "资源策略 · 控制台" });
const { can } = useMe();
const canEditAssets = computed(() => can("blog.asset_settings.manage"));
const variantPresets: AssetVariantPreset[] = [
  { profileKey: "blog-cover", key: "card", label: "文章卡片 · 600×400", width: 600, height: 400, mode: "fill", format: "webp", quality: 85 },
  { profileKey: "blog-cover", key: "home", label: "首页大图 · 1200×800", width: 1200, height: 800, mode: "fill", format: "webp", quality: 88 },
  { profileKey: "blog-cover", key: "grid", label: "文章卡片 · 600×400", width: 600, height: 400, mode: "fill", format: "webp", quality: 85 },
  { profileKey: "blog-cover", key: "og", label: "社交分享 · 1200×630", width: 1200, height: 630, mode: "fill", format: "jpeg", quality: 88 },
  { profileKey: "blog-cover", key: "thumbnail", label: "缩略图 · 300×200", width: 300, height: 200, mode: "fill", format: "webp", quality: 82 },
  { profileKey: "blog-post", key: "thumbnail", label: "正文缩略图 · 960px", width: 960, height: 960, mode: "resize", format: "webp", quality: 84 },
  { profileKey: "blog-post", key: "content", label: "正文大图 · 1920px", width: 1920, height: 1920, mode: "resize", format: "webp", quality: 88 },
];
</script>

<template>
  <div class="space-y-5">
    <ManagePageHeader title="资源策略" />
    <ClientOnly>
      <AssetRegistrationSummary
        expected-namespace="blog"
        :profile-order="['blog-cover', 'blog-post']"
        :can-edit="canEditAssets"
        :variant-presets="variantPresets"
      />
      <template #fallback>
        <SkeletonList :rows="4" />
      </template>
    </ClientOnly>
  </div>
</template>

<script setup lang="ts">
import { AssetRegistrationSummary } from "@yueli/asset-nuxt/components";
import type { AssetVariantSlotDefinition } from "@yueli/asset-nuxt/registration";

definePageMeta({ layout: "manage", middleware: "auth" });
useSeoMeta({ title: "资源策略 · 控制台" });
const { can } = useMe();
const canEditAssets = computed(() => can("blog.asset_settings.manage"));
const variantSlots: AssetVariantSlotDefinition[] = [
  {
    profileKey: "blog-cover", key: "card", label: "文章卡片", usage: "文章列表、搜索结果与归档",
    presetLabels: { standard: "标准 · 600×400", compact: "省流 · 480×320" },
  },
  {
    profileKey: "blog-cover", key: "home", label: "首页大图", usage: "首页精选与文章页封面",
    presetLabels: { standard: "标准 · 1200×800", compact: "省流 · 900×600" },
  },
  {
    profileKey: "blog-cover", key: "thumbnail", label: "封面缩略图", usage: "相关文章与紧凑列表",
    presetLabels: { standard: "标准 · 300×200", sharp: "清晰 · 450×300" },
  },
  {
    profileKey: "blog-cover", key: "og", label: "社交分享", usage: "Open Graph 分享预览",
    presetLabels: { fixed: "固定 · 1200×630" },
  },
  {
    profileKey: "blog-post", key: "inline", label: "正文显示", usage: "文章正文内嵌图片",
    presetLabels: { standard: "标准 · 最长边 800px", sharp: "清晰 · 最长边 960px" },
  },
  {
    profileKey: "blog-post", key: "content", label: "正文大图", usage: "点击正文图片后查看",
    presetLabels: { fixed: "固定 · 最长边 1200px" },
  },
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
        :variant-slots="variantSlots"
      />
      <template #fallback>
        <SkeletonList :rows="4" />
      </template>
    </ClientOnly>
  </div>
</template>

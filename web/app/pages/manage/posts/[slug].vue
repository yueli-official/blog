<script setup lang="ts">
import { createBlogNotifier } from "~/utils/feedback";
import { useActionFeedback } from "@yueli/ui/feedback";
import { ActionFeedbackButton } from "@yueli/ui/feedback/pattern";
import { AssetImageProcessor } from "@yueli/asset-nuxt/components";
import type {
  PostDetail,
  ListTaxonomies,
  TaxonomyView,
  ListSeries,
  HomeConfigResponse,
} from "~/types";
import type { PngCompressionRequest } from "~/composables/useUpload";

// Post editor (author): title + editable slug + rich content, with sidebar panels
// for cover / taxonomies / series / editorial flags, and a collapsible SEO block.
// One primary 保存 persists title+slug+excerpt+content; the sidebar panels apply
// on their own (distinct endpoints). Auth-gated; loaded by slug with the author's
// Bearer so own drafts are visible.
definePageMeta({ layout: "manage", middleware: "auth" });

const route = useRoute();
const slug = route.params.slug as string;
const { call } = useApi();
const { can } = useMe();
const canManageTaxonomy = computed(() => can("blog.taxonomy.manage"));
const canManageFlags = computed(() => can("blog.post_flags.manage"));
const compressionRequest = shallowRef<
  (PngCompressionRequest & { resolve: (confirmed: boolean) => void }) | null
>(null);
function confirmPngToJpeg(request: PngCompressionRequest) {
  compressionRequest.value?.resolve(false);
  return new Promise<boolean>((resolve) => {
    compressionRequest.value = { ...request, resolve };
  });
}
function finishPngCompression(confirmed: boolean) {
  const pending = compressionRequest.value;
  compressionRequest.value = null;
  pending?.resolve(confirmed);
}
function fileSizeLabel(bytes: number) {
  return `${(bytes / 1024 / 1024).toFixed(bytes >= 10 * 1024 * 1024 ? 0 : 1)} MB`;
}
onBeforeUnmount(() => finishPngCompression(false));
const { uploadCover, uploadImage } = useUpload({ confirmPngToJpeg });
const toast = createBlogNotifier(useToast());

type ImageProcessingRequest = {
  file: File;
  purpose: "cover" | "content";
  resolve: (file: File) => void;
  reject: (error: Error) => void;
};
const imageProcessingRequest = shallowRef<ImageProcessingRequest | null>(null);
function processImage(file: File, purpose: "cover" | "content") {
  imageProcessingRequest.value?.reject(new Error("已取消上一项图片处理"));
  return new Promise<File>((resolve, reject) => {
    imageProcessingRequest.value = { file, purpose, resolve, reject };
  });
}
function finishImageProcessing(file: File) {
  const pending = imageProcessingRequest.value;
  imageProcessingRequest.value = null;
  pending?.resolve(file);
}
function cancelImageProcessing() {
  const pending = imageProcessingRequest.value;
  imageProcessingRequest.value = null;
  pending?.reject(new Error("已取消图片处理"));
}
function onImageProcessorOpen(value: boolean) {
  if (!value && imageProcessingRequest.value) cancelImageProcessing();
}
onBeforeUnmount(cancelImageProcessing);

const editorComp = ref<{ markSaved: () => void } | null>(null);
async function uploadInlineImage(file: File): Promise<string> {
  const processed = await processImage(file, "content");
  const { url } = await uploadImage(processed);
  return url;
}

const { data, pending, refresh } = await useAsyncData(
  `edit-${slug}`,
  () => call<PostDetail>(`/api/v1/posts/${slug}`),
  { server: false },
);
const post = computed(() => data.value?.post);
const postId = computed(() => post.value?.id || "");

const { data: siteConfig } = await useAsyncData(
  "blog-editor-site-config",
  () => call<HomeConfigResponse>("/api/v1/home"),
  { server: false },
);
const coverAspectWidth = computed(() =>
  Math.max(1, Number(siteConfig.value?.config.coverAspectWidth) || 3),
);
const coverAspectHeight = computed(() =>
  Math.max(1, Number(siteConfig.value?.config.coverAspectHeight) || 2),
);
const coverAspectRatio = computed(
  () => coverAspectWidth.value / coverAspectHeight.value,
);
const coverAspectLabel = computed(
  () => `${coverAspectWidth.value}:${coverAspectHeight.value}`,
);

const mounted = ref(false);
onMounted(() => {
  mounted.value = true;
});

// ── editable title / slug / excerpt / content ─────────────────────────────────
const form = reactive({ title: "", slug: "", excerpt: "", content: "" });
const publishedAtLocal = ref("");

function localDateTime(value?: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const pad = (part: number) => String(part).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function publishedAtRFC3339() {
  if (!publishedAtLocal.value) return undefined;
  const date = new Date(publishedAtLocal.value);
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}
// SeoMeta after `form` is declared: its title getter reads form.title, and the
// reactive effect evaluates synchronously during setup — placing it above the
// declaration hits form's temporal dead zone (Cannot access 'form' before init).
useSeoMeta({ title: () => `${form.title || "未命名文章"} · 控制台` });
const slugTouched = ref(false);
watch(
  data,
  (d) => {
    if (!d?.post) return;
    form.title = d.post.title;
    form.slug = d.post.slug;
    form.excerpt = d.post.excerpt;
    form.content = d.post.content;
    publishedAtLocal.value = localDateTime(d.post.publishedAt);
  },
  { immediate: true },
);

// One 保存 persists everything. Each domain is its own endpoint (post core /
// taxonomies / series / flags / seo); all are idempotent, so re-saving is safe.
const {
  status: saveStatus,
  pending: markSaving,
  success: markSaved,
  reset: resetSave,
} = useActionFeedback();
let activeSave: Promise<string | false> | null = null;
async function performSave(syncRoute: boolean): Promise<string | false> {
  if (!postId.value) return false;
  markSaving();
  try {
    const publishedAt = publishedAtRFC3339();
    await call(`/api/v1/posts/${postId.value}/taxonomies`, {
      method: "PUT",
      body: { taxonomyIds: selected.value },
    });
    await call(`/api/v1/posts/${postId.value}/series`, {
      method: "PUT",
      body: {
        seriesId: seriesId.value === NO_SERIES ? "" : seriesId.value,
        seriesOrder: Number(seriesOrder.value) || 0,
      },
    });
    if (canManageFlags.value) {
      await call(`/api/v1/posts/${postId.value}/flags`, {
        method: "PUT",
        body: { pinned: pinned.value, featured: featured.value },
      });
    }
    await call(`/api/v1/posts/${postId.value}/seo`, {
      method: "PUT",
      body: { ...seo },
    });
    const core = await call<{ post: PostDetail["post"] }>(
      `/api/v1/posts/${postId.value}`,
      {
        method: "PATCH",
        body: {
          title: form.title,
          slug: form.slug,
          excerpt: form.excerpt,
          content: form.content,
          ...(post.value?.status === "published" && publishedAt
            ? { publishedAt }
            : {}),
        },
      },
    );
    markSaved();
    editorComp.value?.markSaved();
    markEditorDraftSaved();
    const savedSlug = core.post.slug;
    if (syncRoute && savedSlug && savedSlug !== route.params.slug) {
      await navigateTo(`/manage/posts/${encodeURIComponent(savedSlug)}`, {
        replace: true,
      });
    } else if (syncRoute) {
      await refresh();
      markEditorDraftSaved();
    }
    return savedSlug;
  } catch (e: any) {
    resetSave();
    const failureCode =
      e?.data?.code || e?.data?.failure?.code || e?.statusMessage || e?.message;
    const description =
      failureCode === "blog.slug_taken"
        ? "文章地址已被占用，请换一个地址后重试。"
        : e?.data?.message || "请检查输入或网络后重试。";
    toast.add({ title: "保存失败", description, color: "error" });
    return false;
  }
}
async function saveCurrent(syncRoute: boolean) {
  if (activeSave) return activeSave;
  activeSave = performSave(syncRoute);
  try {
    return await activeSave;
  } finally {
    activeSave = null;
  }
}
async function save() {
  return Boolean(await saveCurrent(true));
}

// ── Settings inspector ────────────────────────────────────────────────────────
const seo = reactive({
  metaTitle: "",
  metaDesc: "",
  ogTitle: "",
  ogImage: "",
  canonicalUrl: "",
  robots: "",
});
watch(
  data,
  (d) => {
    if (d?.seo) Object.assign(seo, d.seo);
  },
  { immediate: true },
);

function contentSummary(value: string, limit = 160) {
  const summary = value
    .replace(/<[^>]+>/gu, " ")
    .replace(/```[\s\S]*?```/gu, " ")
    .replace(/[#*`[\]()>_~\-|]/gu, " ")
    .replace(/\s+/gu, " ")
    .trim();
  return summary.length > limit
    ? `${summary.slice(0, limit).trim()}…`
    : summary;
}

function fillSearchMetadata() {
  const title = form.title.trim();
  const description = form.excerpt.trim() || contentSummary(form.content);
  seo.metaTitle = title;
  seo.ogTitle = title;
  seo.metaDesc = description;
}

// ── cover image ──────────────────────────────────────────────────────────────
const coverInput = ref<HTMLInputElement>();
const coverPct = ref(-1);
async function onPickCover(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;
  let processed: File;
  try {
    processed = await processImage(file, "cover");
  } catch {
    return;
  }
  coverPct.value = 0;
  try {
    await uploadCover(postId.value, processed, (pct) => {
      coverPct.value = pct;
    });
    await refresh();
  } catch (err: any) {
    toast.add({
      title: "封面上传失败",
      description: err?.message || "请重试",
      color: "error",
    });
  } finally {
    coverPct.value = -1;
  }
}

// ── taxonomies (M2): category tree (curated) + tags (folksonomy) ──────────────
const { data: taxes, refresh: refreshTaxes } = await useAsyncData(
  "all-taxes",
  () => call<ListTaxonomies>("/api/v1/taxonomies"),
  { server: false, default: () => ({ items: [] as TaxonomyView[] }) },
);
const selected = ref<string[]>([]);
watch(
  data,
  (d) => {
    if (d?.taxonomies) selected.value = d.taxonomies.map((t) => t.id);
  },
  { immediate: true },
);
function toggleTax(id: string) {
  selected.value = selected.value.includes(id)
    ? selected.value.filter((x) => x !== id)
    : [...selected.value, id];
}
const categories = computed(() =>
  (taxes.value?.items || []).filter((t) => t.taxonomy === "category"),
);
const tags = computed(() =>
  (taxes.value?.items || []).filter((t) => t.taxonomy === "tag"),
);
type TaxonomySelectOption = {
  label: string;
  value: string;
  description: string;
  searchText: string;
};
const categoryOptions = computed<TaxonomySelectOption[]>(() => {
  const byId = new Map(categories.value.map((category) => [category.id, category]));
  const pathCache = new Map<string, string[]>();
  const pathFor = (category: TaxonomyView, seen = new Set<string>()): string[] => {
    const cached = pathCache.get(category.id);
    if (cached) return cached;
    if (seen.has(category.id)) return [category.name];
    seen.add(category.id);
    const parent = category.parentId ? byId.get(category.parentId) : undefined;
    const path = parent
      ? [...pathFor(parent, seen), category.name]
      : [category.name];
    pathCache.set(category.id, path);
    return path;
  };
  return categories.value
    .map((category) => {
      const path = pathFor(category);
      const parentPath = path.slice(0, -1).join(" / ");
      return {
        label: category.name,
        value: category.id,
        description: parentPath || "顶级分类",
        searchText: `${path.join(" ")} ${category.slug}`,
      };
    })
    .sort((left, right) =>
      `${left.description}/${left.label}`.localeCompare(
        `${right.description}/${right.label}`,
        "zh-CN",
      ),
    );
});
const tagOptions = computed<TaxonomySelectOption[]>(() =>
  tags.value
    .map((tag) => ({
      label: `#${tag.name}`,
      value: tag.id,
      description: tag.slug,
      searchText: `${tag.name} ${tag.slug}`,
    }))
    .sort((left, right) => left.label.localeCompare(right.label, "zh-CN")),
);
const selectedCategories = computed(() =>
  categories.value.filter((category) => selected.value.includes(category.id)),
);
const selectedTags = computed(() =>
  tags.value.filter((t) => selected.value.includes(t.id)),
);
const selectedCategoryIds = computed<string[]>({
  get: () => selectedCategories.value.map((category) => category.id),
  set: (ids) => {
    const categoryIds = new Set(categories.value.map((category) => category.id));
    selected.value = [
      ...selected.value.filter((id) => !categoryIds.has(id)),
      ...ids.filter((id) => categoryIds.has(id)),
    ];
  },
});
const selectedTagIds = computed<string[]>({
  get: () => selectedTags.value.map((tag) => tag.id),
  set: (ids) => {
    const tagIds = new Set(tags.value.map((tag) => tag.id));
    selected.value = [
      ...selected.value.filter((id) => !tagIds.has(id)),
      ...ids.filter((id) => tagIds.has(id)),
    ];
  },
});
const selectedCategoryCount = computed(
  () => selectedCategories.value.length,
);

// create via the shared modal (supports a custom slug); auto-select on create.
const showCatModal = ref(false);
const showTagModal = ref(false);
const categoryModalName = ref("");
const tagModalName = ref("");
function onTaxCreated(t: TaxonomyView) {
  refreshTaxes();
  if (!selected.value.includes(t.id))
    selected.value = [...selected.value, t.id];
}
function openCategoryCreate(name = "") {
  categoryModalName.value = name.trim();
  showCatModal.value = true;
}
function openTagCreate(name = "") {
  tagModalName.value = name.replace(/^#/u, "").trim();
  showTagModal.value = true;
}

// ── series (M3) ───────────────────────────────────────────────────────────────
const NO_SERIES = "__none__";
const { data: seriesData, refresh: refreshSeries } = await useAsyncData(
  "all-series",
  () => call<ListSeries>("/api/v1/series"),
  { server: false, default: () => ({ items: [] }) },
);
const seriesId = ref(NO_SERIES);
const seriesOrder = ref(0);
watch(
  data,
  (d) => {
    seriesId.value = d?.series?.id || NO_SERIES;
    seriesOrder.value = d?.post?.seriesOrder ?? 0;
  },
  { immediate: true },
);
const seriesItems = computed(() => [
  { label: "（无系列）", value: NO_SERIES },
  ...(seriesData.value?.items || []).map((s) => ({
    label: s.name,
    value: s.id,
  })),
]);
const newSeriesName = ref("");
const creatingSeries = ref(false);
async function createSeries() {
  const name = newSeriesName.value.trim();
  if (!name) return;
  creatingSeries.value = true;
  try {
    const res = await call<{ series: { id: string } }>("/api/v1/series", {
      method: "POST",
      body: { name },
    });
    newSeriesName.value = "";
    await refreshSeries();
    seriesId.value = res.series.id;
  } catch (e: any) {
    toast.add({
      title: "创建系列失败",
      description: e?.data?.message || "请重试",
      color: "error",
    });
  } finally {
    creatingSeries.value = false;
  }
}

// ── protected editorial flags: pinned / featured ─────────────────────────────
const pinned = ref(false);
const featured = ref(false);
watch(
  data,
  (d) => {
    pinned.value = d?.post?.pinned ?? false;
    featured.value = d?.post?.featured ?? false;
  },
  { immediate: true },
);

// ── lifecycle: publish / draft / archive / trash ─────────────────────────────
const busy = ref("");
async function setStatus(status: string) {
  if (busy.value) return;
  busy.value = status;
  try {
    const savedSlug = await saveCurrent(false);
    if (!savedSlug) return;
    await call(`/api/v1/posts/${postId.value}`, {
      method: "PATCH",
      body: { status },
    });
    if (savedSlug !== route.params.slug) {
      await navigateTo(`/manage/posts/${encodeURIComponent(savedSlug)}`, {
        replace: true,
      });
    } else {
      await refresh();
      markEditorDraftSaved();
    }
  } catch (e: any) {
    toast.add({
      title: "操作失败",
      description: e?.data?.message || "请检查发布条件(标题+正文非空)",
      color: "error",
    });
  } finally {
    busy.value = "";
  }
}
const trashing = ref(false);
async function moveToTrash() {
  trashing.value = true;
  try {
    await call(`/api/v1/posts/${postId.value}`, { method: "DELETE" });
    markEditorDraftSaved();
    await navigateTo("/manage/posts?status=trash");
  } catch (e: any) {
    toast.add({
      title: "移入回收站失败",
      description: e?.data?.message || "请重试",
      color: "error",
    });
    trashing.value = false;
  }
}

const statusMeta: Record<
  string,
  { label: string; color: "neutral" | "success" | "warning"; icon: string }
> = {
  draft: { label: "草稿", color: "neutral", icon: "i-tabler-pencil" },
  published: {
    label: "已发布",
    color: "success",
    icon: "i-tabler-circle-check",
  },
  private: { label: "私密", color: "warning", icon: "i-tabler-lock" },
  archived: { label: "已归档", color: "warning", icon: "i-tabler-archive" },
};
const sm = computed(
  () => statusMeta[post.value?.status || "draft"] || statusMeta.draft!,
);
const settingsSections = computed(() => [
  {
    label: "封面与摘要",
    value: "presentation",
    slot: "presentation",
    meta: post.value?.coverUrl ? "已设置封面" : "未设置封面",
  },
  {
    label: "分类与系列",
    value: "organization",
    slot: "organization",
    meta: `${selectedCategoryCount.value} 个分类 · ${selectedTags.value.length} 个标签`,
  },
  {
    label: "发布设置",
    value: "publishing",
    slot: "publishing",
    meta: sm.value.label,
  },
  {
    label: "搜索优化",
    value: "seo",
    slot: "seo",
    meta: seo.metaTitle || seo.metaDesc ? "已自定义" : "使用文章默认值",
  },
]);

type PostEditorDraftState = {
  title: string;
  slug: string;
  excerpt: string;
  content: string;
  publishedAtLocal: string;
  taxonomyIds: string[];
  seriesId: string;
  seriesOrder: number;
  pinned: boolean;
  featured: boolean;
  seo: typeof seo;
};

const editorDraftState = computed<PostEditorDraftState>({
  get: () => ({
    title: form.title,
    slug: form.slug,
    excerpt: form.excerpt,
    content: form.content,
    publishedAtLocal: publishedAtLocal.value,
    taxonomyIds: [...selected.value],
    seriesId: seriesId.value,
    seriesOrder: Number(seriesOrder.value) || 0,
    pinned: pinned.value,
    featured: featured.value,
    seo: { ...seo },
  }),
  set: (draft) => {
    form.title = draft.title;
    form.slug = draft.slug;
    form.excerpt = draft.excerpt;
    form.content = draft.content;
    publishedAtLocal.value = draft.publishedAtLocal;
    selected.value = [...draft.taxonomyIds];
    seriesId.value = draft.seriesId;
    seriesOrder.value = draft.seriesOrder;
    pinned.value = draft.pinned;
    featured.value = draft.featured;
    Object.assign(seo, draft.seo);
  },
});

const {
  autoSavedLabel: editorDraftSavedLabel,
  showDraftRestore: showEditorDraftRestore,
  savedDraft: savedEditorDraft,
  hasUnsavedChanges: hasUnsavedEditorChanges,
  restoreDraft: restoreEditorDraft,
  discardDraft: discardEditorDraft,
  markSaved: markEditorDraftSaved,
  saveNow: saveEditorDraftNow,
  startAutoSave: startEditorDraftAutoSave,
} = useEditorDraft(editorDraftState, {
  mode: "edit",
  entityId: postId,
  keyPrefix: "blog:post-editor",
  hasInitialContent: true,
});

function restorePostEditorDraft() {
  restoreEditorDraft();
  nextTick(autoGrowTitle);
}

onMounted(async () => {
  await nextTick();
  await new Promise<void>((resolve) =>
    requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
  );
  startEditorDraftAutoSave();
});
onBeforeRouteLeave(() => {
  if (!hasUnsavedEditorChanges.value) return true;
  saveEditorDraftNow();
  return window.confirm("还有未保存的更改，确定离开文章编辑页吗？");
});

// ── immersive shell: settings drawer + preview + keyboard shortcuts ───────────
// Writing fills the screen; the low-frequency settings (cover / taxonomies /
// series / flags / excerpt / SEO) live in a slide-over, summoned by ⚙ or ⌘/Ctrl+,.
const settingsOpen = ref(false);
const previewing = ref(false);
async function preview() {
  if (!post.value || previewing.value) return;
  const previewWindow = window.open("about:blank", "_blank");
  if (previewWindow) previewWindow.opener = null;
  previewing.value = true;
  try {
    const savedSlug = await saveCurrent(false);
    if (!savedSlug) {
      previewWindow?.close();
      return;
    }
    const previewURL = new URL(
      `/posts/${encodeURIComponent(savedSlug)}`,
      window.location.origin,
    ).toString();
    if (previewWindow) previewWindow.location.replace(previewURL);
    else window.open(previewURL, "_blank", "noopener,noreferrer");
    if (savedSlug !== route.params.slug) {
      await navigateTo(`/manage/posts/${encodeURIComponent(savedSlug)}`, {
        replace: true,
      });
    } else {
      await refresh();
      markEditorDraftSaved();
    }
  } finally {
    previewing.value = false;
  }
}
// Title is a borderless textarea that wraps + auto-grows (long titles shouldn't
// clip like a single-line input) — the immersive-editor title pattern.
const titleEl = ref<HTMLTextAreaElement>();
function autoGrowTitle() {
  const el = titleEl.value;
  if (!el) return;
  el.style.height = "0px";
  el.style.height = `${el.scrollHeight}px`;
}
watch(
  () => form.title,
  () => nextTick(autoGrowTitle),
);
onMounted(() => nextTick(autoGrowTitle));
// Both meta (⌘ on macOS) and ctrl (Windows/Linux) so Save works everywhere.
// usingInput keeps them live while the title/editor has focus.
defineShortcuts({
  meta_s: {
    usingInput: true,
    handler: () => {
      save();
    },
  },
  ctrl_s: {
    usingInput: true,
    handler: () => {
      save();
    },
  },
  "meta_,": {
    usingInput: true,
    handler: () => {
      settingsOpen.value = !settingsOpen.value;
    },
  },
  "ctrl_,": {
    usingInput: true,
    handler: () => {
      settingsOpen.value = !settingsOpen.value;
    },
  },
  meta_shift_p: { usingInput: true, handler: () => preview() },
  ctrl_shift_p: { usingInput: true, handler: () => preview() },
});
</script>

<template>
  <div class="min-h-full min-w-0 bg-default" data-blog-post-editor>
    <div
      class="sticky top-0 z-30 flex min-h-16 items-center justify-between gap-2 border-b border-default bg-default px-3 py-1.5 sm:gap-4 sm:px-4 sm:py-2 lg:px-8"
      data-blog-editor-commandbar
    >
      <div class="flex min-w-0 items-center gap-2">
        <UDashboardSidebarToggle class="size-11 sm:size-8 lg:hidden" />
        <UTooltip text="返回文章列表">
          <UButton
            to="/manage/posts"
            icon="i-tabler-arrow-left"
            color="neutral"
            variant="ghost"
            square
            class="size-11 sm:size-8"
            aria-label="返回文章列表"
          />
        </UTooltip>
        <span
          class="hidden max-w-[min(32vw,28rem)] truncate text-sm font-semibold text-toned md:block"
        >
          文章编辑
        </span>
        <template v-if="post">
          <span class="hidden h-5 w-px bg-accented sm:block" />
          <UBadge
            class="hidden sm:inline-flex"
            :color="sm.color"
            :icon="sm.icon"
            :label="sm.label"
            variant="subtle"
          />
          <span
            v-if="hasUnsavedEditorChanges"
            class="hidden text-xs text-warning lg:inline"
          >
            {{ editorDraftSavedLabel || "未保存" }}
          </span>
        </template>
      </div>

      <div v-if="post" class="flex shrink-0 items-center gap-1.5">
        <UTooltip text="预览文章 (⌘/Ctrl ⇧ P)">
          <UButton
            icon="i-tabler-eye"
            color="neutral"
            variant="ghost"
            square
            class="size-11 sm:size-8"
            :loading="previewing"
            aria-label="预览文章"
            @click="preview"
          />
        </UTooltip>
        <UTooltip text="文章设置 (⌘/Ctrl ,)">
          <UButton
            icon="i-tabler-adjustments-horizontal"
            color="neutral"
            variant="ghost"
            square
            class="size-11 sm:size-8"
            aria-label="文章设置"
            @click="void (settingsOpen = true)"
          />
        </UTooltip>
        <UButton
          v-if="post.status !== 'published'"
          label="发布"
          icon="i-tabler-rocket"
          color="primary"
          variant="soft"
          class="min-h-11 sm:min-h-8"
          :loading="busy === 'published'"
          @click="setStatus('published')"
        />
        <ActionFeedbackButton
          :status="saveStatus"
          idle-label="保存"
          pending-label="保存中"
          success-label="已保存"
          class="min-h-11 sm:min-h-8"
          @click="save"
        />
      </div>
    </div>

    <div
      v-if="!mounted || (pending && !post)"
      class="mx-auto max-w-4xl space-y-5 py-8"
      aria-label="正在加载文章编辑器"
    >
      <USkeleton class="h-16 w-4/5 rounded-xl" />
      <USkeleton class="h-[34rem] rounded-xl" />
    </div>

    <main
      v-else-if="post"
      class="px-4 pb-12 pt-6 sm:px-6 sm:pb-16 sm:pt-8 lg:px-8 lg:pt-10"
    >
      <section
        class="mx-auto w-full max-w-6xl rounded-xl bg-elevated p-3 shadow-sm sm:rounded-2xl sm:p-4 lg:p-6"
        data-blog-editor-document
        aria-label="文章正文编辑"
      >
        <header class="mb-5 px-1">
          <textarea
            ref="titleEl"
            v-model="form.title"
            rows="1"
            placeholder="未命名文章"
            class="blog-editor-title block w-full resize-none overflow-hidden border-0 bg-transparent font-display text-[1.75rem] font-bold leading-[1.12] tracking-[-0.04em] text-highlighted outline-none placeholder:text-dimmed sm:text-[clamp(2rem,3vw,2.25rem)]"
            aria-label="文章标题"
            @input="autoGrowTitle"
          />
          <div
            class="mt-3 flex min-h-9 items-center gap-1.5 rounded-xl border border-default bg-default/80 px-2.5 py-1.5 text-xs"
          >
            <UIcon name="i-tabler-link" class="size-4 shrink-0 text-primary" />
            <span class="shrink-0 text-dimmed">/posts/</span>
            <input
              v-model="form.slug"
              placeholder="url-slug"
              class="min-w-0 flex-1 bg-transparent text-toned outline-none transition placeholder:text-dimmed focus:text-highlighted"
              aria-label="文章永久链接"
              @input="slugTouched = true"
            />
            <UIcon
              v-if="slugTouched"
              name="i-tabler-edit"
              class="size-3.5 shrink-0 text-dimmed"
            />
          </div>
        </header>

        <UAlert
          v-if="showEditorDraftRestore"
          title="发现未保存的本地草稿"
          icon="i-tabler-device-floppy"
          color="warning"
          variant="subtle"
          class="mb-4"
        >
          <template #description>
            <div
              class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
            >
              <span class="text-sm text-muted">
                {{
                  savedEditorDraft?.savedAt
                    ? `保存于 ${new Date(savedEditorDraft.savedAt).toLocaleString()}`
                    : "包含尚未提交到服务器的编辑"
                }}
              </span>
              <div class="flex gap-2">
                <UButton
                  label="恢复草稿"
                  size="sm"
                  color="primary"
                  variant="soft"
                  @click="restorePostEditorDraft"
                />
                <UButton
                  label="忽略"
                  size="sm"
                  color="neutral"
                  variant="ghost"
                  @click="discardEditorDraft"
                />
              </div>
            </div>
          </template>
        </UAlert>

        <ContentEditor
          ref="editorComp"
          v-model="form.content"
          class="blog-editor-rich-text [&>div>.rounded-xl]:border-default [&>div>.rounded-xl]:bg-muted [&_[data-slot=content]]:mx-auto [&_[data-slot=content]]:min-h-[28rem] [&_[data-slot=content]]:w-full [&_[data-slot=content]]:px-[1.125rem] [&_[data-slot=content]]:py-6 sm:[&_[data-slot=content]]:min-h-[max(40rem,calc(100svh-19rem))] sm:[&_[data-slot=content]]:px-[clamp(2rem,4vw,3rem)] sm:[&_[data-slot=content]]:py-9"
          :image-uploader="uploadInlineImage"
          :draft-enabled="false"
          :allow-heading-one="false"
        />
      </section>
    </main>

    <USlideover
      v-model:open="settingsOpen"
      title="文章设置"
      :ui="{
        content: 'blog-editor-settings-surface w-full max-w-2xl bg-default',
        header: 'bg-default',
        body: 'bg-muted p-4 sm:p-5',
        footer: 'bg-default',
      }"
    >
      <template #body>
        <div v-if="post" class="blog-editor-settings">
          <UAccordion
            :items="settingsSections"
            type="single"
            default-value="presentation"
            :unmount-on-hide="false"
            class="overflow-hidden rounded-[0.625rem] bg-default ring-1 ring-default"
            :ui="{
              item: 'blog-editor-settings-section overflow-hidden border-b border-default last:border-b-0',
              trigger:
                'min-h-14 rounded-none px-3.5 py-3 transition-colors hover:bg-muted sm:min-h-15 sm:px-4',
              content: 'border-t border-muted',
              body: 'p-0',
            }"
            data-blog-settings-accordion
          >
            <template #default="{ item }">
              <span
                class="flex min-w-0 flex-1 items-center justify-between gap-2 text-sm font-semibold text-highlighted sm:gap-4"
              >
                <span>{{ item.label }}</span>
                <span class="truncate text-xs font-normal text-muted">
                  {{ item.meta }}
                </span>
              </span>
            </template>

            <template #presentation>
              <div class="bg-default px-3.5 pb-4 pt-4 sm:px-4 sm:pb-[1.125rem]">
                <div
                  class="grid grid-cols-[6.5rem_minmax(0,1fr)] gap-3 sm:grid-cols-[8rem_minmax(0,1fr)] sm:gap-4"
                >
                  <div
                    class="relative grid aspect-[3/2] w-[6.5rem] place-items-center overflow-hidden rounded-lg border border-default bg-muted sm:w-32"
                    data-blog-cover-preview
                    :style="{
                      aspectRatio: `${coverAspectWidth} / ${coverAspectHeight}`,
                    }"
                  >
                    <img
                      v-if="post.coverUrl"
                      :src="post.coverUrl"
                      alt="文章封面"
                      class="size-full object-cover"
                    />
                    <UIcon
                      v-else
                      name="i-tabler-photo"
                      class="size-6 text-dimmed"
                    />
                    <div
                      v-if="coverPct >= 0"
                      class="absolute inset-0 grid place-items-center bg-default/85"
                    >
                      <span class="text-sm font-semibold text-primary"
                        >{{ coverPct }}%</span
                      >
                    </div>
                  </div>
                  <div
                    class="flex min-w-0 flex-col items-start justify-center gap-2"
                  >
                    <div class="flex items-center gap-2">
                      <span class="text-sm font-medium text-highlighted">{{
                        post.coverUrl ? "文章封面" : "暂无封面"
                      }}</span>
                      <span class="text-xs text-muted">{{
                        coverAspectLabel
                      }}</span>
                    </div>
                    <UButton
                      icon="i-tabler-upload"
                      :label="post.coverUrl ? '更换图片' : '选择图片'"
                      color="neutral"
                      variant="outline"
                      size="sm"
                      :disabled="coverPct >= 0"
                      @click="coverInput?.click()"
                    />
                  </div>
                </div>
                <input
                  ref="coverInput"
                  type="file"
                  accept="image/*"
                  class="hidden"
                  @change="onPickCover"
                />

                <UFormField label="摘要" class="mt-5">
                  <UTextarea
                    v-model="form.excerpt"
                    :rows="3"
                    autoresize
                    :maxrows="6"
                    class="w-full"
                    placeholder="留空时使用正文开头"
                  />
                </UFormField>
              </div>
            </template>

            <template #organization>
              <div class="bg-default px-3.5 pb-4 pt-4 sm:px-4 sm:pb-[1.125rem]">
                <div class="grid gap-6 sm:grid-cols-2">
                  <section class="min-w-0 space-y-3" aria-labelledby="post-category-label">
                    <div
                      class="mb-2 flex min-h-7 items-center justify-between gap-2"
                    >
                      <span id="post-category-label" class="text-sm font-medium text-highlighted"
                        >分类</span
                      >
                      <UButton
                        v-if="canManageTaxonomy"
                        label="新建"
                        icon="i-tabler-plus"
                        size="xs"
                        color="neutral"
                        variant="ghost"
                        @click="openCategoryCreate()"
                      />
                    </div>
                    <USelectMenu
                      v-model="selectedCategoryIds"
                      :items="categoryOptions"
                      value-key="value"
                      label-key="label"
                      multiple
                      :virtualize="{ estimateSize: 40, overscan: 8 }"
                      :filter-fields="['label', 'description', 'searchText']"
                      :search-input="{ placeholder: '搜索分类名称或路径…' }"
                      :create-item="
                        canManageTaxonomy
                          ? { position: 'bottom', when: 'empty' }
                          : false
                      "
                      placeholder="搜索并选择分类"
                      aria-label="选择文章分类"
                      class="w-full"
                      @create="openCategoryCreate"
                    >
                      <template #default>
                        <span
                          :class="
                            selectedCategoryCount
                              ? 'text-highlighted'
                              : 'text-dimmed'
                          "
                        >
                          {{
                            selectedCategoryCount
                              ? `已选 ${selectedCategoryCount} 个分类`
                              : "搜索并选择分类"
                          }}
                        </span>
                      </template>
                      <template #empty>没有匹配的分类</template>
                      <template #create-item-label="{ item }">
                        新建分类“{{ item }}”
                      </template>
                    </USelectMenu>
                    <div
                      v-if="selectedCategories.length"
                      class="flex flex-wrap gap-1.5"
                      aria-label="已选分类"
                    >
                      <UButton
                        v-for="category in selectedCategories"
                        :key="category.id"
                        :label="category.name"
                        :aria-label="`移除分类：${category.name}`"
                        trailing-icon="i-tabler-x"
                        color="neutral"
                        variant="soft"
                        size="xs"
                        @click="toggleTax(category.id)"
                      />
                    </div>
                    <p v-else class="text-xs text-dimmed">未选择分类</p>
                  </section>

                  <section class="min-w-0 space-y-3" aria-labelledby="post-tag-label">
                    <div
                      class="mb-2 flex min-h-7 items-center justify-between gap-2"
                    >
                      <span id="post-tag-label" class="text-sm font-medium text-highlighted"
                        >标签</span
                      >
                      <UButton
                        v-if="canManageTaxonomy"
                        label="新建"
                        icon="i-tabler-plus"
                        size="xs"
                        color="neutral"
                        variant="ghost"
                        @click="openTagCreate()"
                      />
                    </div>
                    <USelectMenu
                      v-model="selectedTagIds"
                      :items="tagOptions"
                      value-key="value"
                      label-key="label"
                      multiple
                      :virtualize="{ estimateSize: 40, overscan: 8 }"
                      :filter-fields="['label', 'description', 'searchText']"
                      :search-input="{ placeholder: '搜索标签名称或 slug…' }"
                      :create-item="
                        canManageTaxonomy
                          ? { position: 'bottom', when: 'empty' }
                          : false
                      "
                      placeholder="搜索并选择标签"
                      aria-label="选择文章标签"
                      class="w-full"
                      @create="openTagCreate"
                    >
                      <template #default>
                        <span
                          :class="
                            selectedTags.length
                              ? 'text-highlighted'
                              : 'text-dimmed'
                          "
                        >
                          {{
                            selectedTags.length
                              ? `已选 ${selectedTags.length} 个标签`
                              : "搜索并选择标签"
                          }}
                        </span>
                      </template>
                      <template #empty>没有匹配的标签</template>
                      <template #create-item-label="{ item }">
                        新建标签“{{ item }}”
                      </template>
                    </USelectMenu>
                    <div
                      v-if="selectedTags.length"
                      class="flex flex-wrap gap-1.5"
                      aria-label="已选标签"
                    >
                      <UButton
                        v-for="tag in selectedTags"
                        :key="tag.id"
                        size="xs"
                        color="primary"
                        variant="soft"
                        :label="`#${tag.name}`"
                        :aria-label="`移除标签：${tag.name}`"
                        trailing-icon="i-tabler-x"
                        @click="toggleTax(tag.id)"
                      />
                    </div>
                    <p v-else class="text-xs text-dimmed">未选择标签</p>
                  </section>
                </div>

                <div class="my-5 border-t border-muted" />

                <div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_8rem]">
                  <UFormField label="系列">
                    <USelectMenu
                      v-model="seriesId"
                      :items="seriesItems"
                      value-key="value"
                      placeholder="选择系列"
                      :search-input="{ placeholder: '搜索系列…' }"
                      class="w-full"
                    />
                  </UFormField>
                  <UFormField v-if="seriesId !== NO_SERIES" label="连载序号">
                    <UInput
                      v-model="seriesOrder"
                      type="number"
                      class="w-full"
                    />
                  </UFormField>
                </div>
                <div class="mt-3 flex gap-2">
                  <UInput
                    v-model="newSeriesName"
                    size="sm"
                    placeholder="新建系列"
                    class="flex-1"
                    @keyup.enter="createSeries"
                  />
                  <UButton
                    icon="i-tabler-plus"
                    size="sm"
                    color="neutral"
                    variant="outline"
                    :loading="creatingSeries"
                    aria-label="创建系列"
                    @click="createSeries"
                  />
                </div>
              </div>
            </template>

            <template #publishing>
              <div class="bg-default px-3.5 pb-4 pt-4 sm:px-4 sm:pb-[1.125rem]">
                <div
                  class="flex items-center justify-between gap-3 rounded-lg bg-muted px-4 py-3"
                >
                  <span class="text-sm font-medium text-highlighted"
                    >当前状态</span
                  >
                  <UBadge
                    :color="sm.color"
                    :icon="sm.icon"
                    :label="sm.label"
                    variant="subtle"
                  />
                </div>
                <div class="mt-3 grid gap-2 sm:grid-cols-2">
                  <UButton
                    v-if="post.status !== 'draft'"
                    label="转回草稿"
                    icon="i-tabler-pencil"
                    color="neutral"
                    variant="outline"
                    :loading="busy === 'draft'"
                    block
                    @click="setStatus('draft')"
                  />
                  <UButton
                    v-if="post.status !== 'archived'"
                    label="归档"
                    icon="i-tabler-archive"
                    color="warning"
                    variant="outline"
                    :loading="busy === 'archived'"
                    block
                    @click="setStatus('archived')"
                  />
                </div>

                <div class="my-5 border-t border-muted" />
                <div>
                  <UFormField label="发布日期">
                    <UInput
                      v-model="publishedAtLocal"
                      type="datetime-local"
                      :disabled="post.status !== 'published'"
                      class="w-full"
                    />
                  </UFormField>
                </div>

                <template v-if="canManageFlags">
                  <div class="my-5 border-t border-muted" />
                  <div class="divide-y divide-default">
                    <label
                      class="flex min-h-12 items-center justify-between gap-3"
                    >
                      <span class="flex items-center gap-2 text-sm text-default"
                        ><UIcon
                          name="i-tabler-pin"
                          class="size-4 text-muted"
                        />置顶</span
                      >
                      <USwitch v-model="pinned" />
                    </label>
                    <label
                      class="flex min-h-12 items-center justify-between gap-3"
                    >
                      <span class="flex items-center gap-2 text-sm text-default"
                        ><UIcon
                          name="i-tabler-sparkles"
                          class="size-4 text-muted"
                        />首页精选</span
                      >
                      <USwitch v-model="featured" />
                    </label>
                  </div>
                </template>
              </div>
            </template>

            <template #seo>
              <div class="bg-default px-3.5 pb-4 pt-4 sm:px-4 sm:pb-[1.125rem]">
                <div class="mb-4 flex justify-end">
                  <UButton
                    label="从文章填充"
                    icon="i-tabler-wand"
                    color="neutral"
                    variant="outline"
                    size="sm"
                    @click="fillSearchMetadata"
                  />
                </div>
                <div class="grid gap-4 sm:grid-cols-2">
                  <UFormField label="Meta 标题"
                    ><UInput v-model="seo.metaTitle" class="w-full"
                  /></UFormField>
                  <UFormField label="OG 标题"
                    ><UInput v-model="seo.ogTitle" class="w-full"
                  /></UFormField>
                  <UFormField label="Meta 描述" class="sm:col-span-2"
                    ><UTextarea v-model="seo.metaDesc" :rows="3" class="w-full"
                  /></UFormField>
                  <UFormField label="Canonical URL" class="sm:col-span-2"
                    ><UInput
                      v-model="seo.canonicalUrl"
                      class="w-full"
                      placeholder="https://…"
                  /></UFormField>
                  <UFormField label="OG 图片 URL" class="sm:col-span-2"
                    ><UInput
                      v-model="seo.ogImage"
                      class="w-full"
                      placeholder="https://…"
                  /></UFormField>
                  <UFormField label="Robots"
                    ><UInput
                      v-model="seo.robots"
                      class="w-full"
                      placeholder="index, follow"
                  /></UFormField>
                </div>
              </div>
            </template>
          </UAccordion>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full items-center justify-between gap-3">
          <UButton
            label="移入回收站"
            icon="i-tabler-trash"
            color="neutral"
            variant="ghost"
            class="min-h-11 text-muted hover:text-error sm:min-h-8"
            :loading="trashing"
            @click="moveToTrash"
          />
          <div class="flex items-center gap-2">
            <UButton
              label="关闭"
              color="neutral"
              variant="outline"
              class="min-h-11 sm:min-h-8"
              @click="void (settingsOpen = false)"
            />
            <ActionFeedbackButton
              :status="saveStatus"
              idle-label="保存"
              pending-label="保存中"
              success-label="已保存"
              class="min-h-11 sm:min-h-8"
              @click="save"
            />
          </div>
        </div>
      </template>
    </USlideover>

    <AssetImageProcessor
      :open="!!imageProcessingRequest"
      :file="imageProcessingRequest?.file"
      :purpose="imageProcessingRequest?.purpose"
      :title="
        imageProcessingRequest?.purpose === 'cover'
          ? '处理文章封面'
          : '处理正文图片'
      "
      :initial-aspect-ratio="
        imageProcessingRequest?.purpose === 'cover'
          ? coverAspectRatio
          : undefined
      "
      :initial-aspect-ratio-label="
        imageProcessingRequest?.purpose === 'cover'
          ? coverAspectLabel
          : undefined
      "
      :fixed-max-output-width="1200"
      fixed-output-type="image/webp"
      @update:open="onImageProcessorOpen"
      @processed="finishImageProcessing($event.file)"
      @cancel="cancelImageProcessing"
      @error="
        toast.add({
          title: '图片处理失败',
          description: $event,
          color: 'error',
        })
      "
    />

    <UModal
      :open="!!compressionRequest"
      title="图片超过当前上传限制"
      description="可以先转换为 JPEG 并压缩后重试。"
      :close="false"
      :dismissible="false"
      :ui="{ footer: 'justify-end' }"
    >
      <template #body>
        <div
          v-if="compressionRequest"
          class="space-y-4"
          data-blog-image-compression-dialog
        >
          <dl class="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-sm">
            <dt class="text-muted">用途</dt>
            <dd class="text-right font-medium text-highlighted">
              {{
                compressionRequest.purpose === "cover" ? "文章封面" : "正文图片"
              }}
            </dd>
            <dt class="text-muted">原始大小</dt>
            <dd class="text-right font-medium text-highlighted">
              {{ fileSizeLabel(compressionRequest.file.size) }}
            </dd>
            <dt class="text-muted">资源中心上限</dt>
            <dd class="text-right font-medium text-highlighted">
              {{
                compressionRequest.maxBytes
                  ? fileSizeLabel(compressionRequest.maxBytes)
                  : "当前配置"
              }}
            </dd>
          </dl>
          <p class="text-sm leading-6 text-muted">
            转换会使用白色填充透明区域；如果 JPEG
            仍然超限，将保留明确提示，由你自行压缩或前往资源中心调整限制。
          </p>
        </div>
      </template>
      <template #footer>
        <UButton
          label="自行处理"
          color="neutral"
          variant="outline"
          @click="finishPngCompression(false)"
        />
        <UButton
          label="转为 JPEG 并重试"
          icon="i-tabler-photo-cog"
          color="primary"
          @click="finishPngCompression(true)"
        />
      </template>
    </UModal>

    <TaxonomyCreateModal
      kind="category"
      v-model:open="showCatModal"
      :categories="categories"
      :default-name="categoryModalName"
      @created="onTaxCreated"
    />
    <TaxonomyCreateModal
      kind="tag"
      v-model:open="showTagModal"
      :default-name="tagModalName"
      @created="onTaxCreated"
    />
  </div>
</template>

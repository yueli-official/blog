<script setup lang="ts">
import { useActionFeedback } from "@yueli/ui/feedback";
import { useBlogSiteTitle } from "~/composables/useBlogSiteTitle";
import {
  blogSettingsSaveMessages,
  useBlogSettingsProtection,
} from "~/utils/manage";
import { createBlogNotifier } from "~/utils/feedback";
import { useVueSettingsWorkflow } from "@yueli/ui/settings/vue";
import { SettingSection } from "@yueli/ui/settings/pattern";
import type {
  ContactLink,
  FriendLink,
  HomeConfig,
  HomeConfigResponse,
} from "~/types";

definePageMeta({ layout: "manage", middleware: "auth" });
useSeoMeta({ title: "设置 · 控制台" });

const { can } = useMe();
const mounted = ref(false);
const canEdit = computed(
  () => mounted.value && can("blog.site_settings.manage"),
);
const { call } = useApi();
const toast = createBlogNotifier(useToast());
const route = useRoute();
const router = useRouter();
const siteTitle = useBlogSiteTitle();
const saveError = ref("");
type FriendLinkField = keyof Pick<FriendLink, "label" | "url">;
type FriendLinkError = Partial<Record<FriendLinkField, string>>;
const friendLinkErrors = ref<FriendLinkError[]>([]);
type ContactLinkField = keyof ContactLink;
type ContactLinkError = Partial<Record<ContactLinkField, string>>;
const contactLinkErrors = ref<ContactLinkError[]>([]);
const section = ref<"footer" | "site">("site");
const sectionKeys = ["footer", "site"] as const;
const settingsSections = [
  { value: "site", label: "站点", icon: "i-tabler-world" },
  { value: "footer", label: "页脚", icon: "i-tabler-layout-bottombar" },
] as const;

const form = reactive<HomeConfig>({
  eyebrow: "",
  title: "",
  subtitle: "",
  siteTitle: "",
  siteDescription: "",
  supportEmail: "",
  footerTagline: "",
  footerCopyright: "",
  friendLinks: [],
  contactLinks: [],
});
const {
  status: saveStatus,
  pending: markSaving,
  success: markSaved,
  reset: resetSave,
} = useActionFeedback();
const settingsState = useVueSettingsWorkflow({
  snapshot: () => form,
  restore: (snapshot) => Object.assign(form, snapshot),
});
onMounted(() => {
  mounted.value = true;
});
useBlogSettingsProtection(() => settingsState.dirty.value);

const {
  data,
  pending: loading,
  error: loadError,
  refresh,
} = await useAsyncData(
  "manage-blog-site-config",
  () => call<HomeConfigResponse>("/api/v1/home"),
  { server: false },
);
const showLoading = computed(() => !mounted.value || loading.value);

watch(
  data,
  (value) => {
    const cfg = value?.config;
    if (!cfg) return;
    const contactLinks = cfg.contactLinks?.length
      ? cfg.contactLinks
      : cfg.supportEmail
        ? [
            {
              value: cfg.supportEmail,
              url: `mailto:${cfg.supportEmail}`,
            },
          ]
        : [];
    Object.assign(form, cfg, {
      friendLinks: cfg.friendLinks || [],
      contactLinks,
    });
    siteTitle.value = cfg.siteTitle;
    nextTick(settingsState.capture);
  },
  { immediate: true },
);

watch(
  () => route.query.section,
  (value) => {
    section.value =
      typeof value === "string" &&
      sectionKeys.includes(value as typeof section.value)
        ? (value as typeof section.value)
        : "site";
  },
  { immediate: true },
);

watch(section, (value) => {
  if (route.query.section === value) return;
  router.replace({ query: { ...route.query, section: value } });
});

async function save() {
  if (!canEdit.value) return;
  if (!validateContactLinks() || !validateFriendLinks()) return;
  markSaving();
  saveError.value = "";
  try {
    await call<HomeConfigResponse>("/api/v1/home", {
      method: "PATCH",
      body: {
        ...form,
        supportEmail: "",
        footerTagline: form.siteDescription.trim(),
      },
    });
    await refresh();
    siteTitle.value = form.siteTitle.trim();
    settingsState.capture();
    markSaved();
  } catch (error: any) {
    resetSave();
    saveError.value = blogFailureMessage(error, "请稍后重试");
    toast.add({
      title: "设置保存失败",
      description: saveError.value,
      color: "error",
    });
  }
}

function discardChanges() {
  settingsState.discard();
  friendLinkErrors.value = [];
  contactLinkErrors.value = [];
  saveError.value = "";
  resetSave();
}

function addContactLink() {
  if (!canEdit.value || form.contactLinks.length >= 12) return;
  const index = form.contactLinks.length;
  form.contactLinks.push({ value: "", url: "" });
  contactLinkErrors.value = [];
  nextTick(() =>
    document.getElementById(`contact-link-value-${index}`)?.focus(),
  );
}

function removeContactLink(index: number) {
  if (!canEdit.value) return;
  form.contactLinks.splice(index, 1);
  contactLinkErrors.value = [];
}

function moveContactLink(index: number, offset: -1 | 1) {
  if (!canEdit.value) return;
  const nextIndex = index + offset;
  if (nextIndex < 0 || nextIndex >= form.contactLinks.length) return;
  const [item] = form.contactLinks.splice(index, 1);
  if (item) form.contactLinks.splice(nextIndex, 0, item);
  contactLinkErrors.value = [];
}

function clearContactLinkError(index: number, field: ContactLinkField) {
  if (!contactLinkErrors.value[index]?.[field]) return;
  contactLinkErrors.value[index] = {
    ...contactLinkErrors.value[index],
    [field]: undefined,
  };
}

function validContactURL(value: string) {
  if (!value.trim()) return true;
  try {
    const parsed = new URL(value);
    if (["http:", "https:"].includes(parsed.protocol)) {
      return Boolean(parsed.hostname) && !parsed.username && !parsed.password;
    }
    return parsed.protocol === "mailto:" && Boolean(parsed.pathname);
  } catch {
    return false;
  }
}

function validateContactLinks() {
  const errors = form.contactLinks.map<ContactLinkError>((item) => {
    const linkErrors: ContactLinkError = {};
    const value = item.value.trim();
    if (!value) linkErrors.value = "请输入联系方式内容";
    else if ([...value].length > 80)
      linkErrors.value = "内容不能超过 80 个字符";
    if (!validContactURL(item.url)) {
      linkErrors.url = "请输入 HTTP(S) 或 mailto 地址";
    }
    return linkErrors;
  });
  contactLinkErrors.value = errors;
  const firstInvalid = errors.findIndex((item) => Object.keys(item).length > 0);
  if (firstInvalid < 0) return true;
  nextTick(() => {
    const firstField = Object.keys(errors[firstInvalid] || {})[0];
    document
      .getElementById(`contact-link-${firstField}-${firstInvalid}`)
      ?.focus();
  });
  return false;
}

function addFriendLink() {
  if (!canEdit.value || form.friendLinks.length >= 24) return;
  const index = form.friendLinks.length;
  form.friendLinks.push({ label: "", url: "" });
  friendLinkErrors.value = [];
  nextTick(() =>
    document.getElementById(`friend-link-label-${index}`)?.focus(),
  );
}

function removeFriendLink(index: number) {
  if (!canEdit.value) return;
  form.friendLinks.splice(index, 1);
  friendLinkErrors.value = [];
}

function moveFriendLink(index: number, offset: -1 | 1) {
  if (!canEdit.value) return;
  const nextIndex = index + offset;
  if (nextIndex < 0 || nextIndex >= form.friendLinks.length) return;
  const [item] = form.friendLinks.splice(index, 1);
  if (item) form.friendLinks.splice(nextIndex, 0, item);
  friendLinkErrors.value = [];
}

function clearFriendLinkError(index: number, field: FriendLinkField) {
  if (!friendLinkErrors.value[index]?.[field]) return;
  friendLinkErrors.value[index] = {
    ...friendLinkErrors.value[index],
    [field]: undefined,
  };
}

function validFriendLinkURL(value: string) {
  try {
    const parsed = new URL(value);
    return (
      ["http:", "https:"].includes(parsed.protocol) &&
      Boolean(parsed.hostname) &&
      !parsed.username &&
      !parsed.password
    );
  } catch {
    return false;
  }
}

function validateFriendLinks() {
  const seen = new Set<string>();
  const errors = form.friendLinks.map<FriendLinkError>((item) => {
    const linkErrors: FriendLinkError = {};
    const label = item.label.trim();
    const url = item.url.trim();
    if (!label) linkErrors.label = "请输入友链名称";
    else if ([...label].length > 40)
      linkErrors.label = "名称不能超过 40 个字符";
    if (!url) linkErrors.url = "请输入友链地址";
    else if (!validFriendLinkURL(url))
      linkErrors.url = "请输入完整的 HTTP(S) 地址";
    else {
      const duplicateKey = new URL(url).href.replace(/\/$/u, "");
      if (seen.has(duplicateKey)) linkErrors.url = "该地址已经添加";
      seen.add(duplicateKey);
    }
    return linkErrors;
  });
  friendLinkErrors.value = errors;
  const firstInvalid = errors.findIndex((item) => Object.keys(item).length > 0);
  if (firstInvalid < 0) return true;
  nextTick(() => {
    const firstField = Object.keys(errors[firstInvalid] || {})[0];
    document
      .getElementById(`friend-link-${firstField}-${firstInvalid}`)
      ?.focus();
  });
  return false;
}
</script>

<template>
  <div class="space-y-5">
    <ManagePageHeader title="站点设置">
      <template #actions>
        <ManageSettingsActions
          :dirty="settingsState.dirty.value"
          :status="saveStatus"
          :disabled="!canEdit"
          :messages="blogSettingsSaveMessages"
          @discard="discardChanges"
          @save="save"
        />
      </template>
    </ManagePageHeader>
    <ManageTabbedSurface
      v-model="section"
      :items="settingsSections"
      navigation-label="设置分区"
      data-manage-surface="settings"
    >
      <div class="space-y-5 p-4 sm:p-5">
        <div v-if="mounted && !canEdit">
          <UAlert
            color="neutral"
            variant="subtle"
            icon="i-tabler-lock"
            title="只读设置"
            description="只有具备站点设置能力的角色可以修改公开配置。"
          />
        </div>

        <SettingSection v-if="showLoading" title="正在加载设置">
          <div class="grid gap-4">
            <USkeleton class="h-9 w-full" />
            <USkeleton class="h-9 w-full" />
            <USkeleton class="h-24 w-full" />
          </div>
        </SettingSection>

        <UAlert
          v-else-if="loadError"
          color="error"
          variant="subtle"
          icon="i-tabler-alert-circle"
          title="设置加载失败"
          description="站点配置尚未初始化或服务不可用，请先运行开发环境 provision。"
        />

        <template v-else-if="section === 'footer'">
        <SettingSection title="页脚内容">
          <div class="grid gap-4">
            <UFormField label="版权信息">
              <UInput
                v-model="form.footerCopyright"
                :disabled="!canEdit"
                placeholder="© 2026 Yueli"
                class="w-full"
              />
            </UFormField>
          </div>
        </SettingSection>

        <SettingSection title="联系">
          <template #actions>
            <UButton
              label="添加联系方式"
              icon="i-tabler-address-book"
              color="neutral"
              variant="soft"
              size="sm"
              :disabled="!canEdit || form.contactLinks.length >= 12"
              @click="addContactLink"
            />
          </template>

          <div
            v-if="form.contactLinks.length === 0"
            class="flex min-h-28 flex-col items-center justify-center rounded-lg border border-dashed border-muted px-4 text-center"
          >
            <UIcon name="i-tabler-address-book" class="size-5 text-dimmed" />
            <p class="mt-2 text-sm text-muted">暂未添加联系方式</p>
          </div>

          <div
            v-else
            class="overflow-hidden rounded-lg border border-muted"
            data-contact-links-editor
          >
            <article
              v-for="(link, index) in form.contactLinks"
              :key="index"
              class="border-b border-muted p-4 last:border-b-0 sm:p-5"
              data-contact-link-row
            >
              <div class="mb-4 flex items-center justify-between gap-3">
                <div class="flex min-w-0 items-center gap-2.5">
                  <span
                    class="grid size-8 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary"
                  >
                    <UIcon name="i-tabler-address-book" class="size-4" />
                  </span>
                  <span class="text-sm font-medium text-highlighted">
                    联系方式 {{ index + 1 }}
                  </span>
                </div>
                <div class="flex shrink-0 items-center gap-1">
                  <UButton
                    icon="i-tabler-arrow-up"
                    color="neutral"
                    variant="ghost"
                    size="sm"
                    :disabled="!canEdit || index === 0"
                    :aria-label="`上移联系方式 ${index + 1}`"
                    @click="moveContactLink(index, -1)"
                  />
                  <UButton
                    icon="i-tabler-arrow-down"
                    color="neutral"
                    variant="ghost"
                    size="sm"
                    :disabled="
                      !canEdit || index === form.contactLinks.length - 1
                    "
                    :aria-label="`下移联系方式 ${index + 1}`"
                    @click="moveContactLink(index, 1)"
                  />
                  <UButton
                    icon="i-tabler-trash"
                    color="error"
                    variant="ghost"
                    size="sm"
                    :disabled="!canEdit"
                    :aria-label="`删除联系方式 ${index + 1}`"
                    @click="removeContactLink(index)"
                  />
                </div>
              </div>

              <div class="grid gap-x-4 gap-y-3 sm:grid-cols-2">
                <UFormField
                  label="内容"
                  required
                  :error="contactLinkErrors[index]?.value"
                >
                  <UInput
                    :id="`contact-link-value-${index}`"
                    v-model="link.value"
                    :disabled="!canEdit"
                    maxlength="80"
                    placeholder="邮箱地址或群号"
                    class="w-full"
                    @update:model-value="clearContactLinkError(index, 'value')"
                  />
                </UFormField>
                <UFormField
                  label="链接"
                  hint="可选"
                  :error="contactLinkErrors[index]?.url"
                >
                  <UInput
                    :id="`contact-link-url-${index}`"
                    v-model="link.url"
                    :disabled="!canEdit"
                    placeholder="mailto:me@example.com 或 https://..."
                    class="w-full"
                    @update:model-value="clearContactLinkError(index, 'url')"
                  />
                </UFormField>
              </div>
            </article>
          </div>
        </SettingSection>

        <SettingSection title="友链">
          <template #actions>
            <UButton
              label="添加友链"
              icon="i-tabler-link-plus"
              color="neutral"
              variant="soft"
              size="sm"
              :disabled="!canEdit || form.friendLinks.length >= 24"
              @click="addFriendLink"
            />
          </template>

          <div
            v-if="form.friendLinks.length === 0"
            class="flex min-h-28 flex-col items-center justify-center rounded-lg border border-dashed border-muted px-4 text-center"
          >
            <UIcon name="i-tabler-link" class="size-5 text-dimmed" />
            <p class="mt-2 text-sm text-muted">暂未添加友链</p>
          </div>

          <div
            v-else
            class="overflow-hidden rounded-lg border border-muted"
            data-friend-links-editor
          >
            <article
              v-for="(link, index) in form.friendLinks"
              :key="index"
              class="border-b border-muted p-4 last:border-b-0 sm:p-5"
              data-friend-link-row
            >
              <div class="mb-4 flex items-center justify-between gap-3">
                <div class="flex min-w-0 items-center gap-2.5">
                  <span
                    class="grid size-8 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary"
                  >
                    <UIcon name="i-tabler-link" class="size-4" />
                  </span>
                  <span class="text-sm font-medium text-highlighted">
                    友链 {{ index + 1 }}
                  </span>
                </div>
                <div class="flex shrink-0 items-center gap-1">
                  <UButton
                    icon="i-tabler-arrow-up"
                    color="neutral"
                    variant="ghost"
                    size="sm"
                    :disabled="!canEdit || index === 0"
                    :aria-label="`上移友链 ${index + 1}`"
                    @click="moveFriendLink(index, -1)"
                  />
                  <UButton
                    icon="i-tabler-arrow-down"
                    color="neutral"
                    variant="ghost"
                    size="sm"
                    :disabled="
                      !canEdit || index === form.friendLinks.length - 1
                    "
                    :aria-label="`下移友链 ${index + 1}`"
                    @click="moveFriendLink(index, 1)"
                  />
                  <UButton
                    icon="i-tabler-trash"
                    color="error"
                    variant="ghost"
                    size="sm"
                    :disabled="!canEdit"
                    :aria-label="`删除友链 ${index + 1}`"
                    @click="removeFriendLink(index)"
                  />
                </div>
              </div>

              <div class="grid gap-x-4 gap-y-3 sm:grid-cols-2">
                <UFormField
                  label="内容"
                  required
                  :error="friendLinkErrors[index]?.label"
                >
                  <UInput
                    :id="`friend-link-label-${index}`"
                    v-model="link.label"
                    :disabled="!canEdit"
                    maxlength="40"
                    placeholder="站点名称"
                    class="w-full"
                    @update:model-value="clearFriendLinkError(index, 'label')"
                  />
                </UFormField>
                <UFormField
                  label="地址"
                  required
                  :error="friendLinkErrors[index]?.url"
                >
                  <UInput
                    :id="`friend-link-url-${index}`"
                    v-model="link.url"
                    :disabled="!canEdit"
                    type="url"
                    placeholder="https://example.com"
                    class="w-full"
                    @update:model-value="clearFriendLinkError(index, 'url')"
                  />
                </UFormField>
              </div>
            </article>
          </div>
        </SettingSection>
        </template>

        <template v-else>
        <SettingSection title="站点信息">
          <div class="grid max-w-3xl gap-y-4">
            <UFormField label="站点名称" required>
              <UInput
                v-model="form.siteTitle"
                :disabled="!canEdit"
                class="w-full"
              />
            </UFormField>
            <UFormField label="站点描述">
              <UTextarea
                v-model="form.siteDescription"
                :disabled="!canEdit"
                :rows="3"
                class="w-full"
              />
            </UFormField>
          </div>
        </SettingSection>
        </template>
      </div>
    </ManageTabbedSurface>
  </div>
</template>

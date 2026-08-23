<script setup lang="ts">
interface RoleView {
  key: string;
  displayName: string;
  kind: string;
  protected: boolean;
  capabilities: string[];
  assignmentSources: string[];
}

interface ApplicationView {
  id: string;
  subject: string;
  role: string;
  reason: string;
}

interface GrantView {
  id: string;
  subject: string;
  role: string;
  source: string;
}

interface ConsoleView {
  activeRevision: number;
  policy: { number: number; state: string };
  roles: RoleView[];
  automaticRules: { key: string; enabled: boolean }[];
  applications: ApplicationView[];
  grants: GrantView[];
  capabilities: { key: string; displayName: string }[];
}

interface PublicUser {
  userKey: string;
  handle: string;
  displayName: string;
}

interface AuthorizationUserRow {
  subject: string;
  grants: GrantView[];
}

type AuthorizationTab = "applications" | "permissions" | "users";

definePageMeta({ layout: "manage", middleware: "auth" });
useSeoMeta({ title: "权限与申请" });

const PAGE_SIZE = 20;
const { call } = useApi();
const { isAdministrator } = useMe();
const toast = useToast();
const busy = ref(false);
const mounted = ref(false);
const activeTab = ref<AuthorizationTab>("applications");
const createRoleOpen = ref(false);
const grantOpen = ref(false);
const expandedRoleKey = ref("");
const applicationQuery = ref("");
const applicationRole = ref("all");
const applicationPage = ref(1);
const selectedApplications = ref<string[]>([]);
const userQuery = ref("");
const userRole = ref("all");
const userPage = ref(1);
const selectedUsers = ref<string[]>([]);
const identityUsers = ref<Record<string, PublicUser>>({});

const roleForm = reactive({
  key: "",
  displayName: "",
  capabilities: [] as string[],
  assignmentSources: ["application", "invitation", "direct"] as string[],
});
const grantForm = reactive({ subject: "", role: "author" });

const { data, pending, error, refresh } = await useAsyncData(
  "blog-authorization-console",
  () => call<ConsoleView>("/api/v1/authorization/manage/console"),
  { server: false },
);

const state = computed(() => data.value);
const draft = computed(() => state.value?.policy.state === "draft");
const tabItems = computed(() => [
  {
    label: "申请",
    value: "applications",
    icon: "i-tabler-inbox",
    badge: state.value?.applications.length || undefined,
  },
  { label: "权限", value: "permissions", icon: "i-tabler-shield-lock" },
  {
    label: "用户管理",
    value: "users",
    icon: "i-tabler-users",
    badge: userRows.value.length || undefined,
  },
]);
const roleOptions = computed(() => [
  { label: "全部角色", value: "all" },
  ...(state.value?.roles || []).map((role) => ({ label: role.displayName, value: role.key })),
]);
const grantRoleOptions = computed(() => (state.value?.roles || [])
  .filter((role) => role.kind !== "custom" || role.capabilities.length)
  .map((role) => ({ label: role.displayName, value: role.key })));

const subjectKey = computed(() => {
  const subjects = [
    ...(state.value?.applications || []).map((item) => item.subject),
    ...(state.value?.grants || []).map((item) => item.subject),
  ];
  return [...new Set(subjects.filter(Boolean))].sort().join(",");
});

const filteredApplications = computed(() => {
  const query = applicationQuery.value.trim().toLocaleLowerCase();
  return (state.value?.applications || []).filter((application) => {
    if (applicationRole.value !== "all" && application.role !== applicationRole.value) return false;
    if (!query) return true;
    return [
      application.subject,
      userName(application.subject),
      identityUsers.value[application.subject]?.handle,
      application.reason,
      roleLabel(application.role),
    ].some((value) => value?.toLocaleLowerCase().includes(query));
  });
});
const pagedApplications = computed(() => filteredApplications.value.slice(
  (applicationPage.value - 1) * PAGE_SIZE,
  applicationPage.value * PAGE_SIZE,
));
const allApplicationsSelected = computed(() => pagedApplications.value.length > 0
  && pagedApplications.value.every((item) => selectedApplications.value.includes(item.id)));

const userRows = computed<AuthorizationUserRow[]>(() => {
  const rows = new Map<string, GrantView[]>();
  for (const grant of state.value?.grants || []) {
    rows.set(grant.subject, [...(rows.get(grant.subject) || []), grant]);
  }
  return [...rows.entries()]
    .map(([subject, grants]) => ({ subject, grants }))
    .sort((left, right) => userName(left.subject).localeCompare(userName(right.subject), "zh-CN"));
});
const filteredUsers = computed(() => {
  const query = userQuery.value.trim().toLocaleLowerCase();
  return userRows.value.filter((user) => {
    if (userRole.value !== "all" && !user.grants.some((grant) => grant.role === userRole.value)) return false;
    if (!query) return true;
    return [user.subject, userName(user.subject), identityUsers.value[user.subject]?.handle]
      .some((value) => value?.toLocaleLowerCase().includes(query));
  });
});
const pagedUsers = computed(() => filteredUsers.value.slice(
  (userPage.value - 1) * PAGE_SIZE,
  userPage.value * PAGE_SIZE,
));
const allUsersSelected = computed(() => pagedUsers.value.length > 0
  && pagedUsers.value.every((item) => selectedUsers.value.includes(item.subject)));

watch(subjectKey, async (value) => {
  if (!import.meta.client || !value) return;
  const subjects = value.split(",");
  const users: Record<string, PublicUser> = { ...identityUsers.value };
  try {
    for (let index = 0; index < subjects.length; index += 100) {
      const result = await $fetch<{ users: PublicUser[] }>("/identity-api/api/v1/users", {
        query: { ids: subjects.slice(index, index + 100).join(",") },
      });
      for (const user of result.users || []) users[user.userKey] = user;
    }
    identityUsers.value = users;
  } catch {
    // Identity enrichment is best-effort. Authorization remains manageable by subject.
  }
}, { immediate: true });

watch([applicationQuery, applicationRole], () => {
  applicationPage.value = 1;
  selectedApplications.value = [];
});
watch([userQuery, userRole], () => {
  userPage.value = 1;
  selectedUsers.value = [];
});

onMounted(() => {
  mounted.value = true;
});

function roleLabel(key: string) {
  return state.value?.roles.find((role) => role.key === key)?.displayName || key;
}

function userName(subject: string) {
  return identityUsers.value[subject]?.displayName || identityUsers.value[subject]?.handle || subject;
}

function userMeta(subject: string) {
  const user = identityUsers.value[subject];
  if (!user) return subject;
  return user.handle ? `@${user.handle} · ${subject}` : subject;
}

function sourceLabel(source: string) {
  return ({
    application: "申请批准",
    invitation: "邀请加入",
    direct: "直接授予",
    automatic: "自动授权",
    bootstrap: "系统初始化",
  } as Record<string, string>)[source] || source;
}

async function mutate(task: () => Promise<unknown>) {
  if (busy.value) return;
  busy.value = true;
  try {
    const result = await task();
    if (result === false) return;
    await refresh();
  } catch (failure) {
    const message = failure instanceof Error
      ? failure.message
      : (failure as { data?: { message?: string } }).data?.message;
    toast.add({
      title: "操作失败",
      description: message || "请刷新后重试。",
      color: "error",
      icon: "i-tabler-alert-circle",
    });
  } finally {
    busy.value = false;
  }
}

function createDraft() {
  if (!state.value?.activeRevision) return;
  return mutate(() => call("/api/v1/authorization/manage/policies/drafts", {
    method: "POST",
    body: { expectedActiveRevision: state.value!.activeRevision },
  }));
}

function toggleRoleCapability(role: RoleView, capability: string) {
  if (!draft.value || role.protected) return;
  const capabilities = role.capabilities.includes(capability)
    ? role.capabilities.filter((item) => item !== capability)
    : [...role.capabilities, capability];
  return mutate(() => call(
    `/api/v1/authorization/manage/policies/${state.value!.policy.number}/roles/${role.key}/capabilities`,
    { method: "PUT", body: { capabilities } },
  ));
}

function toggleAutomatic(enabled: boolean) {
  const rule = state.value?.automaticRules[0];
  if (!draft.value || !rule) return;
  return mutate(() => call(
    `/api/v1/authorization/manage/policies/${state.value!.policy.number}/automatic/${rule.key}`,
    { method: "PUT", body: { enabled } },
  ));
}

async function validateAndActivate() {
  const current = state.value;
  if (!draft.value || !current) return;
  await mutate(async () => {
    const validation = await call<{ valid: boolean; violations: string[] }>(
      `/api/v1/authorization/manage/policies/${current.policy.number}/validate`,
      { method: "POST" },
    );
    if (!validation.valid) throw new Error(validation.violations.join("；"));
    const impact = await call<{ removedBindings: number }>(
      `/api/v1/authorization/manage/policies/${current.policy.number}/preview`,
      { method: "POST" },
    );
    if (impact.removedBindings > 0
      && !window.confirm(`本次发布会移除 ${impact.removedBindings} 项能力绑定，是否继续？`)) return false;
    await call(`/api/v1/authorization/manage/policies/${current.policy.number}/activate`, {
      method: "POST",
      body: { expectedActiveRevision: current.activeRevision },
    });
  });
}

function reviewOne(application: ApplicationView, decision: "approve" | "reject") {
  return mutate(() => reviewRequest(application.id, decision));
}

function reviewRequest(id: string, decision: "approve" | "reject") {
  return call(`/api/v1/authorization/manage/applications/${id}/review`, {
    method: "POST",
    body: { decision, reason: decision === "approve" ? "管理员批准" : "管理员拒绝" },
  });
}

function reviewSelected(decision: "approve" | "reject") {
  const ids = [...selectedApplications.value];
  if (!ids.length) return;
  return mutate(async () => {
    await Promise.all(ids.map((id) => reviewRequest(id, decision)));
    selectedApplications.value = [];
  });
}

function toggleApplication(id: string, selected: boolean) {
  selectedApplications.value = selected
    ? [...new Set([...selectedApplications.value, id])]
    : selectedApplications.value.filter((item) => item !== id);
}

function toggleAllApplications(selected: boolean) {
  const pageIds = pagedApplications.value.map((item) => item.id);
  selectedApplications.value = selected
    ? [...new Set([...selectedApplications.value, ...pageIds])]
    : selectedApplications.value.filter((id) => !pageIds.includes(id));
}

function createRole() {
  if (!draft.value || !state.value) return;
  return mutate(async () => {
    await call(`/api/v1/authorization/manage/policies/${state.value!.policy.number}/roles`, {
      method: "POST",
      body: roleForm,
    });
    createRoleOpen.value = false;
    Object.assign(roleForm, {
      key: "",
      displayName: "",
      capabilities: [],
      assignmentSources: ["application", "invitation", "direct"],
    });
  });
}

function grantRole() {
  if (!grantForm.subject.trim() || !grantForm.role) return;
  return mutate(async () => {
    await call("/api/v1/authorization/manage/grants", {
      method: "POST",
      body: { subject: grantForm.subject.trim(), role: grantForm.role },
    });
    grantForm.subject = "";
    grantOpen.value = false;
  });
}

function revokeGrant(grant: GrantView) {
  if (!window.confirm(`确定撤销 ${userName(grant.subject)} 的${roleLabel(grant.role)}角色吗？`)) return;
  return mutate(() => call(`/api/v1/authorization/manage/grants/${grant.id}`, { method: "DELETE" }));
}

function toggleUser(subject: string, selected: boolean) {
  selectedUsers.value = selected
    ? [...new Set([...selectedUsers.value, subject])]
    : selectedUsers.value.filter((item) => item !== subject);
}

function toggleAllUsers(selected: boolean) {
  const pageSubjects = pagedUsers.value.map((item) => item.subject);
  selectedUsers.value = selected
    ? [...new Set([...selectedUsers.value, ...pageSubjects])]
    : selectedUsers.value.filter((subject) => !pageSubjects.includes(subject));
}

function revokeSelectedUsers() {
  const subjects = [...selectedUsers.value];
  const grants = (state.value?.grants || []).filter((grant) => subjects.includes(grant.subject));
  if (!grants.length) return;
  if (!window.confirm(`确定撤销所选 ${subjects.length} 位用户的全部本站角色吗？`)) return;
  return mutate(async () => {
    await Promise.all(grants.map((grant) => call(
      `/api/v1/authorization/manage/grants/${grant.id}`,
      { method: "DELETE" },
    )));
    selectedUsers.value = [];
  });
}
</script>

<template>
  <div id="authorization" class="w-full space-y-5">
    <ManagePageHeader title="权限与申请">
      <template #actions>
        <UButton
          v-if="mounted && state && activeTab === 'permissions' && !draft"
          label="编辑权限"
          icon="i-tabler-pencil"
          :loading="busy"
          @click="createDraft"
        />
        <UButton
          v-else-if="mounted && activeTab === 'permissions' && draft"
          label="验证并发布"
          icon="i-tabler-rocket"
          :loading="busy"
          @click="validateAndActivate"
        />
        <UButton
          v-else-if="mounted && activeTab === 'users'"
          label="添加用户"
          icon="i-tabler-user-plus"
          @click="() => { grantOpen = true }"
        />
      </template>
    </ManagePageHeader>

    <ClientOnly>
      <UAlert
        v-if="!isAdministrator"
        color="error"
        icon="i-tabler-lock"
        title="只有管理员可以管理本站权限"
        description="博客权限独立存储，不继承用户中心或其他站点的管理员角色。"
      />
      <SkeletonList v-else-if="pending" :rows="8" />
      <UAlert
        v-else-if="error || !state"
        color="error"
        icon="i-tabler-alert-circle"
        title="权限配置加载失败"
        description="请检查 Blog API 与本站数据库状态。"
      />

      <ManageTabbedSurface
        v-else
        v-model="activeTab"
        :items="tabItems"
        navigation-label="权限管理"
        data-manage-surface="authorization"
      >
        <div v-if="activeTab === 'applications'" class="min-w-0">
          <div class="grid gap-3 border-b border-default p-4 sm:grid-cols-[minmax(0,1fr)_14rem] sm:px-5">
            <UInput
              v-model="applicationQuery"
              icon="i-tabler-search"
              placeholder="搜索申请人或原因"
              class="w-full"
            />
            <USelect
              v-model="applicationRole"
              :items="roleOptions"
              value-key="value"
              class="w-full"
              aria-label="筛选申请角色"
            />
          </div>

          <div
            v-if="selectedApplications.length"
            class="flex flex-wrap items-center justify-between gap-3 border-b border-default bg-primary/5 px-4 py-3 sm:px-5"
            data-authorization-application-bulk
          >
            <span class="text-sm font-medium text-highlighted">已选择 {{ selectedApplications.length }} 项申请</span>
            <div class="flex items-center gap-2">
              <UButton
                label="批量拒绝"
                color="neutral"
                variant="outline"
                size="sm"
                :disabled="busy"
                @click="reviewSelected('reject')"
              />
              <UButton label="批量批准" size="sm" :disabled="busy" @click="reviewSelected('approve')" />
            </div>
          </div>

          <div v-if="filteredApplications.length" class="min-w-0">
            <div class="flex items-center border-b border-default px-4 py-3 sm:px-5">
              <UCheckbox
                :model-value="allApplicationsSelected"
                label="选择本页"
                @update:model-value="toggleAllApplications(Boolean($event))"
              />
            </div>
            <div class="divide-y divide-default">
              <article
                v-for="application in pagedApplications"
                :key="application.id"
                class="grid min-w-0 gap-3 px-4 py-4 sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-center sm:px-5"
              >
                <UCheckbox
                  :model-value="selectedApplications.includes(application.id)"
                  :aria-label="`选择 ${userName(application.subject)} 的申请`"
                  @update:model-value="toggleApplication(application.id, Boolean($event))"
                />
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="truncate text-sm font-medium text-highlighted">{{ userName(application.subject) }}</p>
                    <UBadge :label="roleLabel(application.role)" color="neutral" variant="soft" />
                  </div>
                  <p class="mt-1 truncate text-xs text-muted">{{ userMeta(application.subject) }}</p>
                  <p v-if="application.reason" class="mt-2 text-sm text-muted">{{ application.reason }}</p>
                </div>
                <div class="ml-7 flex items-center gap-2 sm:ml-0">
                  <UButton
                    label="拒绝"
                    color="neutral"
                    variant="ghost"
                    size="sm"
                    :disabled="busy"
                    @click="reviewOne(application, 'reject')"
                  />
                  <UButton label="批准" size="sm" :disabled="busy" @click="reviewOne(application, 'approve')" />
                </div>
              </article>
            </div>
            <div
              v-if="filteredApplications.length > PAGE_SIZE"
              class="flex justify-end border-t border-default px-4 py-3 sm:px-5"
            >
              <UPagination
                v-model:page="applicationPage"
                :total="filteredApplications.length"
                :items-per-page="PAGE_SIZE"
              />
            </div>
          </div>
          <ManageEmpty v-else icon="i-tabler-inbox" text="当前没有匹配的申请" class="m-5" />
        </div>

        <div v-else-if="activeTab === 'permissions'" class="min-w-0">
          <div v-if="draft" class="border-b border-default p-4 sm:px-5">
            <UAlert
              color="warning"
              variant="subtle"
              icon="i-tabler-pencil"
              title="权限更改尚未发布"
              description="调整完成后，使用页面右上角的“验证并发布”。"
            />
          </div>

          <section>
            <div class="flex flex-wrap items-center justify-between gap-3 border-b border-default px-4 py-4 sm:px-5">
              <h2 class="text-sm font-semibold text-highlighted">角色与能力</h2>
              <UButton
                v-if="draft"
                label="新建自定义角色"
                icon="i-tabler-plus"
                color="neutral"
                variant="outline"
                size="sm"
                @click="() => { createRoleOpen = true }"
              />
            </div>
            <div class="divide-y divide-default">
              <article v-for="role in state.roles" :key="role.key" class="min-w-0 px-4 py-4 sm:px-5">
                <button
                  type="button"
                  class="flex w-full items-center gap-3 text-left"
                  :aria-expanded="expandedRoleKey === role.key"
                  @click="expandedRoleKey = expandedRoleKey === role.key ? '' : role.key"
                >
                  <span class="grid size-9 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary">
                    <UIcon name="i-tabler-shield-lock" class="size-5" />
                  </span>
                  <span class="min-w-0 flex-1">
                    <span class="flex flex-wrap items-center gap-2">
                      <span class="font-medium text-highlighted">{{ role.displayName }}</span>
                      <UBadge
                        :label="role.protected ? '受保护' : role.kind === 'custom' ? '自定义' : '内置'"
                        color="neutral"
                        variant="soft"
                      />
                    </span>
                    <span class="mt-1 block text-xs text-muted">{{ role.capabilities.length }} 项能力</span>
                  </span>
                  <UIcon
                    name="i-tabler-chevron-down"
                    class="size-5 shrink-0 text-muted transition-transform"
                    :class="expandedRoleKey === role.key ? 'rotate-180' : ''"
                  />
                </button>
                <div
                  v-if="expandedRoleKey === role.key"
                  class="mt-4 grid gap-3 border-t border-default pt-4 sm:ml-12 sm:grid-cols-2 xl:grid-cols-3"
                >
                  <UCheckbox
                    v-for="capability in state.capabilities"
                    :key="capability.key"
                    :model-value="role.capabilities.includes(capability.key)"
                    :label="capability.displayName"
                    :disabled="role.protected || !draft || busy"
                    @update:model-value="toggleRoleCapability(role, capability.key)"
                  />
                </div>
              </article>
            </div>
          </section>

          <section class="border-t border-default px-4 py-5 sm:px-5">
            <div class="flex items-center justify-between gap-4">
              <div>
                <h2 class="text-sm font-semibold text-highlighted">注册用户自动成为作者</h2>
                <p class="mt-1 text-xs text-muted">开启后，新用户首次进入博客时获得作者角色。</p>
              </div>
              <USwitch
                :model-value="state.automaticRules[0]?.enabled ?? false"
                :disabled="!draft || busy"
                @update:model-value="toggleAutomatic(Boolean($event))"
              />
            </div>
          </section>
        </div>

        <div v-else class="min-w-0">
          <div class="grid gap-3 border-b border-default p-4 sm:grid-cols-[minmax(0,1fr)_14rem] sm:px-5">
            <UInput v-model="userQuery" icon="i-tabler-search" placeholder="搜索用户" class="w-full" />
            <USelect
              v-model="userRole"
              :items="roleOptions"
              value-key="value"
              class="w-full"
              aria-label="筛选用户角色"
            />
          </div>

          <div
            v-if="selectedUsers.length"
            class="flex flex-wrap items-center justify-between gap-3 border-b border-default bg-primary/5 px-4 py-3 sm:px-5"
            data-authorization-user-bulk
          >
            <span class="text-sm font-medium text-highlighted">已选择 {{ selectedUsers.length }} 位用户</span>
            <UButton
              label="批量撤销角色"
              color="error"
              variant="soft"
              size="sm"
              :disabled="busy"
              @click="revokeSelectedUsers"
            />
          </div>

          <div v-if="filteredUsers.length" class="min-w-0">
            <div class="flex items-center border-b border-default px-4 py-3 sm:px-5">
              <UCheckbox
                :model-value="allUsersSelected"
                label="选择本页"
                @update:model-value="toggleAllUsers(Boolean($event))"
              />
            </div>
            <div class="divide-y divide-default">
              <article
                v-for="user in pagedUsers"
                :key="user.subject"
                class="grid min-w-0 gap-3 px-4 py-4 sm:grid-cols-[auto_minmax(12rem,0.8fr)_minmax(16rem,1.2fr)] sm:items-center sm:px-5"
                data-authorization-user-row
              >
                <UCheckbox
                  :model-value="selectedUsers.includes(user.subject)"
                  :aria-label="`选择 ${userName(user.subject)}`"
                  @update:model-value="toggleUser(user.subject, Boolean($event))"
                />
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium text-highlighted">{{ userName(user.subject) }}</p>
                  <p class="mt-1 truncate text-xs text-muted">{{ userMeta(user.subject) }}</p>
                </div>
                <div class="ml-7 flex min-w-0 flex-wrap gap-2 sm:ml-0 sm:justify-end">
                  <span
                    v-for="grant in user.grants"
                    :key="grant.id"
                    class="inline-flex min-w-0 items-center gap-1 rounded-md bg-elevated px-2 py-1 text-xs text-default ring-1 ring-inset ring-default"
                  >
                    <span>{{ roleLabel(grant.role) }}</span>
                    <span class="hidden text-muted lg:inline">· {{ sourceLabel(grant.source) }}</span>
                    <UButton
                      icon="i-tabler-x"
                      color="neutral"
                      variant="ghost"
                      size="xs"
                      :aria-label="`撤销 ${userName(user.subject)} 的${roleLabel(grant.role)}角色`"
                      :disabled="busy"
                      @click="revokeGrant(grant)"
                    />
                  </span>
                </div>
              </article>
            </div>
            <div
              v-if="filteredUsers.length > PAGE_SIZE"
              class="flex justify-end border-t border-default px-4 py-3 sm:px-5"
            >
              <UPagination v-model:page="userPage" :total="filteredUsers.length" :items-per-page="PAGE_SIZE" />
            </div>
          </div>
          <ManageEmpty v-else icon="i-tabler-users" text="当前没有匹配的用户" class="m-5" />
        </div>
      </ManageTabbedSurface>

      <template #fallback>
        <SkeletonList :rows="8" />
      </template>
    </ClientOnly>

    <UModal v-model:open="grantOpen" title="添加用户" description="为用户授予本站角色。">
      <template #body>
        <div class="space-y-4">
          <UFormField label="用户标识" required>
            <UInput v-model="grantForm.subject" placeholder="Identity 用户 ID" class="w-full" />
          </UFormField>
          <UFormField label="角色" required>
            <USelect v-model="grantForm.role" value-key="value" :items="grantRoleOptions" class="w-full" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton label="取消" color="neutral" variant="outline" @click="() => { grantOpen = false }" />
          <UButton
            label="添加用户"
            icon="i-tabler-user-plus"
            :disabled="!grantForm.subject.trim() || busy"
            :loading="busy"
            @click="grantRole"
          />
        </div>
      </template>
    </UModal>

    <UModal v-model:open="createRoleOpen" title="新建自定义角色" description="配置完成并发布后生效。">
      <template #body>
        <div class="space-y-4">
          <UFormField label="角色标识" required>
            <UInput v-model="roleForm.key" placeholder="editor" class="w-full" />
          </UFormField>
          <UFormField label="显示名称" required>
            <UInput v-model="roleForm.displayName" placeholder="编辑" class="w-full" />
          </UFormField>
          <UFormField label="能力">
            <div class="grid gap-2 sm:grid-cols-2">
              <UCheckbox
                v-for="capability in state?.capabilities ?? []"
                :key="capability.key"
                :model-value="roleForm.capabilities.includes(capability.key)"
                :label="capability.displayName"
                @update:model-value="
                  roleForm.capabilities = $event
                    ? [...roleForm.capabilities, capability.key]
                    : roleForm.capabilities.filter((item) => item !== capability.key)
                "
              />
            </div>
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton label="取消" color="neutral" variant="outline" @click="() => { createRoleOpen = false }" />
          <UButton
            label="创建角色"
            :disabled="!roleForm.key.trim() || !roleForm.displayName.trim()"
            :loading="busy"
            @click="createRole"
          />
        </div>
      </template>
    </UModal>
  </div>
</template>

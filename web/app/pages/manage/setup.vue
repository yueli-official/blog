<script setup lang="ts">
definePageMeta({ layout: false, middleware: "auth" });
useSeoMeta({ title: "初始化站点" });

const siteTitle = useBlogSiteTitle();
const { user } = useAuth();
const { call } = useApi();
const { status, refresh, markClaimed } = useAdministratorClaim();
const { refreshMe } = useMe();
const phase = ref<"idle" | "claiming" | "complete">("idle");
const conflict = ref(false);
const failure = ref(false);

const identity = computed(() => user.value?.name || user.value?.email || "当前账号");
const initial = computed(() => identity.value.trim().charAt(0).toUpperCase() || "用");
const actionLabel = computed(() => {
  if (phase.value === "claiming") return "初始化中";
  if (phase.value === "complete") return "已完成";
  return "初始化站点并成为管理员";
});

async function claimAdministrator() {
  if (phase.value !== "idle") return;
  phase.value = "claiming";
  conflict.value = false;
  failure.value = false;
  try {
    await call<{ claimed: boolean; created: boolean }>(
      "/api/v1/authorization/setup/claim",
      { method: "POST" },
    );
    phase.value = "complete";
    markClaimed();
    await refreshMe();
    await navigateTo("/manage");
  } catch (error: unknown) {
    const code = (error as { data?: { code?: string }; code?: string })?.data?.code
      || (error as { code?: string })?.code;
    const latest = await refresh().catch(() => null);
    if (code === "blog.initial_administrator_already_claimed" || latest?.claimed) {
      conflict.value = true;
    } else {
      failure.value = true;
    }
    phase.value = "idle";
  }
}
</script>

<template>
  <div class="min-h-svh bg-muted/35 text-default">
    <header class="border-b border-default bg-default/95">
      <div class="mx-auto flex min-h-16 max-w-5xl items-center justify-between gap-4 px-4 sm:px-6">
        <NuxtLink to="/" class="flex min-w-0 items-center gap-3 text-highlighted">
          <span class="grid size-9 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary ring-1 ring-primary/20">
            <UIcon name="i-tabler-feather" class="size-5" />
          </span>
          <strong class="truncate text-base">{{ siteTitle }}</strong>
        </NuxtLink>
        <ConsumerManageAccountControl home-to="/" show-appearance trigger-mode="inline" />
      </div>
    </header>

    <main class="mx-auto grid min-h-[calc(100svh-4rem)] max-w-5xl place-items-center px-4 py-10 sm:px-6">
      <UCard class="yueli-card w-full max-w-xl bg-elevated shadow-sm" :ui="{ body: 'p-6 sm:p-8' }">
        <div class="space-y-7">
          <div class="space-y-3">
            <span class="grid size-12 place-items-center rounded-2xl bg-primary/10 text-primary ring-1 ring-primary/20">
              <UIcon name="i-tabler-shield-check" class="size-6" />
            </span>
            <div class="space-y-1.5">
              <h1 class="font-display text-2xl font-bold tracking-[-0.025em] text-highlighted sm:text-3xl">
                初始化站点
              </h1>
              <p class="text-sm leading-6 text-muted">
                本站尚未设置管理员。完成后，当前账号将获得本站全部管理权限；此操作仅可成功一次。
              </p>
            </div>
          </div>

          <div class="flex items-center gap-3 rounded-xl border border-default bg-default px-4 py-3">
            <UAvatar :text="initial" size="md" class="shrink-0" />
            <div class="min-w-0">
              <p class="truncate text-sm font-semibold text-highlighted">{{ identity }}</p>
              <p v-if="user?.email && user.email !== identity" class="truncate text-xs text-muted">
                {{ user.email }}
              </p>
            </div>
          </div>

          <UAlert
            v-if="conflict"
            color="warning"
            icon="i-tabler-user-shield"
            title="本站已由另一位用户完成初始化"
            description="当前账号没有获得管理员权限。"
          />
          <UAlert
            v-else-if="failure"
            color="error"
            icon="i-tabler-alert-circle"
            title="初始化失败"
            description="请检查服务状态后重试。"
          />

          <div class="flex flex-wrap justify-end gap-2">
            <UButton v-if="conflict" to="/" color="neutral" variant="outline" label="返回首页" />
            <UButton
              v-else
              :label="actionLabel"
              icon="i-tabler-sparkles"
              size="lg"
              :loading="phase === 'claiming'"
              :disabled="phase === 'complete' || status?.claimed"
              @click="claimAdministrator"
            />
          </div>
        </div>
      </UCard>
    </main>
  </div>
</template>

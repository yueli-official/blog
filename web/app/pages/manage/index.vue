<script setup lang="ts">
import DashboardTrendChart from "~/components/DashboardTrendChart.vue";
import type { DashboardOverview, MyComments, MyPosts, PostView } from "~/types";

definePageMeta({ layout: "manage", middleware: "auth" });
useSeoMeta({ title: "控制台" });
const { call } = useApi();
const { creating, createDraft } = useCreatePostDraft();
const period = ref(14);
const periodItems = [
  { label: "7 天", value: 7 },
  { label: "14 天", value: 14 },
  { label: "30 天", value: 30 },
];
const dashboardCardClass = "bg-default shadow-sm";

const {
  data: posts,
  pending: postsPending,
  error: postsError,
} = await useAsyncData(
  "dashboard-posts",
  () => call<MyPosts>("/api/v1/posts/mine", { query: { page: 1, size: 1 } }),
  {
    server: false,
    default: () => ({
      items: [] as PostView[],
      total: 0,
      page: 1,
      size: 1,
      counts: {} as Record<string, number>,
      totalViews: 0,
    }),
  },
);
const { data: pendingCommentsData, error: commentsError } = await useAsyncData(
  "dashboard-comments",
  () =>
    call<MyComments>("/api/v1/comments/mine", {
      query: { status: 2, page: 1, size: 1 },
    }),
  { server: false, default: () => ({ items: [], total: 0, page: 1, size: 1 }) },
);
const {
  data: analytics,
  pending: analyticsPending,
  error: analyticsError,
} = await useAsyncData(
  "dashboard-analytics",
  () =>
    call<DashboardOverview>("/api/v1/dashboard/overview", {
      query: { days: period.value },
    }),
  {
    server: false,
    watch: [period],
    default: () => ({
      days: period.value,
      allTimeViews: 0,
      allTimeUniqueVisitorDays: 0,
      periodViews: 0,
      periodUniqueVisitorDays: 0,
      previousPeriodViews: 0,
      previousUniqueVisitorDays: 0,
      series: [],
      topPosts: [],
      topSources: [],
    }),
  },
);

const mounted = ref(false);
onMounted(() => {
  mounted.value = true;
});
const showSkeleton = useMinimumLoading(
  computed(
    () =>
      !mounted.value ||
      postsPending.value ||
      (analyticsPending.value && !analytics.value?.series.length),
  ),
);
const counts = computed<Record<string, number>>(
  () => posts.value?.counts ?? {},
);
const draftCount = computed(() => counts.value.draft ?? 0);
const pendingComments = computed(() => pendingCommentsData.value?.total ?? 0);
const dashboardError = computed(
  () => postsError.value || commentsError.value || analyticsError.value,
);
const topPeak = computed(() =>
  Math.max(1, ...(analytics.value?.topPosts ?? []).map((post) => post.views)),
);
const sourcePeak = computed(() =>
  Math.max(
    1,
    ...(analytics.value?.topSources ?? []).map((source) => source.views),
  ),
);

const formatter = new Intl.NumberFormat("zh-CN");
function formatNumber(value: number) {
  return formatter.format(value);
}
function formatMetricValue(value: number) {
  return Number.isInteger(value) ? formatNumber(value) : value.toFixed(1);
}
function comparison(current: number, previous: number, unit = "") {
  const suffix = unit ? ` ${unit}` : "";
  if (!previous)
    return current
      ? `本期新增 ${formatNumber(current)}${suffix}`
      : "与前期持平";
  const percent = Math.round(((current - previous) / previous) * 100);
  return `${percent >= 0 ? "+" : ""}${percent}% 较前 ${period.value} 天`;
}
const readingDepth = computed(() => {
  const visitors = analytics.value?.periodUniqueVisitorDays ?? 0;
  return visitors ? (analytics.value?.periodViews ?? 0) / visitors : 0;
});
const visitorAverage = computed(
  () => (analytics.value?.periodUniqueVisitorDays ?? 0) / period.value,
);
const visitorPeakPoint = computed(() =>
  (analytics.value?.series ?? []).reduce<
    DashboardOverview["series"][number] | undefined
  >(
    (peak, point) =>
      !peak || point.uniqueVisitorDays > peak.uniqueVisitorDays ? point : peak,
    undefined,
  ),
);
const visitorComparison = computed(() =>
  comparison(
    analytics.value?.periodUniqueVisitorDays ?? 0,
    analytics.value?.previousUniqueVisitorDays ?? 0,
    "访客日",
  ),
);
const visitorComparisonColor = computed(() =>
  (analytics.value?.periodUniqueVisitorDays ?? 0) >=
  (analytics.value?.previousUniqueVisitorDays ?? 0)
    ? ("success" as const)
    : ("warning" as const),
);
function sourceLabel(value: string) {
  const labels: Record<string, string> = {
    direct: "直接访问",
    "google.com": "Google",
    "bing.com": "Bing",
    "baidu.com": "百度",
    "zhihu.com": "知乎",
    "x.com": "X",
    "weibo.com": "微博",
  };
  return labels[value] ?? value;
}
const metricCards = computed(() => [
  {
    label: "累计浏览",
    value: analytics.value?.allTimeViews ?? 0,
    detail: `${formatNumber(analytics.value?.allTimeUniqueVisitorDays ?? 0)} 访客日`,
    icon: "i-tabler-eye",
    tone: "blue" as const,
    to: "/manage/posts",
  },
  {
    label: `近 ${period.value} 天浏览`,
    value: analytics.value?.periodViews ?? 0,
    detail: comparison(
      analytics.value?.periodViews ?? 0,
      analytics.value?.previousPeriodViews ?? 0,
    ),
    icon: "i-tabler-chart-line",
    tone: "blue-soft" as const,
    to: "/manage",
  },
  {
    label: "浏览深度",
    value: readingDepth.value,
    detail: "平均每访客日浏览",
    icon: "i-tabler-chart-dots-3",
    tone: "blue-deep" as const,
    to: "/manage",
  },
  {
    label: "已发布文章",
    value: counts.value.published ?? 0,
    detail: `${draftCount.value} 篇草稿待继续`,
    icon: "i-tabler-circle-check",
    tone: "neutral" as const,
    to: "/manage/posts?status=published",
  },
]);
</script>

<template>
  <div class="space-y-5" data-dashboard-analytics>
    <ManagePageHeader title="控制台">
      <template #actions>
        <UButton
          icon="i-tabler-plus"
          label="写新文章"
          :loading="creating"
          @click="createDraft"
        />
      </template>
    </ManagePageHeader>

    <UAlert
      v-if="dashboardError"
      color="error"
      variant="subtle"
      icon="i-tabler-alert-circle"
      title="统计暂时不可用"
      description="内容管理仍可使用；刷新后仍失败时再检查 Blog API。"
    />
    <UAlert
      v-if="pendingComments"
      color="warning"
      variant="soft"
      orientation="horizontal"
      icon="i-tabler-message-exclamation"
      :title="`${pendingComments} 条评论待审核`"
    >
      <template #actions>
        <UButton
          to="/manage/comments"
          color="warning"
          variant="soft"
          label="审核评论"
          trailing-icon="i-tabler-arrow-right"
        />
      </template>
    </UAlert>

    <div v-if="showSkeleton" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <USkeleton v-for="item in 4" :key="item" class="h-32 rounded-xl" />
    </div>
    <div v-else class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <ManageMetricCard
        v-for="card in metricCards"
        :key="card.label"
        :label="card.label"
        :value="formatMetricValue(card.value)"
        :detail="card.detail"
        :to="card.to"
        :icon="card.icon"
        :tone="card.tone"
        data-dashboard-metric
      />
    </div>

    <div
      class="grid items-start gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(18rem,1fr)]"
    >
      <UCard
        variant="soft"
        :class="[dashboardCardClass, 'divide-y-0']"
        data-dashboard-trend
        :aria-busy="analyticsPending"
        :ui="{
          header: 'p-5 sm:p-6',
          body: 'px-5 pb-5 pt-0 sm:px-6 sm:pb-6 sm:pt-0',
        }"
      >
        <template #header>
          <div
            class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between"
          >
            <h2 class="text-base font-semibold text-highlighted">浏览趋势</h2>
            <UTabs
              v-model="period"
              :items="periodItems"
              :content="false"
              :disabled="!mounted || analyticsPending"
              size="sm"
              class="w-fit"
              aria-label="统计时间范围"
            />
          </div>
        </template>
        <DashboardTrendChart :points="analytics?.series ?? []" />
      </UCard>

      <UCard
        title="访客趋势"
        variant="soft"
        :class="[dashboardCardClass, 'divide-y-0']"
        data-dashboard-audience
        :ui="{
          header: 'p-5 sm:p-6',
          body: 'space-y-5 px-5 pb-5 pt-0 sm:px-6 sm:pb-6 sm:pt-0',
        }"
      >
        <div class="flex items-end justify-between gap-4">
          <div>
            <p
              class="text-3xl font-semibold tabular-nums tracking-tight text-highlighted"
            >
              {{ formatNumber(analytics?.periodUniqueVisitorDays ?? 0) }}
            </p>
            <p class="mt-1 text-xs text-muted">访客日</p>
          </div>
          <UBadge
            :color="visitorComparisonColor"
            variant="soft"
            :label="visitorComparison"
          />
        </div>
        <DashboardTrendChart
          :points="analytics?.series ?? []"
          metric="uniqueVisitorDays"
          compact
        />
        <dl class="grid grid-cols-2 gap-4">
          <div>
            <dt class="text-xs text-muted">日均访客日</dt>
            <dd
              class="mt-1 text-lg font-semibold tabular-nums text-highlighted"
            >
              {{ visitorAverage.toFixed(1) }}
            </dd>
          </div>
          <div>
            <dt class="text-xs text-muted">单日峰值</dt>
            <dd
              class="mt-1 text-lg font-semibold tabular-nums text-highlighted"
            >
              {{ formatNumber(visitorPeakPoint?.uniqueVisitorDays ?? 0) }}
            </dd>
          </div>
        </dl>
      </UCard>
    </div>

    <div
      class="grid items-start gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(18rem,1fr)]"
    >
      <UCard
        title="热门文章"
        variant="soft"
        :class="[dashboardCardClass, 'divide-y-0']"
        data-dashboard-top-posts
        :ui="{ header: 'p-5 sm:p-6', body: 'p-0 sm:p-0' }"
      >
        <div v-if="analytics?.topPosts.length" class="divide-y divide-default">
          <NuxtLink
            v-for="(post, index) in analytics.topPosts"
            :key="post.id"
            :to="`/manage/posts/${post.slug}`"
            class="group grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 px-5 py-4 transition hover:bg-accented sm:px-6"
          >
            <UBadge color="neutral" variant="soft" :label="String(index + 1)" />
            <div class="min-w-0 space-y-2">
              <p
                class="truncate text-sm font-medium text-highlighted group-hover:text-primary"
              >
                {{ post.title }}
              </p>
              <UProgress :model-value="post.views" :max="topPeak" size="2xs" />
              <p class="text-xs text-muted">
                {{ post.uniqueVisitorDays }} 访客日
              </p>
            </div>
            <div class="text-end">
              <p class="text-lg font-semibold tabular-nums text-highlighted">
                {{ formatNumber(post.views) }}
              </p>
              <p class="text-xs text-dimmed">次浏览</p>
            </div>
          </NuxtLink>
        </div>
        <div v-else class="p-8 text-center text-sm text-muted">
          当前周期还没有文章浏览记录。
        </div>
      </UCard>

      <div class="space-y-4">
        <UCard
          title="流量来源"
          variant="soft"
          :class="[dashboardCardClass, 'divide-y-0']"
          data-dashboard-sources
          :ui="{
            header: 'p-5 sm:p-6',
            body: 'space-y-4 px-5 pb-5 pt-0 sm:px-6 sm:pb-6 sm:pt-0',
          }"
        >
          <div v-if="analytics?.topSources?.length" class="space-y-4">
            <div
              v-for="source in analytics.topSources"
              :key="source.source"
              class="space-y-2"
            >
              <div class="flex items-center justify-between gap-3 text-sm">
                <span class="min-w-0 truncate text-toned">{{
                  sourceLabel(source.source)
                }}</span>
                <span
                  class="shrink-0 font-medium tabular-nums text-highlighted"
                  >{{ formatNumber(source.views) }}</span
                >
              </div>
              <UProgress
                :model-value="source.views"
                :max="sourcePeak"
                size="2xs"
              />
            </div>
          </div>
          <p v-else class="text-sm text-muted">
            从现在开始采集；当前周期还没有可归因的来源。
          </p>
        </UCard>
      </div>
    </div>
  </div>
</template>

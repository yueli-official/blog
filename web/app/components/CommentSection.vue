<script setup lang="ts">
import { PublicCommentThread } from "@yueli/ui/comments";
import type {
  PublicCommentDraft,
  PublicCommentMessages,
  PublicCommentOrder,
  PublicCommentState,
} from "@yueli/ui/comments";
import type { CommentView, CommentList } from "~/types";

// Reader-facing comment area: a two-level thread list + a compose box. Fetched
// client-side (member state matters and comments aren't SEO-critical).
const props = defineProps<{ slug: string; commentStatus: number }>();

const { call } = useApi();
const { loggedIn, user, login } = useAuth();
const { profile } = useMe();

const items = ref<CommentView[]>([]);
const total = ref(0);
const state = ref<PublicCommentState>("loading");
const order = ref<PublicCommentOrder>("asc");
const avatarSrc = useVerifiedImage(
  () => user.value?.avatar || profile.value?.avatarUrl,
);
const viewer = computed(() => ({
  authenticated: loggedIn.value,
  name:
    user.value?.name || profile.value?.displayName || user.value?.email || "",
  avatarUrl: avatarSrc.value,
}));
const messages: PublicCommentMessages = {
  count: (count) => `${count} 条评论`,
  replies: (count) => `${count} 条回复`,
  sort: "评论排序",
  loading: "正在加载评论",
  oldest: "最早",
  newest: "最新",
  reply: "回复",
  cancelReply: "取消回复",
  anonymous: "匿名用户",
  empty: "还没有评论，来说第一句吧",
  closed: "本文已关闭评论",
  loadError: "评论加载失败",
  retry: "重新加载",
  writeComment: "写下你的评论…",
  writeReply: "写下回复…",
  authorName: "昵称 *",
  authorEmail: "邮箱（选填，不公开）",
  anonymousHint: "匿名评论需要审核",
  login: "登录后免审核",
  submit: "发表评论",
  submitReply: "回复",
  submitted: "评论已发布",
  pending: "评论已提交，待审核后显示",
  submitError: "发表失败，请重试",
  nameRequired: "请填写昵称",
};

async function load() {
  state.value = "loading";
  try {
    const r = await call<CommentList>(`/api/v1/posts/${props.slug}/comments`, {
      query: { page: 1, size: 100, sortOrder: order.value },
    });
    items.value = r.items;
    total.value = r.total;
    state.value = "ready";
  } catch {
    state.value = "error";
  }
}

async function submitComment(draft: PublicCommentDraft) {
  try {
    const result = await call<{ pending: boolean }>(
      `/api/v1/posts/${props.slug}/comments`,
      {
        method: "POST",
        body: {
          content: draft.content,
          parentId: draft.parentId,
          authorName: draft.authorName,
          authorEmail: draft.authorEmail,
        },
      },
    );
    if (!result.pending) await load();
    return { pending: result.pending };
  } catch (error: any) {
    throw new Error(blogFailureMessage(error, messages.submitError));
  }
}

onMounted(load);
watch(order, load);

function formatCommentTime(value: string) {
  return rel(value);
}
</script>

<template>
  <section class="mt-16">
    <PublicCommentThread
      v-model:order="order"
      :comments="items"
      :total="total"
      :state="state"
      :viewer="viewer"
      :messages="messages"
      :format-time="formatCommentTime"
      :submit="submitComment"
      :login="login"
      :retry="load"
      :closed="commentStatus === 0"
      allow-anonymous
      input-position="bottom"
    />
  </section>
</template>

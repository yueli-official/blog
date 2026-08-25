import { createBlogNotifier } from "~/utils/feedback";
import type { PostView } from "~/types";

export function useCreatePostDraft() {
  const { call } = useApi();
  const toast = createBlogNotifier(useToast());
  const creating = ref(false);

  async function createDraft() {
    if (creating.value) return;
    creating.value = true;
    try {
      const response = await call<{ post: PostView }>("/api/v1/posts", {
        method: "POST",
        body: { title: "未命名文章" },
      });
      await navigateTo(`/manage/posts/${response.post.slug}`);
    } catch (error: any) {
      toast.add({
        title: "创建失败",
        description: error?.data?.message || "请检查网络后重试。",
        color: "error",
        icon: "i-tabler-alert-circle",
      });
    } finally {
      creating.value = false;
    }
  }

  return { creating, createDraft };
}

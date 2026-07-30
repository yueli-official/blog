import {
  normalizeFeedbackNotice,
  type FeedbackTone,
} from "@yueli/ui/feedback";

export interface BlogToastInput {
  id?: string | number;
  title?: string;
  description?: string;
  color?: string;
  duration?: number;
  type?: "foreground" | "background";
  close?: boolean;
  icon?: string;
  [key: string]: unknown;
}

const feedbackTones = new Set<FeedbackTone>([
  "neutral",
  "success",
  "info",
  "warning",
  "error",
]);

function normalizeBlogToast(input: BlogToastInput): BlogToastInput {
  const tone = feedbackTones.has(input.color as FeedbackTone)
    ? (input.color as FeedbackTone)
    : "neutral";
  const notice = normalizeFeedbackNotice({
    id: input.id,
    title: input.title,
    description: input.description,
    tone,
    duration: input.duration,
    foreground:
      input.type === undefined ? undefined : input.type === "foreground",
    close: input.close,
    icon: input.icon,
  });
  return {
    ...input,
    id: notice.id,
    color: notice.tone,
    duration: notice.duration,
    type: notice.foreground ? "foreground" : "background",
    close: notice.close,
    icon: notice.icon,
  };
}

/** 把 Foundation 的反馈合同映射为 Nuxt UI toast 输入。 */
export function createBlogNotifier<NativeToastInput>(toast: {
  add(input: NativeToastInput): unknown;
}) {
  return {
    add(input: BlogToastInput) {
      return toast.add(normalizeBlogToast(input) as NativeToastInput);
    },
  };
}

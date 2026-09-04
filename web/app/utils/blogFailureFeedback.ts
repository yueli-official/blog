import { resolveFailureFeedback, type ProblemParams } from "@yueli/http-runtime";
import { blogFailurePresentation, type BlogFailureCode } from "../generated/blogFailure";

const messages: Record<BlogFailureCode, string> = {
  "blog.abuse_attempt_replayed": "这次提交已经处理，请勿重复提交。",
  "blog.abuse_unavailable": "内容安全服务暂时不可用。",
  "blog.asset_too_large": "图片大小超过当前上传限制。",
  "blog.authorization_unavailable": "权限服务暂时不可用。",
  "blog.challenge_required": "请先完成人机验证。",
  "blog.comment_not_found": "评论不存在或已被删除。",
  "blog.comment_rejected": "评论未通过内容安全检查。",
  "blog.comments_closed": "这篇文章已关闭评论。",
  "blog.forbidden": "你没有执行此操作的权限。",
  "blog.initial_administrator_already_claimed": "站点管理员已经完成初始化。",
  "blog.invalid_input": "提交内容不符合要求。",
  "blog.invalid_state": "当前状态不允许执行此操作。",
  "blog.not_found": "请求的内容不存在。",
  "blog.rate_limited": "操作过于频繁，请稍后再试。",
  "blog.slug_taken": "这个链接地址已被使用。",
  "blog.upstream_failed": "依赖服务暂时不可用。",
};

const recoveries: Record<string, string> = {
  "recovery.retry_later": "请稍后重试。",
};

function resolveText(code: string, _params: ProblemParams) {
  if (!(code in blogFailurePresentation)) return undefined;
  const typedCode = code as BlogFailureCode;
  const presentation = blogFailurePresentation[typedCode];
  const recoveryKey = "recoveryKey" in presentation
    ? presentation.recoveryKey
    : undefined;
  return {
    message: messages[typedCode],
    ...(recoveryKey
      ? { recovery: recoveries[recoveryKey] }
      : {}),
  };
}

export function blogFailureFeedback(error: unknown, fallback: string) {
  return resolveFailureFeedback(error, { fallback, resolveText });
}

export function blogFailureMessage(error: unknown, fallback: string) {
  const feedback = blogFailureFeedback(error, fallback);
  return feedback.recovery
    ? `${feedback.message}${feedback.recovery}`
    : feedback.message;
}

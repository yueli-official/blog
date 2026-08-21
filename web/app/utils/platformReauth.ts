export interface BlogReauthOptions {
  readonly requireLoggedIn?: boolean;
}

export type BlogReauthHandler = (
  options?: BlogReauthOptions,
) => Promise<boolean>;

export function getOptionalBlogReauth(): BlogReauthHandler | undefined {
  const nuxtApp = tryUseNuxtApp() as
    | (ReturnType<typeof useNuxtApp> & {
        $platformReauth?: BlogReauthHandler;
      })
    | null;
  return nuxtApp?.$platformReauth;
}

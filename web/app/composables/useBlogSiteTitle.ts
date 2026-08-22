export function useBlogSiteTitle() {
  const { brand } = useSiteRuntime();
  return useState<string>("blog-site-title", () =>
    String(brand.value || "月离博客"),
  );
}

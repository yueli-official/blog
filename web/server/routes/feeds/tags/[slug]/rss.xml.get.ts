export default defineEventHandler((event) => {
  const slug = decodeURIComponent(getRouterParam(event, "slug") ?? "");
  return serveDiscoveryArtifact(event, `feeds/tags/${slug}/rss.xml`);
});

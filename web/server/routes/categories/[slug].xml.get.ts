export default defineEventHandler((event) => {
  const pathname = getRequestURL(event).pathname;
  const slug = decodeURIComponent(
    (pathname.split("/").pop() ?? "").replace(/\.xml$/u, ""),
  );
  return serveDiscoveryArtifact(event, `categories/${slug}.xml`);
});

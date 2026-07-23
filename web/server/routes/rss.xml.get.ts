export default defineEventHandler((event) =>
  serveDiscoveryArtifact(event, "rss.xml"),
);

import { readFileSync } from "node:fs";
import test from "node:test";
import assert from "node:assert/strict";

test("resource strategy displays the registered Asset namespace", () => {
  const config = readFileSync(
    new URL("../../nuxt.config.ts", import.meta.url),
    "utf8",
  );
  const page = readFileSync(
    new URL("../pages/manage/assets.vue", import.meta.url),
    "utf8",
  );
  assert.match(
    config,
    /assetSiteKey:\s*process\.env\.NUXT_PUBLIC_ASSET_SITE_KEY\s*\|\|\s*["']blog["']/u,
  );
  assert.match(page, /expected-namespace="blog"/u);
  assert.match(page, /AssetPolicyPage/u);
  assert.doesNotMatch(page, /ManageAssetSettings/u);
});

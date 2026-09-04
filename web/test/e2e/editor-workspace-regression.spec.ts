import { expect, test } from "@playwright/test";
import { loginE2E } from "./runtime";

test("Blog reuses the shared standard and pure-editor immersive workspace", async ({ browser }) => {
  const siteURL = process.env.BLOG_E2E_URL!;
  const context = await loginE2E(browser, { viewport: { width: 1440, height: 900 } }, undefined, siteURL);
  const list = await context.request.get(new URL("/api/v1/posts?page=1&size=10", siteURL).toString());
  expect(list.ok()).toBeTruthy();
  const posts = (await list.json()).items as Array<{ slug: string }>;
  expect(posts.length).toBeGreaterThan(0);

  const page = await context.newPage();
  await page.goto(new URL(`/manage/posts/${posts[0]!.slug}`, siteURL).toString());
  await page.waitForLoadState("load");
  await expect(page.getByRole("button", { name: "沉浸式协作" })).toBeVisible();
  await expect(page.getByLabel("摘要")).toHaveCount(0);
  await expect(page.getByLabel("路径标识")).toHaveCount(0);
  await page.locator("#manage-main").evaluate((element) => { element.scrollTop = element.scrollHeight; });
  await expect(page.getByLabel("文章标题")).toBeInViewport();
  await expect(page.getByRole("toolbar").first()).toBeInViewport();
  await page.getByRole("button", { name: "文章设置" }).click();
  await expect(page.getByLabel("摘要")).toBeVisible();
  await expect(page.getByLabel("路径标识")).toBeVisible();
  await page.getByRole("button", { name: "文章设置" }).click();

  await page.getByRole("button", { name: "沉浸式协作" }).click();
  await expect(page.locator("[data-blog-editor-workspace]")).toHaveAttribute("data-collaboration-mode", "immersive");
  await expect(page.locator("[data-blog-editor-commandbar]")).toHaveCount(0);
  await expect(page.getByLabel("文章标题")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "退出沉浸式协作" })).toBeVisible();
  await expect(page.getByRole("toolbar").first()).toBeInViewport();

  const mobile = await context.newPage();
  await mobile.setViewportSize({ width: 390, height: 844 });
  await mobile.goto(new URL(`/manage/posts/${posts[0]!.slug}`, siteURL).toString());
  await mobile.waitForLoadState("load");
  await mobile.getByRole("button", { name: "沉浸式协作" }).click();
  await expect(mobile.locator("[data-blog-editor-commandbar]")).toHaveCount(0);
  await expect(mobile.getByRole("button", { name: "退出沉浸式协作" })).toBeVisible();
  await context.close();
});

test("Blog exposes protocol-native create and no-content statuses", async ({ browser }) => {
  const siteURL = process.env.BLOG_E2E_URL!;
  const context = await loginE2E(browser, {}, undefined, siteURL);
  const created = await context.request.post(new URL("/api/v1/posts", siteURL).toString(), {
    data: { title: `HTTP Result ${Date.now()}`, content: "status contract" },
  });
  expect(created.status()).toBe(201);
  const body = await created.json() as { post: { id: string } };

  const trashed = await context.request.delete(
    new URL(`/api/v1/posts/${body.post.id}`, siteURL).toString(),
  );
  expect(trashed.status()).toBe(204);
  expect(await trashed.body()).toHaveLength(0);
  const removed = await context.request.delete(new URL(`/api/v1/posts/${body.post.id}/permanent`, siteURL).toString());
  expect(removed.status()).toBe(204);
  expect(await removed.body()).toHaveLength(0);
  await context.close();
});

import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { productSites } from "./contracts";
import { loginE2E, requiredEnv, settleNuxt } from "./runtime";

const accountURL = requiredEnv("BLOG_E2E_ACCOUNT_URL");
const e2eEmail = requiredEnv("BLOG_E2E_EMAIL");
const e2ePassword = requiredEnv("BLOG_E2E_PASSWORD");
const missing = "__platform_e2e_missing__";
const imageFixtureBase64 =
  "iVBORw0KGgoAAAANSUhEUgAAAKAAAABaCAYAAAA/xl1SAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAAJcEhZcwAADsMAAA7DAcdvqGQAAAISSURBVHhe7dRRSkNRFENRJ+PIHH4HoVxQBEmhIG1ekh1Yv333yMa394/bJ+BCgLAiQFgRIKwIEFYECCsChBUBwooAYUWAsCJAWBEgrAgQVgQIKwKEFQHCigAf8J+p38MvArzjGVPfWUeAf7xi6rurCPCbY+oda+YDvMLUu1ZMB3ilqfctmAzwylPvbTYXYMLUu1tNBZg09f5GMwEmTt3RZiLA5Kl7mhDgxafuaVIfYMPUXS2qA2yauq8BAYZM3degNsDGqTvTEWDQ1J3pKgNsnro3GQGGTd2brC7Aham7UxFg4NTdqQgwcOruVFUBLk3dn4gAQ6fuT0SAoVP3JyLA0Kn7ExFg6NT9iQgwdOr+RAQYOnV/opoA16b+Bon4Dxg6dX8iAgyduj8RAYZO3Z+IAEOn7k9EgKFT9yciwNCp+xMRYOjU/YmqAjwWpu5ORYCBU3enIsDAqbtT1QV4NE/dm4wAw6buTVYZ4NE4dWc6AgyaujNdbYBH09R9DQgwZOq+BtUBHg1Td7WoD/BInrqnCQFefOqeJhMBHolTd7SZCfBImnp/o6kAj4Spd7eaC/C48tR7m00G+ONKU+9bMB3gcYWpd62YD/CHY+odawjwj1dMfXcVAd7xjKnvrCPAB/xn6vfwiwBhRYCwIkBYESCsCBBWBAgrAoQVAcKKAGFFgLAiQFgRIKwIEFYECCsChNHt8wtKVG37Suz0VQAAAABJRU5ErkJggg==";

function captureErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") errors.push(`console: ${message.text()}`);
  });
  page.on("pageerror", (error) => errors.push(`pageerror: ${error.message}`));
  page.on("response", (response) => {
    if (response.status() >= 500)
      errors.push(`http ${response.status()}: ${response.url()}`);
  });
  return errors;
}

async function purgeTestPost(
  context: BrowserContext,
  siteURL: string,
  postId: string,
) {
  await context.request.delete(
    new URL(`/api/v1/posts/${postId}`, siteURL).toString(),
  );
  await context.request.delete(
    new URL(`/api/v1/posts/${postId}/permanent`, siteURL).toString(),
  );
}

export function registerJourneySuite(product: string) {
  for (const site of productSites(product)) {
    const contract = site.contract;
    test.describe(`${site.slug} (${site.product})`, () => {
      test("公开入口完成渲染且没有浏览器错误", async ({ page }) => {
        const errors = captureErrors(page);
        const response = await page.goto(
          new URL(contract.public.path, site.url).toString(),
          { waitUntil: "domcontentloaded" },
        );
        expect(response?.ok()).toBeTruthy();
        await expect(
          page.locator(contract.public.readySelector).first(),
        ).toBeVisible();
        await settleNuxt(page);
        expect(errors).toEqual([]);
      });

      test("首页精选封面加载后保持固定桌面构图", async ({ page }, testInfo) => {
        await page.setViewportSize({ width: 1440, height: 900 });
        let releaseMedia = () => undefined;
        const mediaGate = new Promise<void>((resolve) => {
          releaseMedia = resolve;
        });
        await page.route("**/media/**", async (route) => {
          await mediaGate;
          await route.continue();
        });
        const response = await page.goto(
          new URL(contract.public.path, site.url).toString(),
          { waitUntil: "domcontentloaded" },
        );
        expect(response?.ok()).toBeTruthy();

        const firstDot = page.getByRole("button", { name: "精选第 1 篇" });
        await expect(firstDot).toBeVisible();
        const featured = firstDot.locator("xpath=ancestor::section[1]");
        const lead = featured.locator('a[href^="/posts/"]').first();
        const image = lead.locator("img");
        await expect(image).toBeVisible();

        const measureFeatured = () =>
          page.evaluate(() => {
            const dot = document.querySelector<HTMLButtonElement>(
              'button[aria-label="精选第 1 篇"]',
            );
            const section = dot?.closest("section");
            const leadColumn = section?.firstElementChild;
            const card =
              section?.querySelector<HTMLAnchorElement>('a[href^="/posts/"]');
            const sectionBox = section?.getBoundingClientRect();
            const leadColumnBox = leadColumn?.getBoundingClientRect();
            const cardBox = card?.getBoundingClientRect();
            return {
              sectionHeight: sectionBox?.height ?? 0,
              sectionBottom: sectionBox?.bottom ?? 0,
              leadColumnHeight: leadColumnBox?.height ?? 0,
              cardHeight: cardBox?.height ?? 0,
              cardBottom: cardBox?.bottom ?? 0,
            };
          });

        const beforeLoad = await measureFeatured();
        releaseMedia();
        await image.evaluate((element: HTMLImageElement) => {
          if (element.complete && element.naturalWidth > 0) return;
          return new Promise<void>((resolve, reject) => {
            element.addEventListener("load", () => resolve(), { once: true });
            element.addEventListener(
              "error",
              () => reject(new Error("精选封面加载失败")),
              {
                once: true,
              },
            );
          });
        });
        const afterLoad = await measureFeatured();
        await page.screenshot({
          path: testInfo.outputPath("featured-cover-loaded.png"),
          fullPage: false,
        });

        for (const geometry of [beforeLoad, afterLoad]) {
          expect(geometry.sectionHeight).toBeLessThanOrEqual(452);
          expect(geometry.leadColumnHeight).toBeLessThanOrEqual(452);
          expect(geometry.cardHeight).toBeLessThanOrEqual(412);
          expect(geometry.cardBottom).toBeLessThanOrEqual(
            geometry.sectionBottom + 1,
          );
        }
        expect(
          Math.abs(afterLoad.cardHeight - beforeLoad.cardHeight),
        ).toBeLessThanOrEqual(1);
      });

      test("已发布文章在 LAN HTTP 下可以阅读", async ({ page }) => {
        const errors = captureErrors(page);
        await page.addInitScript(() => {
          Object.defineProperty(globalThis.crypto, "randomUUID", {
            configurable: true,
            value: undefined,
          });
        });
        await page.goto(new URL(contract.public.path, site.url).toString(), {
          waitUntil: "domcontentloaded",
        });
        const articleLink = page.locator('a[href^="/posts/"]').first();
        await expect(articleLink).toBeVisible();
        const articleTitle = (
          await articleLink.getByRole("heading").innerText()
        ).trim();
        const articlePath = await articleLink.getAttribute("href");
        expect(articlePath).toBeTruthy();

        await page.goto(new URL(articlePath!, site.url).toString(), {
          waitUntil: "domcontentloaded",
        });
        await page.waitForTimeout(1_500);
        expect(
          await page.getByText("crypto.randomUUID is not a function").count(),
        ).toBe(0);
        await expect(
          page.getByRole("heading", { name: articleTitle }).first(),
        ).toBeVisible();
        await settleNuxt(page);
        expect(errors).toEqual([]);
      });

      test("匿名访问管理入口进入账户登录流程", async ({ page }) => {
        const errors = captureErrors(page);
        await page.goto(new URL(contract.manage.path, site.url).toString(), {
          waitUntil: "domcontentloaded",
        });
        await expect(page).toHaveURL(
          new RegExp(
            `^${accountURL.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}/login(?:\\?|$)`,
          ),
        );
        await expect(
          page.getByRole("heading", { name: "欢迎回来" }),
        ).toBeVisible();
        await settleNuxt(page);
        expect(errors).toEqual([]);
      });

      test("账户登录后阅读公开文章不会丢失管理会话", async ({ page }) => {
        const errors = captureErrors(page);
        const manageURL = new URL(contract.manage.path, site.url).toString();
        await page.goto(manageURL, { waitUntil: "domcontentloaded" });
        await expect(
          page.getByRole("heading", { name: "欢迎回来" }),
        ).toBeVisible();
        await settleNuxt(page);
        await page.getByRole("textbox", { name: /邮箱/ }).fill(e2eEmail);
        await page.getByLabel(/密码/).fill(e2ePassword);
        await page.getByRole("button", { name: "登录", exact: true }).click();
        await page.waitForURL(manageURL);
        await expect(page.locator("[data-blog-dashboard-panel]")).toBeVisible();

        await page.goto(new URL(contract.public.path, site.url).toString(), {
          waitUntil: "domcontentloaded",
        });
        const articlePath = await page
          .locator('a[href^="/posts/"]')
          .first()
          .getAttribute("href");
        expect(articlePath).toBeTruthy();
        await page.goto(new URL(articlePath!, site.url).toString(), {
          waitUntil: "domcontentloaded",
        });
        await page.waitForTimeout(1_500);
        await expect(
          page.getByRole("link", { name: "登录", exact: true }),
        ).toHaveCount(0);

        await page.goto(manageURL, { waitUntil: "domcontentloaded" });
        await expect(page).toHaveURL(manageURL);
        await expect(page.locator("[data-blog-dashboard-panel]")).toBeVisible();
        expect(errors).toEqual([]);
      });

      test("中央会话可以静默重建产品会话", async ({ page }) => {
        const manageURL = new URL(contract.manage.path, site.url).toString();
        await page.goto(manageURL, { waitUntil: "domcontentloaded" });
        await expect(
          page.getByRole("heading", { name: "欢迎回来" }),
        ).toBeVisible();
        await settleNuxt(page);
        await page.getByRole("textbox", { name: /邮箱/ }).fill(e2eEmail);
        await page.getByLabel(/密码/).fill(e2ePassword);
        await page.getByRole("button", { name: "登录", exact: true }).click();
        await page.waitForURL(manageURL);
        await expect(page.locator("[data-blog-dashboard-panel]")).toBeVisible();

        await page.context().clearCookies({ name: "rs_session" });
        await page
          .goto(manageURL, { waitUntil: "domcontentloaded" })
          .catch((error) => {
            if (!String(error).includes("ERR_ABORTED")) throw error;
          });
        await expect(page).toHaveURL(manageURL, { timeout: 10_000 });
        await expect(page.locator("[data-blog-dashboard-panel]")).toBeVisible();
        await expect(
          page.getByRole("heading", { name: "欢迎回来" }),
        ).toHaveCount(0);
      });

      test("已登录运营者可以进入管理界面", async ({ browser }) => {
        const context = await loginE2E(browser);
        const page = await context.newPage();
        const errors = captureErrors(page);
        try {
          const manageURL = new URL(contract.manage.path, site.url).toString();
          const manageResponse = await page.goto(manageURL, {
            waitUntil: "domcontentloaded",
          });
          const manageHTML = await manageResponse?.text();
          expect(manageHTML).not.toMatch(/正在打开[^\n]{0,16}控制台/u);
          expect(manageHTML).toContain("data-admin-shell");
          await expect(page).toHaveURL(manageURL);
          await expect(
            page.locator(contract.manage.readySelector).first(),
          ).toBeVisible();
          await expect(
            page
              .getByRole("heading", {
                name: new RegExp(contract.manage.heading),
              })
              .first(),
          ).toBeVisible();
          await expect(
            page.locator('[data-admin-sidebar-appearance="commercial"]'),
          ).toBeVisible();
          await expect(
            page.locator('[data-admin-sidebar-brand] a[href="/"]'),
          ).toBeVisible();
          await expect(
            page.locator("[data-admin-sidebar-brand]").getByText("月离博客", {
              exact: true,
            }),
          ).toBeVisible();
          await expect(
            page.getByText("控制台在线", { exact: true }),
          ).toHaveCount(0);
          await expect(
            page.locator("[data-blog-dashboard-panel]"),
          ).toBeVisible();
          await expect(page.locator("[data-dashboard-metric]")).toHaveCount(4);
          await expect(page.locator("[data-dashboard-trend]")).toBeVisible();
          await expect(page.locator("[data-dashboard-audience]")).toBeVisible();
          await expect(
            page.locator("[data-dashboard-top-posts]"),
          ).toBeVisible();
          await expect(page.locator("[data-dashboard-sources]")).toBeVisible();
          const sevenDayResponse = page.waitForResponse(
            (response) =>
              response.url().includes("/api/v1/dashboard/overview?days=7") &&
              response.ok(),
          );
          await page.getByRole("tab", { name: "7 天" }).click();
          await expect(page.locator("[data-dashboard-metric]")).toHaveCount(4);
          await sevenDayResponse;
          await expect(
            page.locator(
              '[data-dashboard-trend] [data-dashboard-trend-chart] svg[aria-label*="最近 7 天"]',
            ),
          ).toBeVisible();
          await expect(
            page.getByRole("heading", { name: "最近工作" }),
          ).toHaveCount(0);
          await expect(
            page.getByRole("heading", { name: "快捷动作" }),
          ).toHaveCount(0);
          await expect(
            page.getByRole("heading", { name: "待处理" }),
          ).toHaveCount(0);
          await expect(
            page.getByRole("button", { name: /打开.+站点菜单/ }),
          ).toHaveCount(0);

          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "domcontentloaded",
          });
          const postsHeader = page.locator("[data-manage-posts-header]");
          await expect(postsHeader).toBeVisible();
          await expect(
            postsHeader.getByRole("heading", { name: "文章", exact: true }),
          ).toBeVisible();
          await expect(
            postsHeader.getByRole("button", { name: "写新文章", exact: true }),
          ).toHaveCount(1);
          await expect(
            page.getByRole("navigation", { name: "内容状态" }),
          ).toHaveCount(0);

          await page.getByRole("button", { name: "筛选", exact: true }).click();
          await page.getByLabel("文章状态").click();
          await page.getByRole("option", { name: /已发布/ }).click();
          await expect(page).toHaveURL(/(?:\?|&)status=published(?:&|$)/);
          await expect(
            page.getByText("状态：已发布", { exact: true }),
          ).toBeVisible();

          await settleNuxt(page);
          expect(errors).toEqual([]);
        } finally {
          await context.close();
        }
      });

      test("管理页面使用统一页头且不重复解释页面职责", async ({ browser }) => {
        const context = await loginE2E(browser);
        const page = await context.newPage();
        const errors = captureErrors(page);
        const pages = [
          { path: "/manage/posts", title: "文章" },
          { path: "/manage/comments", title: "评论" },
          { path: "/manage/categories", title: "分类" },
          { path: "/manage/tags", title: "标签" },
          { path: "/manage/settings", title: "站点设置" },
          { path: "/manage/assets", title: "资源策略" },
          { path: "/manage/authorization", title: "权限与申请" },
        ];
        const headingStyles: Array<{ path: string; style: string }> = [];

        try {
          for (const target of pages) {
            const errorCount = errors.length;
            await page.goto(new URL(target.path, site.url).toString(), {
              waitUntil: "domcontentloaded",
            });
            const header = page.locator("[data-manage-page-header]");
            const heading = header.getByRole("heading", {
              name: target.title,
              exact: true,
            });
            await expect(header).toHaveCount(1);
            await expect(heading).toBeVisible();
            await page.evaluate(async () => {
              await document.fonts.ready;
              await new Promise<void>((resolve) =>
                requestAnimationFrame(() =>
                  requestAnimationFrame(() => resolve()),
                ),
              );
            });
            await expect(header.locator("p")).toHaveCount(0);
            await expect(page.locator("h1:visible")).toHaveCount(1);
            if (target.path === "/manage/comments") {
              await expect(
                page.getByText("全站评论审核", { exact: false }),
              ).toHaveCount(0);
            }
            if (target.path === "/manage/assets") {
              await expect(
                page.locator("[data-asset-registration-summary]"),
              ).toBeVisible();
              await expect(
                page.getByRole("heading", { name: "有效用途", exact: true }),
              ).toBeVisible();
            }
            headingStyles.push({
              path: target.path,
              style: await heading.evaluate((element) => {
                const style = getComputedStyle(element);
                return [
                  style.fontSize,
                  style.fontWeight,
                  style.lineHeight,
                ].join("/");
              }),
            });
            expect(
              errors.slice(errorCount),
              `${target.path} 不应产生浏览器错误`,
            ).toEqual([]);
          }

          expect(headingStyles).toEqual(
            headingStyles.map((item) => ({
              path: item.path,
              style: headingStyles[0]!.style,
            })),
          );
        } finally {
          await context.close();
        }
      });

      test("设置入口使用产品语言并提供完整分区导航", async ({
        browser,
      }, testInfo) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        const errors = captureErrors(page);
        let originalConfig: Record<string, unknown> | null = null;
        try {
          await page.goto(new URL("/manage/settings", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const home = await context.request.get(
            new URL("/api/v1/home", site.url).toString(),
          );
          expect(home.ok()).toBeTruthy();
          originalConfig = (await home.json()).config;
          const settingsNavigation = page.getByRole("navigation", {
            name: "设置分区",
          });
          await expect(settingsNavigation).toBeVisible();
          await expect(
            page.getByRole("heading", { name: "站点信息", exact: true }),
          ).toBeVisible();
          await expect(
            page.getByRole("heading", { name: "文章封面", exact: true }),
          ).toBeVisible();
          await expect(
            page.locator('[data-settings-navigation-layout="sidebar"]'),
          ).toBeVisible();
          await expect(
            page.getByRole("button", { name: "站点", exact: true }),
          ).toBeVisible();
          await expect(
            page.getByRole("button", { name: "首页", exact: true }),
          ).toHaveCount(0);
          const footerNavigation = page.getByRole("button", {
            name: "页脚",
            exact: true,
          });
          await expect(
            footerNavigation.locator('[data-slot="leadingIcon"]'),
          ).toBeVisible();
          await footerNavigation.click();
          await expect(page).toHaveURL(/(?:\?|&)section=footer(?:&|$)/u);
          await expect(
            page.getByRole("heading", { name: "页脚内容", exact: true }),
          ).toBeVisible();
          await expect(
            page.getByRole("heading", { name: "联系", exact: true }),
          ).toBeVisible();
          await expect(
            page.getByRole("heading", { name: "友链", exact: true }),
          ).toBeVisible();
          await page.getByRole("button", { name: "站点", exact: true }).click();
          await expect(page).toHaveURL(/(?:\?|&)section=site(?:&|$)/u);
          await expect(
            page.getByRole("heading", { name: "站点信息", exact: true }),
          ).toBeVisible();
          const settingsGeometry = await page.evaluate(() => {
            const navigation = document.querySelector<HTMLElement>(
              '[data-settings-navigation-layout="sidebar"] nav',
            );
            const heading = Array.from(document.querySelectorAll("h2")).find(
              (element) => element.textContent?.trim() === "站点信息",
            );
            const section = heading?.closest("section");
            const navigationBox = navigation?.getBoundingClientRect();
            const sectionBox = section?.getBoundingClientRect();
            return {
              navigationX: navigationBox?.x ?? 0,
              sectionX: sectionBox?.x ?? 0,
              sectionBackground: section
                ? getComputedStyle(section).backgroundColor
                : "",
            };
          });
          expect(settingsGeometry.navigationX).toBeLessThan(
            settingsGeometry.sectionX,
          );
          expect(settingsGeometry.sectionBackground).toBe("rgb(255, 255, 255)");
          await expect(page.getByLabel("联系邮箱")).toHaveCount(0);
          await expect(
            page.getByRole("button", { name: /搜索控制台/ }),
          ).toBeVisible();
          await expect(
            page.getByText("YUELI · BLOG OS", { exact: true }),
          ).toHaveCount(0);
          await expect(
            page.getByText("搜索博客后台", { exact: false }),
          ).toHaveCount(0);
          await page.screenshot({
            path: testInfo.outputPath("site-settings-desktop.png"),
            fullPage: false,
          });

          await page.goto(new URL("/manage/assets", site.url).toString(), {
            waitUntil: "networkidle",
          });
          await expect(
            page.getByRole("heading", { name: "资源策略", exact: true }),
          ).toBeVisible();
          await expect(
            page.getByRole("heading", { name: "资源注册", exact: true }),
          ).toBeVisible();
          await expect(
            page.getByRole("heading", { name: "有效用途", exact: true }),
          ).toBeVisible();
          await expect(page.getByLabel("站点名称")).toHaveCount(0);
          await expect(
            page.getByText("blog-cover", { exact: true }),
          ).toBeVisible();
          await expect(
            page.getByText("blog-post", { exact: true }),
          ).toBeVisible();
          await expect(
            page.getByText("已接受", { exact: true }),
          ).toBeVisible();
          await expect(page.getByLabel("允许的格式")).toHaveCount(0);
          await page.screenshot({
            path: testInfo.outputPath("asset-policy-desktop.png"),
            fullPage: false,
          });
          await expect(page.getByRole("button", { name: "保存" })).toHaveCount(0);
          await expect(page.getByText("avif", { exact: true })).toHaveCount(0);

          await page.setViewportSize({ width: 390, height: 844 });
          await page.goto(
            new URL("/manage/settings?section=site", site.url).toString(),
            { waitUntil: "networkidle" },
          );
          await expect(
            page.getByRole("combobox", { name: "设置分区" }),
          ).toBeVisible();
          await page.screenshot({
            path: testInfo.outputPath("site-settings-mobile.png"),
            fullPage: false,
          });
          await page.setViewportSize({ width: 1440, height: 900 });
          const nextTitle = `月离博客验收 ${Date.now()}`;
          const nextDescription = `站点描述验收 ${Date.now()}`;
          await page.getByLabel("站点名称").fill(nextTitle);
          await page.getByLabel("站点描述").fill(nextDescription);
          const updated = page.waitForResponse(
            (response) =>
              response.request().method() === "PATCH" &&
              response.url().endsWith("/api/v1/home"),
          );
          await page.getByRole("button", { name: "保存", exact: true }).click();
          expect((await updated).ok()).toBeTruthy();
          await expect(
            page.getByText(nextTitle, { exact: true }).first(),
          ).toBeVisible();
          const publicPage = await context.newPage();
          await publicPage.goto(site.url, { waitUntil: "networkidle" });
          await expect(
            publicPage
              .locator("[data-public-footer]")
              .getByText(nextDescription, { exact: true }),
          ).toBeVisible();
          await publicPage.close();
          expect(errors).toEqual([]);
        } finally {
          if (originalConfig) {
            await context.request.patch(
              new URL("/api/v1/home", site.url).toString(),
              { data: originalConfig },
            );
          }
          await context.close();
        }
      });

      test("页脚联系与友链保存后公开展示并可恢复", async ({
        browser,
      }, testInfo) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        const errors = captureErrors(page);
        let originalConfig:
          | (Record<string, unknown> & {
              friendLinks?: unknown[];
              contactLinks?: unknown[];
              supportEmail?: string;
            })
          | null = null;
        const suffix = Date.now();
        const firstLabel = `友链甲 ${suffix}`;
        const secondLabel = `友链乙 ${suffix}`;
        try {
          const home = await context.request.get(
            new URL("/api/v1/home", site.url).toString(),
          );
          expect(home.ok()).toBeTruthy();
          originalConfig = (await home.json()).config;
          const originalCount = Array.isArray(originalConfig?.friendLinks)
            ? originalConfig.friendLinks.length
            : 0;
          const originalContactCount = Array.isArray(
            originalConfig?.contactLinks,
          )
            ? originalConfig.contactLinks.length
            : 0;
          const visibleContactCount =
            originalContactCount > 0
              ? originalContactCount
              : originalConfig?.supportEmail
                ? 1
                : 0;

          await page.goto(
            new URL("/manage/settings?section=footer", site.url).toString(),
            { waitUntil: "networkidle" },
          );
          const addContact = page.getByRole("button", {
            name: "添加联系方式",
            exact: true,
          });
          await addContact.click();
          await page
            .locator(`#contact-link-value-${visibleContactCount}`)
            .fill("QQ群 123456789");
          await page
            .locator(`#contact-link-url-${visibleContactCount}`)
            .fill(`https://qm.qq.com/example-${suffix}`);

          await addContact.click();
          await page
            .locator(`#contact-link-value-${visibleContactCount + 1}`)
            .fill("hello@example.com");
          await page
            .locator(`#contact-link-url-${visibleContactCount + 1}`)
            .fill("mailto:hello@example.com");

          const add = page.getByRole("button", {
            name: "添加友链",
            exact: true,
          });
          await add.click();
          await page
            .locator(`#friend-link-label-${originalCount}`)
            .fill(firstLabel);
          await page
            .locator(`#friend-link-url-${originalCount}`)
            .fill(`https://example.com/friend-a-${suffix}`);

          await add.click();
          await page
            .locator(`#friend-link-label-${originalCount + 1}`)
            .fill(secondLabel);
          await page
            .locator(`#friend-link-url-${originalCount + 1}`)
            .fill(`https://example.org/friend-b-${suffix}`);
          await page
            .getByRole("button", {
              name: `上移友链 ${originalCount + 2}`,
              exact: true,
            })
            .click();

          const updated = page.waitForResponse(
            (response) =>
              response.request().method() === "PATCH" &&
              response.url().endsWith("/api/v1/home"),
          );
          await page.getByRole("button", { name: "保存", exact: true }).click();
          expect((await updated).ok()).toBeTruthy();
          await expect(
            page.getByRole("button", { name: "已保存", exact: true }),
          ).toBeVisible();

          await page.reload({ waitUntil: "networkidle" });
          await expect(page.locator("[data-contact-link-row]")).toHaveCount(
            visibleContactCount + 2,
          );
          const savedRows = page.locator("[data-friend-link-row]");
          await expect(savedRows).toHaveCount(originalCount + 2);
          await expect(
            savedRows.nth(originalCount).locator('input[type="text"]').first(),
          ).toHaveValue(secondLabel);
          await expect(
            savedRows
              .nth(originalCount + 1)
              .locator('input[type="text"]')
              .first(),
          ).toHaveValue(firstLabel);
          await page.screenshot({
            path: testInfo.outputPath("footer-settings-desktop.png"),
            fullPage: false,
          });
          await page.setViewportSize({ width: 390, height: 844 });
          await page.reload({ waitUntil: "networkidle" });
          await page.screenshot({
            path: testInfo.outputPath("footer-settings-mobile.png"),
            fullPage: false,
          });
          await page.setViewportSize({ width: 1440, height: 900 });

          const publicPage = await context.newPage();
          const publicErrors = captureErrors(publicPage);
          await publicPage.goto(site.url, { waitUntil: "networkidle" });
          const footer = publicPage.locator("[data-public-footer]");
          await expect(footer).toBeVisible();
          await expect(
            footer.getByRole("navigation", { name: "页脚浏览" }),
          ).toBeVisible();
          await expect(
            footer.getByRole("navigation", { name: "友情链接" }),
          ).toBeVisible();
          await expect(
            footer.getByRole("heading", { name: "联系", exact: true }),
          ).toBeVisible();
          await expect(
            footer.getByText("QQ群 123456789", { exact: true }),
          ).toBeVisible();
          await expect(
            footer.getByRole("link", { name: /hello@example\.com/u }),
          ).toHaveAttribute("href", "mailto:hello@example.com");
          await expect(
            footer.getByRole("link", { name: new RegExp(secondLabel) }),
          ).toHaveAttribute("target", "_blank");
          await expect(
            footer.getByText("订阅更新", { exact: true }),
          ).toHaveCount(0);
          await expect(
            footer.locator('[class*="i-tabler-external-link"]'),
          ).toHaveCount(0);
          await footer.screenshot({
            path: testInfo.outputPath("public-footer-desktop.png"),
          });

          await publicPage.setViewportSize({ width: 390, height: 844 });
          await publicPage.reload({ waitUntil: "networkidle" });
          expect(
            await publicPage.evaluate(
              () => document.documentElement.scrollWidth - window.innerWidth,
            ),
          ).toBeLessThanOrEqual(1);
          await publicPage.locator("[data-public-footer]").screenshot({
            path: testInfo.outputPath("public-footer-mobile.png"),
          });
          expect(publicErrors).toEqual([]);
          await publicPage.close();

          const darkContext = await loginE2E(
            browser,
            { viewport: { width: 1440, height: 900 } },
            "dark",
          );
          try {
            const darkPage = await darkContext.newPage();
            await darkPage.goto(site.url, { waitUntil: "networkidle" });
            await darkPage.locator("[data-public-footer]").screenshot({
              path: testInfo.outputPath("public-footer-dark.png"),
            });
          } finally {
            await darkContext.close();
          }
          expect(errors).toEqual([]);
        } finally {
          if (originalConfig) {
            const restored = await context.request.patch(
              new URL("/api/v1/home", site.url).toString(),
              { data: originalConfig },
            );
            expect(restored.ok()).toBeTruthy();
          }
          await context.close();
        }
      });

      test("设置保存操作统一位于页面右上角", async ({ browser }, testInfo) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        try {
          await page.goto(
            new URL("/manage/settings?section=site", site.url).toString(),
            { waitUntil: "networkidle" },
          );
          const siteTitle = page.getByLabel("站点名称");
          await siteTitle.fill(`${await siteTitle.inputValue()} 临时修改`);
          const actions = page.locator("[data-settings-header-actions]");
          await expect(actions).toBeVisible();
          await expect(page.locator("[data-settings-save-dock]")).toHaveCount(
            0,
          );
          const geometry = await page.evaluate(() => {
            const header = document.querySelector<HTMLElement>(
              "[data-manage-page-header]",
            );
            const saveActions = document.querySelector<HTMLElement>(
              "[data-settings-header-actions]",
            );
            const headerBox = header?.getBoundingClientRect();
            const actionsBox = saveActions?.getBoundingClientRect();
            return {
              headerTop: headerBox?.top ?? 0,
              headerRight: headerBox?.right ?? 0,
              actionsTop: actionsBox?.top ?? 0,
              actionsRight: actionsBox?.right ?? 0,
            };
          });
          expect(geometry.actionsTop).toBeGreaterThanOrEqual(
            geometry.headerTop,
          );
          expect(geometry.actionsRight).toBeLessThanOrEqual(
            geometry.headerRight,
          );
          await page.screenshot({
            path: testInfo.outputPath("settings-header-save.png"),
            fullPage: false,
          });
          await page.getByRole("button", { name: "放弃", exact: true }).click();
        } finally {
          await context.close();
        }
      });

      test("管理内容在宽屏与中等视口都铺满可用区域", async ({
        browser,
      }, testInfo) => {
        const context = await loginE2E(browser, {
          viewport: { width: 2028, height: 900 },
        });
        const page = await context.newPage();
        try {
          for (const viewport of [
            { width: 2028, height: 900 },
            { width: 812, height: 844 },
          ]) {
            await page.setViewportSize(viewport);
            await page.goto(new URL("/manage/posts", site.url).toString(), {
              waitUntil: "networkidle",
            });
            const geometry = await page.evaluate(() => {
              const main = document.querySelector<HTMLElement>("#manage-main");
              const box = main?.getBoundingClientRect();
              return {
                viewportWidth: window.innerWidth,
                mainLeft: box?.left ?? 0,
                mainRight: box?.right ?? 0,
                overflow:
                  document.documentElement.scrollWidth - window.innerWidth,
              };
            });
            expect(
              geometry.viewportWidth - geometry.mainRight,
              `${viewport.width}px 管理内容右侧不应留下未使用区域`,
            ).toBeLessThanOrEqual(1);
            expect(geometry.overflow).toBeLessThanOrEqual(1);
            if (viewport.width < 1024) expect(geometry.mainLeft).toBe(0);
            await page.screenshot({
              path: testInfo.outputPath(`manage-posts-${viewport.width}.png`),
              fullPage: false,
            });
          }
        } finally {
          await context.close();
        }
      });

      test("站点设置与资源策略在深色和窄屏下保持清晰层级", async ({
        browser,
      }, testInfo) => {
        const context = await loginE2E(
          browser,
          { viewport: { width: 1440, height: 900 } },
          "dark",
        );
        const page = await context.newPage();
        const errors = captureErrors(page);
        try {
          for (const target of [
            { path: "/manage/settings?section=site", slug: "site" },
            { path: "/manage/assets", slug: "asset-policy" },
          ]) {
            await page.goto(new URL(target.path, site.url).toString(), {
              waitUntil: "networkidle",
            });
            const surface = page
              .locator("section")
              .filter({ has: page.locator("h2") })
              .first();
            await expect(surface).toBeVisible();
            const colors = await surface.evaluate((element) => ({
              body: getComputedStyle(document.body).backgroundColor,
              surface: getComputedStyle(element).backgroundColor,
            }));
            expect(colors.surface).not.toBe(colors.body);
            await page.screenshot({
              path: testInfo.outputPath(`${target.slug}-settings-dark.png`),
              fullPage: false,
            });
          }

          await page.setViewportSize({ width: 390, height: 844 });
          await page.goto(new URL("/manage/assets", site.url).toString(), {
            waitUntil: "networkidle",
          });
          await expect(
            page.getByRole("heading", { name: "资源策略", exact: true }),
          ).toBeVisible();
          expect(
            await page.evaluate(
              () => document.documentElement.scrollWidth - window.innerWidth,
            ),
          ).toBeLessThanOrEqual(1);
          await page.screenshot({
            path: testInfo.outputPath("asset-policy-mobile-dark.png"),
            fullPage: false,
          });
          expect(errors).toEqual([]);
        } finally {
          await context.close();
        }
      });

      test("消费者管理员可以修改并持久化自己的资源设置", async ({ browser }) => {
        const context = await loginE2E(browser);
        const page = await context.newPage();
        let originalSize = "";
        try {
          await page.goto(new URL("/manage/assets", site.url).toString(), { waitUntil: "networkidle" });
          await page.locator("[data-asset-registration-edit]").click();
          const cover = page.locator('[data-asset-profile-editor="blog-cover"]');
          const size = cover.getByLabel("大小上限（MB）");
          originalSize = await size.inputValue();
          const changedSize = originalSize === "19" ? "18" : "19";
          await size.fill(changedSize);
          const saved = page.waitForResponse((response) =>
            response.request().method() === "PUT" && /\/asset-api\/api\/v1\/assets\/registration$/.test(response.url()),
          );
          await page.locator("[data-asset-registration-save]").click();
          expect((await saved).ok()).toBeTruthy();
          await expect(page.getByText("已保存", { exact: true })).toBeVisible();

          await page.reload({ waitUntil: "networkidle" });
          await page.locator("[data-asset-registration-edit]").click();
          await expect(page.locator('[data-asset-profile-editor="blog-cover"]').getByLabel("大小上限（MB）")).toHaveValue(changedSize);
        } finally {
          if (originalSize) {
            await page.goto(new URL("/manage/assets", site.url).toString(), { waitUntil: "networkidle" }).catch(() => undefined);
            await page.locator("[data-asset-registration-edit]").click().catch(() => undefined);
            const size = page.locator('[data-asset-profile-editor="blog-cover"]').getByLabel("大小上限（MB）");
            if (await size.isVisible().catch(() => false)) {
              await size.fill(originalSize);
              await page.locator("[data-asset-registration-save]").click();
              await expect(page.getByText("已保存", { exact: true })).toBeVisible();
            }
          }
          await context.close();
        }
      });

      test("消费者只能选择代码声明的交付预设", async ({ browser }) => {
        const context = await loginE2E(browser);
        const page = await context.newPage();
        let changed = false;
        try {
          await page.goto(new URL("/manage/assets", site.url).toString(), {
            waitUntil: "networkidle",
          });
          await page.locator("[data-asset-registration-edit]").click();
          const cover = page.locator('[data-asset-profile-editor="blog-cover"]');
          await expect(cover.locator("[data-asset-add-variant]")).toHaveCount(0);
          await expect(cover.getByRole("button", { name: /删除规格/ })).toHaveCount(0);
          await expect(cover.getByRole("textbox", { name: "规格名称" })).toHaveCount(0);
          const preset = cover.getByRole("combobox", { name: "文章卡片预设" });
          await preset.click();
          await page.getByRole("option", { name: "省流 · 480×320" }).click();
          changed = true;
          const saved = page.waitForResponse((response) =>
            response.request().method() === "PUT" &&
            /\/asset-api\/api\/v1\/assets\/registration$/.test(response.url()),
          );
          await page.locator("[data-asset-registration-save]").click();
          expect((await saved).ok()).toBeTruthy();
          await expect(page.getByText("已保存", { exact: true })).toBeVisible();

          await page.reload({ waitUntil: "networkidle" });
          await page.locator("[data-asset-registration-edit]").click();
          await expect(
            page.locator('[data-asset-profile-editor="blog-cover"]')
              .getByRole("combobox", { name: "文章卡片预设" }),
          ).toContainText("省流 · 480×320");
        } finally {
          if (changed) {
            await page.goto(new URL("/manage/assets", site.url).toString(), {
              waitUntil: "networkidle",
            }).catch(() => undefined);
            await page.locator("[data-asset-registration-edit]").click().catch(() => undefined);
            const preset = page.locator('[data-asset-profile-editor="blog-cover"]')
              .getByRole("combobox", { name: "文章卡片预设" });
            await preset.click();
            await page.getByRole("option", { name: "标准 · 600×400" }).click();
            await page.locator("[data-asset-registration-save]").click();
            await expect(page.getByText("已保存", { exact: true })).toBeVisible();
          }
          await context.close();
        }
      });

      test("资源策略编辑在手机、平板和桌面宽度内完整重排", async ({ browser }, testInfo) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1280, height: 900 },
        });
        const page = await context.newPage();
        try {
          for (const width of [390, 768, 1024, 1280]) {
            await page.setViewportSize({ width, height: 900 });
            await page.goto(new URL("/manage/assets", site.url).toString(), {
              waitUntil: "networkidle",
            });
            await expect(page.locator("[data-asset-profile-summary]")).toHaveCount(2);
            await expect(page.locator("[data-asset-variant-summary]")).toHaveCount(6);
            const firstProfilePadding = await page
              .locator('[data-asset-profile-summary="blog-cover"]')
              .evaluate((element) => Number.parseFloat(getComputedStyle(element).paddingTop));
            expect(firstProfilePadding, `overview padding at ${width}px`).toBeGreaterThanOrEqual(16);
            if (width === 390 || width === 1280) {
              await page.screenshot({
                path: testInfo.outputPath(`asset-overview-${width}.png`),
                fullPage: true,
              });
            }
            await page.locator("[data-asset-registration-edit]").click();
            await expect(page.locator("[data-asset-registration-editor]")).toBeVisible();
            await expect(page.getByRole("tab", { name: /文章封面/ })).toBeVisible();
            await expect(page.getByRole("tab", { name: /文章正文图片/ })).toBeVisible();
            const tabOverflow = await page.getByRole("tablist").evaluate((element) => ({
              horizontal: element.scrollWidth - element.clientWidth,
              vertical: element.scrollHeight - element.clientHeight,
              overflowY: getComputedStyle(element).overflowY,
            }));
            expect(tabOverflow.vertical, `tablist vertical overflow at ${width}px`).toBeLessThanOrEqual(1);
            expect(tabOverflow.overflowY, `tablist overflow mode at ${width}px`).toBe("visible");
            await expect(page.locator('[data-asset-profile-editor="blog-cover"]')).toBeVisible();
            await expect(page.locator('[data-asset-profile-editor="blog-post"]')).toHaveCount(0);
            await expect(page.getByRole("textbox", { name: "规格名称" })).toHaveCount(0);
            await expect(page.locator("[data-asset-add-variant]")).toHaveCount(0);
            await expect(page.getByRole("button", { name: /删除规格/ })).toHaveCount(0);
            if (width === 1024) {
              const cover = page.locator('[data-asset-profile-editor="blog-cover"]');
              await expect(cover.locator("[data-asset-variant-editor]")).toHaveCount(4);
              await expect(cover.getByText("文章列表、搜索结果与归档", { exact: true })).toBeVisible();
              await expect(cover.getByRole("combobox", { name: "文章卡片预设" })).toBeVisible();
              await page.getByRole("tab", { name: /文章正文图片/ }).click();
              await expect(page.locator('[data-asset-profile-editor="blog-post"]')).toBeVisible();
              await expect(page.locator('[data-asset-profile-editor="blog-cover"]')).toHaveCount(0);
              await expect(page.locator('[data-asset-profile-editor="blog-post"] [data-asset-variant-editor]')).toHaveCount(2);
              await expect(page.getByText("点击正文图片后查看", { exact: true })).toBeVisible();
              await page.getByRole("tab", { name: /文章封面/ }).click();
            }
            const overflow = await page.evaluate(() => ({
              document: document.documentElement.scrollWidth - window.innerWidth,
              editor: (() => {
                const element = document.querySelector<HTMLElement>(
                  "[data-asset-registration-editor]",
                );
                return element ? element.scrollWidth - element.clientWidth : -1;
              })(),
            }));
            expect(overflow.document, `viewport ${width}px`).toBeLessThanOrEqual(1);
            expect(overflow.editor, `editor ${width}px`).toBeLessThanOrEqual(1);
            if (width === 390 || width === 1280) {
              await page.screenshot({
                path: testInfo.outputPath(`asset-variants-${width}.png`),
                fullPage: true,
              });
            }
          }
        } finally {
          await context.close();
        }
      });

      test("正文图片先加载缩略图并按需打开处理后大图", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1280, height: 900 },
        });
        const page = await context.newPage();
        const filename = `article-rendition-${Date.now()}.webp`;
        const alt = "正文图片预览";
        let postId = "";
        let assetId = "";
        try {
          await page.goto(new URL("/manage", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const source = Buffer.from(await page.evaluate(() => {
            const canvas = document.createElement("canvas");
            canvas.width = 160;
            canvas.height = 90;
            const context = canvas.getContext("2d");
            if (!context) throw new Error("Canvas unavailable");
            context.fillStyle = "#2563eb";
            context.fillRect(0, 0, canvas.width, canvas.height);
            return canvas.toDataURL("image/webp", 0.88).split(",")[1] || "";
          }), "base64");
          const initialized = await context.request.post(
            new URL("/api/v1/images", site.url).toString(),
            { data: { filename, mime: "image/webp", size: source.length } },
          );
          expect(initialized.ok(), await initialized.text()).toBeTruthy();
          const init = await initialized.json();
          const upload = await context.request.put(
            new URL(init.uploadUrl, site.url).toString(),
            { data: source, headers: init.uploadHeaders || {} },
          );
          expect(upload.ok()).toBeTruthy();
          const finalized = await context.request.post(
            new URL("/api/v1/images/finalize", site.url).toString(),
            { data: { uploadToken: init.uploadToken } },
          );
          expect(finalized.ok()).toBeTruthy();
          const thumbnailURL = (await finalized.json()).url as string;
          expect(thumbnailURL).toMatch(/name=inline/);

          const created = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `正文图片交付验收 ${Date.now()}`,
                content: `![${alt}](${thumbnailURL})`,
              },
            },
          );
          expect(created.ok()).toBeTruthy();
          const body = await created.json();
          postId = body.post.id;
          const publish = await context.request.patch(
            new URL(`/api/v1/posts/${postId}`, site.url).toString(),
            { data: { status: "published" } },
          );
          expect(publish.ok()).toBeTruthy();

          await page.goto(
            new URL(`/posts/${body.post.slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );
          const inlineImage = page.locator(`.content-prose img[alt="${alt}"]`);
          await expect(inlineImage).toBeVisible();
          expect(await inlineImage.getAttribute("src")).toContain("name=inline");
          await expect(inlineImage).toHaveAttribute("role", "button");
          const largeResponse = page.waitForResponse((response) => {
            const url = new URL(response.url());
            return url.pathname.includes("/media/") && url.searchParams.get("name") === "content";
          });
          await inlineImage.click();
          expect((await largeResponse).status()).toBeLessThan(400);
          const preview = page.getByRole("dialog", { name: alt });
          await expect(preview).toBeVisible();
          const largeImage = preview.locator("img");
          await expect(largeImage).toBeVisible();
          expect(await largeImage.getAttribute("src")).toContain("name=content");

          const assets = await context.request.get(
            new URL(
              "/asset-api/api/v1/assets?siteKey=blog-main&profileKey=blog-post&page=1&size=100",
              site.url,
            ).toString(),
          );
          expect(assets.ok()).toBeTruthy();
          assetId = ((await assets.json()).items || []).find(
            (asset: { filename?: string }) => asset.filename === filename,
          )?.id || "";
          expect(assetId).not.toBe("");
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          if (assetId) {
            await context.request.delete(
              new URL(`/asset-api/api/v1/assets/${assetId}`, site.url).toString(),
            );
          }
          await context.close();
        }
      });

      test("管理主题使用博客蓝并支持明暗模式", async ({ browser }) => {
        for (const colorMode of ["light", "dark"] as const) {
          const context = await loginE2E(
            browser,
            { viewport: { width: 1280, height: 600 } },
            colorMode,
          );
          const page = await context.newPage();
          const errors = captureErrors(page);
          try {
            await page.goto(new URL("/manage/comments", site.url).toString(), {
              waitUntil: "domcontentloaded",
            });
            await expect(
              page.getByRole("heading", { name: "评论", exact: true }),
            ).toBeVisible();
            const activeIcon = page.locator(
              '[data-admin-sidebar-primary] [data-slot="link"][data-active] [data-slot="linkLeadingIcon"]',
            );
            await expect(activeIcon).toBeVisible();
            const theme = await page.evaluate(() => {
              const bodyStyle = getComputedStyle(document.body);
              const activeIcon = document.querySelector(
                '[data-admin-sidebar-primary] [data-slot="link"][data-active] [data-slot="linkLeadingIcon"]',
              );
              const inactiveIcon = document.querySelector(
                '[data-admin-sidebar-primary] [data-slot="link"]:not([data-active]) [data-slot="linkLeadingIcon"]',
              );
              const activeIconStyle = activeIcon
                ? getComputedStyle(activeIcon)
                : null;
              const inactiveIconStyle = inactiveIcon
                ? getComputedStyle(inactiveIcon)
                : null;
              const sidebar = document.querySelector(
                '[data-admin-sidebar-appearance="commercial"]',
              );
              const sidebarBody = sidebar?.querySelector('[data-slot="body"]');
              const sidebarFooter = sidebar?.querySelector(
                "[data-admin-sidebar-account]",
              );
              const footerRect = sidebarFooter?.getBoundingClientRect();
              return {
                dark: document.documentElement.classList.contains("dark"),
                body: bodyStyle.backgroundColor,
                activeIcon: activeIconStyle?.color,
                activeIconPaint: activeIconStyle?.backgroundColor,
                activeIconOpacity: activeIconStyle?.opacity,
                inactiveIcon: inactiveIconStyle?.color,
                inactiveIconPaint: inactiveIconStyle?.backgroundColor,
                inactiveIconOpacity: inactiveIconStyle?.opacity,
                sidebarHeight: sidebar?.getBoundingClientRect().height,
                viewportHeight: window.innerHeight,
                navigationCount: sidebar?.querySelectorAll(
                  '[data-admin-sidebar-primary] [data-slot="link"]',
                ).length,
                sidebarOverflowY: sidebarBody
                  ? getComputedStyle(sidebarBody).overflowY
                  : "",
                footerVisible: Boolean(
                  footerRect &&
                  footerRect.top >= 0 &&
                  footerRect.bottom <= window.innerHeight,
                ),
              };
            });
            expect({
              dark: theme.dark,
              body: theme.body,
              activeIcon: theme.activeIcon,
              activeIconPaint: theme.activeIconPaint,
              activeIconOpacity: theme.activeIconOpacity,
              inactiveIcon: theme.inactiveIcon,
              inactiveIconPaint: theme.inactiveIconPaint,
              inactiveIconOpacity: theme.inactiveIconOpacity,
            }).toEqual(
              colorMode === "dark"
                ? {
                    dark: true,
                    body: "rgb(11, 15, 23)",
                    activeIcon: "rgb(147, 197, 253)",
                    activeIconPaint: "rgb(147, 197, 253)",
                    activeIconOpacity: "1",
                    inactiveIcon: "rgb(168, 183, 202)",
                    inactiveIconPaint: "rgb(168, 183, 202)",
                    inactiveIconOpacity: "1",
                  }
                : {
                    dark: false,
                    body: "rgb(248, 250, 252)",
                    activeIcon: "rgb(37, 99, 235)",
                    activeIconPaint: "rgb(37, 99, 235)",
                    activeIconOpacity: "1",
                    inactiveIcon: "rgb(71, 85, 105)",
                    inactiveIconPaint: "rgb(71, 85, 105)",
                    inactiveIconOpacity: "1",
                  },
            );
            expect(theme.sidebarHeight).toBe(theme.viewportHeight);
            expect(theme.navigationCount).toBe(8);
            expect(theme.sidebarOverflowY).toBe("auto");
            expect(theme.footerVisible).toBe(true);
            await expect(page.getByText(/控制台在线|博客服务在线/)).toHaveCount(
              0,
            );
            expect(errors).toEqual([]);
          } finally {
            await context.close();
          }
        }
      });

      test("文章列表快捷入口与编辑工作台可用", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        const errors = captureErrors(page);
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "domcontentloaded",
          });
          const publicLink = page
            .getByRole("link", { name: /查看前台文章/ })
            .first();
          await expect(publicLink).toBeVisible();
          await expect(publicLink).toHaveAttribute("href", /^\/posts\//);
          await expect(publicLink).toHaveAttribute("target", "_blank");
          const publicLabel = await publicLink.getAttribute("aria-label");
          const publishedTitle =
            publicLabel?.replace(/^查看前台文章：/, "") || "";
          expect(publishedTitle).not.toBe("");
          const editLink = page.getByRole("link", {
            name: `编辑文章：${publishedTitle}`,
          });
          const editHref = await editLink.getAttribute("href");
          expect(editHref).toMatch(/^\/manage\/posts\//);

          await page.goto(new URL(editHref!, site.url).toString(), {
            waitUntil: "domcontentloaded",
          });
          await expect(page.locator("[data-blog-post-editor]")).toBeVisible();
          await expect(page.getByText("未保存", { exact: true })).toHaveCount(
            0,
          );
          await expect(
            page.locator("[data-blog-editor-commandbar]"),
          ).toBeVisible();
          await expect(
            page.locator("[data-blog-editor-document]"),
          ).toBeVisible();
          await expect(page.getByLabel("文章标题")).toBeVisible();
          await page.getByRole("button", { name: "标题", exact: true }).click();
          await expect(page.getByText("标题 2", { exact: true })).toBeVisible();
          await expect(page.getByText("标题 1", { exact: true })).toHaveCount(
            0,
          );
          await page.keyboard.press("Escape");
          const documentBox = await page
            .locator("[data-blog-editor-document]")
            .boundingBox();
          const articleBox = await page
            .locator("[data-blog-editor-document] .ProseMirror")
            .boundingBox();
          expect(documentBox?.width || 0).toBeGreaterThan(1_050);
          expect(articleBox?.width || 0).toBeGreaterThan(950);
          const editorPresentation = await page.evaluate(() => {
            const documentElement = document.querySelector(
              "[data-blog-editor-document]",
            );
            const title = document.querySelector(".blog-editor-title");
            return {
              background: documentElement
                ? getComputedStyle(documentElement).backgroundColor
                : "",
              titleFontSize: title
                ? Number.parseFloat(getComputedStyle(title).fontSize)
                : 0,
            };
          });
          expect(editorPresentation.background).not.toBe("rgba(0, 0, 0, 0)");
          expect(editorPresentation.titleFontSize).toBeLessThanOrEqual(36);
          await expect(
            page.getByRole("button", { name: "预览文章" }),
          ).toBeVisible();
          await page.getByRole("button", { name: "文章设置" }).click();
          await expect(
            page.getByRole("heading", { name: "文章设置", exact: true }),
          ).toBeVisible();
          await expect(
            page.locator("[data-blog-settings-accordion]"),
          ).toBeVisible();
          await expect(
            page.getByRole("button", { name: /封面与摘要/ }),
          ).toBeVisible();
          await expect(
            page.getByRole("button", { name: /分类与系列/ }),
          ).toBeVisible();
          await expect(
            page.getByRole("button", { name: /发布设置/ }),
          ).toBeVisible();
          await expect(
            page.getByRole("button", { name: /搜索优化/ }),
          ).toBeVisible();
          const inspectorPresentation = await page.evaluate(() => {
            const surface = document.querySelector(
              ".blog-editor-settings-surface",
            );
            const accordion = document.querySelector(
              "[data-blog-settings-accordion]",
            );
            const section = document.querySelector(
              ".blog-editor-settings-section",
            );
            const styles = surface ? getComputedStyle(surface) : null;
            return {
              background: styles?.getPropertyValue("--ui-bg").trim() || "",
              muted: styles?.getPropertyValue("--ui-bg-muted").trim() || "",
              sectionShadow: section ? getComputedStyle(section).boxShadow : "",
              radius: accordion
                ? Number.parseFloat(getComputedStyle(accordion).borderRadius)
                : 0,
              decorativeIcons: document.querySelectorAll(
                "[data-blog-settings-accordion] [data-slot='trigger'] > [data-slot='leadingIcon']",
              ).length,
            };
          });
          expect(inspectorPresentation.background).toBe("#ffffff");
          expect(inspectorPresentation.muted).toBe("#f6f8fa");
          expect(inspectorPresentation.sectionShadow).toBe("none");
          expect(inspectorPresentation.radius).toBeLessThanOrEqual(10);
          expect(inspectorPresentation.decorativeIcons).toBe(0);
          const coverBox = await page
            .locator("[data-blog-cover-preview]")
            .boundingBox();
          expect(coverBox?.height || 0).toBeLessThanOrEqual(90);
          expect((coverBox?.width || 0) / (coverBox?.height || 1)).toBeCloseTo(
            3 / 2,
            1,
          );
          await page.getByRole("button", { name: /发布设置/ }).click();
          await expect(
            page.getByText("当前状态", { exact: true }),
          ).toBeVisible();
          await page.getByRole("button", { name: /搜索优化/ }).click();
          await expect(page.getByLabel("Meta 标题")).toBeVisible();
          await page
            .getByRole("button", { name: "保存", exact: true })
            .last()
            .click();
          await expect(
            page.getByRole("button", { name: "已保存", exact: true }).last(),
          ).toBeVisible();

          const overflow = await page.evaluate(() => ({
            viewport: document.documentElement.clientWidth,
            document: document.documentElement.scrollWidth,
          }));
          expect(overflow.document).toBeLessThanOrEqual(overflow.viewport);

          const layers = await page.evaluate(async () => {
            const sidebar = document.querySelector(
              '[data-admin-sidebar-appearance="commercial"]',
            );
            const account = sidebar?.querySelector(
              "[data-admin-sidebar-account]",
            );
            const main = document.querySelector("#manage-main");
            const commandbar = document.querySelector(
              "[data-blog-editor-commandbar]",
            );
            if (!sidebar || !account || !main || !commandbar)
              throw new Error("管理层级缺失");
            main.scrollTop = main.scrollHeight;
            await new Promise<void>((resolve) =>
              requestAnimationFrame(() =>
                requestAnimationFrame(() => resolve()),
              ),
            );
            const sidebarRect = sidebar.getBoundingClientRect();
            const accountRect = account.getBoundingClientRect();
            const mainRect = main.getBoundingClientRect();
            const commandbarRect = commandbar.getBoundingClientRect();
            return {
              viewportHeight: window.innerHeight,
              windowScrollY: window.scrollY,
              sidebarTop: sidebarRect.top,
              sidebarBottom: sidebarRect.bottom,
              accountBottom: accountRect.bottom,
              mainTop: mainRect.top,
              mainBottom: mainRect.bottom,
              commandbarTop: commandbarRect.top,
              mainScrollTop: main.scrollTop,
              mainClientHeight: main.clientHeight,
              mainScrollHeight: main.scrollHeight,
            };
          });
          expect(layers.windowScrollY).toBe(0);
          expect(layers.sidebarTop).toBe(0);
          expect(layers.sidebarBottom).toBe(layers.viewportHeight);
          expect(layers.accountBottom).toBeLessThanOrEqual(
            layers.viewportHeight,
          );
          expect(layers.mainTop).toBe(0);
          expect(layers.commandbarTop).toBe(0);
          expect(layers.mainBottom).toBe(layers.viewportHeight);
          expect(layers.mainScrollTop).toBeGreaterThan(0);
          expect(layers.mainScrollHeight).toBeGreaterThan(
            layers.mainClientHeight,
          );
          expect(errors).toEqual([]);
        } finally {
          await context.close();
        }
      });

      test("草稿预览和发布都会先保存当前编辑", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let postId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `保存后预览验收 ${Date.now()}`,
                content: "<p>服务端旧正文</p>",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          const slug = created.post.slug;
          await page.goto(
            new URL(`/manage/posts/${slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );

          const title = page.getByLabel("文章标题");
          const editable = page.locator('.tiptap[contenteditable="true"]');
          await title.fill("预览中的最新标题");
          await editable.click();
          await page.keyboard.press("Control+A");
          await page.keyboard.type("预览中的最新正文");

          const previewOpened = page.waitForEvent("popup");
          await page.getByRole("button", { name: "预览文章" }).click();
          const preview = await previewOpened;
          await preview.waitForLoadState("networkidle");
          await expect(preview).toHaveURL(
            new URL(`/posts/${encodeURIComponent(slug)}`, site.url).toString(),
          );
          await expect(
            preview.getByRole("heading", {
              level: 1,
              name: "预览中的最新标题",
            }),
          ).toBeVisible();
          await expect(preview.getByText("预览中的最新正文")).toBeVisible();
          await expect(
            preview.getByText("预览草稿", { exact: true }),
          ).toBeVisible();
          await preview.close();

          await title.fill("发布前最新标题");
          await editable.click();
          await page.keyboard.press("Control+A");
          await page.keyboard.type("发布前最新正文");
          const patches: Record<string, unknown>[] = [];
          page.on("request", (request) => {
            if (
              request.method() === "PATCH" &&
              request.url().endsWith(`/api/v1/posts/${postId}`)
            ) {
              patches.push(request.postDataJSON());
            }
          });
          const published = page.waitForResponse(
            (response) =>
              response.request().method() === "PATCH" &&
              response.url().endsWith(`/api/v1/posts/${postId}`) &&
              response.request().postDataJSON()?.status === "published",
          );
          await page.getByRole("button", { name: "发布", exact: true }).click();
          expect((await published).ok()).toBeTruthy();
          expect(patches[0]).toMatchObject({
            title: "发布前最新标题",
          });
          expect(patches.at(-1)).toMatchObject({ status: "published" });

          const detail = await context.request.get(
            new URL(`/api/v1/posts/${slug}`, site.url).toString(),
          );
          expect(detail.ok()).toBeTruthy();
          const body = await detail.json();
          expect(body.post.title).toBe("发布前最新标题");
          expect(body.post.content).toContain("发布前最新正文");
          expect(body.post.status).toBe("published");
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          await context.close();
        }
      });

      test("分类与标签使用搜索下拉且只常驻已选项", async ({
        browser,
      }, testInfo) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const editHref = await page
            .locator('a[aria-label^="编辑文章："]')
            .first()
            .getAttribute("href");
          expect(editHref).toBeTruthy();
          const taxonomyResponse = await context.request.get(
            new URL("/api/v1/taxonomies", site.url).toString(),
          );
          expect(taxonomyResponse.ok()).toBeTruthy();
          const taxonomyBody = await taxonomyResponse.json();
          const fakeCategories = Array.from({ length: 48 }, (_, index) => ({
            id: `e2e-category-${index + 1}`,
            taxonomy: "category",
            name: `虚拟分类 ${String(index + 1).padStart(2, "0")}`,
            slug: `virtual-category-${index + 1}`,
            description: "",
            parentId: index > 31 ? "e2e-category-1" : "",
            postCount: index,
          }));
          const fakeTags = Array.from({ length: 36 }, (_, index) => ({
            id: `e2e-tag-${index + 1}`,
            taxonomy: "tag",
            name: `虚拟标签 ${String(index + 1).padStart(2, "0")}`,
            slug: `virtual-tag-${index + 1}`,
            description: "",
            parentId: "",
            postCount: index,
          }));
          await page.route("**/api/v1/taxonomies", async (route) => {
            await route.fulfill({
              status: 200,
              contentType: "application/json",
              body: JSON.stringify({
                ...taxonomyBody,
                items: [
                  ...(taxonomyBody.items || []),
                  ...fakeCategories,
                  ...fakeTags,
                ],
              }),
            });
          });

          await page.goto(new URL(editHref!, site.url).toString(), {
            waitUntil: "networkidle",
          });
          await page.getByRole("button", { name: "文章设置" }).click();
          await page.getByRole("button", { name: /分类与系列/ }).click();
          await expect(
            page.getByText("虚拟分类 48", { exact: true }),
          ).toHaveCount(0);
          await expect(
            page.getByText("#虚拟标签 36", { exact: true }),
          ).toHaveCount(0);

          const categorySelector = page.getByRole("button", {
            name: "选择文章分类",
          });
          await categorySelector.click();
          await page
            .getByPlaceholder("搜索分类名称或路径…")
            .fill("虚拟分类 48");
          await page.getByText("虚拟分类 48", { exact: true }).click();
          await categorySelector.click();
          await expect(
            page.getByRole("button", { name: "移除分类：虚拟分类 48" }),
          ).toBeVisible();

          const tagSelector = page.getByRole("button", {
            name: "选择文章标签",
          });
          await tagSelector.click();
          await page
            .getByPlaceholder("搜索标签名称或 slug…")
            .fill("virtual-tag-36");
          await page.getByText("#虚拟标签 36", { exact: true }).click();
          await tagSelector.click();
          await expect(
            page.getByRole("button", { name: "移除标签：虚拟标签 36" }),
          ).toBeVisible();
          await expect(
            page.getByText("虚拟分类 01", { exact: true }),
          ).toHaveCount(0);
          await expect(
            page.getByText("#虚拟标签 01", { exact: true }),
          ).toHaveCount(0);
          await page.screenshot({
            path: testInfo.outputPath("taxonomy-selectors-desktop.png"),
            fullPage: false,
          });
          await page.setViewportSize({ width: 390, height: 844 });
          await page.screenshot({
            path: testInfo.outputPath("taxonomy-selectors-mobile.png"),
            fullPage: false,
          });
        } finally {
          await context.close();
        }
      });

      test("文章级本地草稿恢复标题摘要和正文", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let postId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `本地草稿验收 ${Date.now()}`,
                content: "<p>服务端正文</p>",
                excerpt: "服务端摘要",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          await page.goto(
            new URL(`/manage/posts/${created.post.slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );

          await page.getByLabel("文章标题").fill("本地恢复标题");
          const editable = page.locator('.tiptap[contenteditable="true"]');
          await editable.click();
          await page.keyboard.press("Control+A");
          await page.keyboard.type("本地恢复正文");
          await page.getByRole("button", { name: "文章设置" }).click();
          await page
            .getByRole("textbox", { name: "摘要" })
            .fill("本地恢复摘要");
          await page.getByRole("button", { name: "关闭", exact: true }).click();
          await page.evaluate(() =>
            window.dispatchEvent(new Event("pagehide")),
          );

          await page.reload({ waitUntil: "networkidle" });
          await expect(
            page.getByText("发现未保存的本地草稿", { exact: true }),
          ).toBeVisible();
          await page.getByRole("button", { name: "恢复草稿" }).click();
          await expect(page.getByLabel("文章标题")).toHaveValue("本地恢复标题");
          await expect(editable).toContainText("本地恢复正文");
          await page.getByRole("button", { name: "文章设置" }).click();
          await expect(page.getByRole("textbox", { name: "摘要" })).toHaveValue(
            "本地恢复摘要",
          );
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          await context.close();
        }
      });

      test("保存新文章地址后同步编辑路由", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let postId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `文章地址验收 ${Date.now()}`,
                content: "<p>正文</p>",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          await page.goto(
            new URL(`/manage/posts/${created.post.slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );

          const nextSlug = `editor-route-${Date.now()}`;
          await page.getByLabel("文章永久链接").fill(nextSlug);
          await page
            .getByRole("button", { name: "保存", exact: true })
            .last()
            .click();
          await expect(page).toHaveURL(
            new URL(`/manage/posts/${nextSlug}`, site.url).toString(),
          );
          await page.reload({ waitUntil: "networkidle" });
          await expect(page.getByLabel("文章永久链接")).toHaveValue(nextSlug);
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          await context.close();
        }
      });

      test("编辑器删除直接进入回收站并可恢复", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        const errors = captureErrors(page);
        let postId = "";
        let slug = "";
        let nativeDialogs = 0;
        page.on("dialog", async (dialog) => {
          nativeDialogs += 1;
          await dialog.dismiss();
        });
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `回收站验收 ${Date.now()}`,
                content: "<p>temporary</p>",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          slug = created.post.slug;
          const publish = await context.request.patch(
            new URL(`/api/v1/posts/${postId}`, site.url).toString(),
            { data: { status: "published" } },
          );
          expect(publish.ok()).toBeTruthy();

          await page.goto(
            new URL(`/manage/posts/${slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );
          await expect(
            page.getByText("文章编辑", { exact: true }),
          ).toBeVisible();
          await expect(
            page.getByRole("button", { name: "更多文章操作" }),
          ).toHaveCount(0);
          await page.getByRole("button", { name: "文章设置" }).click();
          const settings = page.getByRole("dialog", { name: "文章设置" });
          const trashButton = settings.getByRole("button", {
            name: "移入回收站",
          });
          const settingsSave = settings.getByRole("button", {
            name: "保存",
            exact: true,
          });
          const trashBox = await trashButton.boundingBox();
          const saveBox = await settingsSave.boundingBox();
          expect(trashBox?.x || 0).toBeLessThan(saveBox?.x || 0);

          const trashed = page.waitForResponse(
            (response) =>
              response.request().method() === "DELETE" &&
              response.url().endsWith(`/api/v1/posts/${postId}`),
          );
          await trashButton.click();
          expect((await trashed).ok()).toBeTruthy();
          await expect(page).toHaveURL(/\/manage\/posts\?status=trash$/);
          expect(nativeDialogs).toBe(0);
          await expect(
            page.getByText(created.post.title, { exact: true }),
          ).toBeVisible();

          const hidden = await context.request.get(
            new URL(`/api/v1/posts/${slug}`, site.url).toString(),
          );
          expect(hidden.status()).toBe(404);

          const restored = page.waitForResponse(
            (response) =>
              response.request().method() === "POST" &&
              response.url().endsWith(`/api/v1/posts/${postId}/restore`),
          );
          await page
            .getByRole("button", { name: `恢复文章：${created.post.title}` })
            .click();
          expect((await restored).ok()).toBeTruthy();
          await expect(
            page.getByText(created.post.title, { exact: true }),
          ).toHaveCount(0);
          const visible = await context.request.get(
            new URL(`/api/v1/posts/${slug}`, site.url).toString(),
          );
          expect(visible.ok()).toBeTruthy();
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          await context.close();
        }
      });

      test("文章发布日期与搜索信息可以维护", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let postId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const title = `文章元数据验收 ${Date.now()}`;
          const excerpt = "这是用于搜索结果和文章卡片的摘要。";
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title,
                content: "<h2>正文标题</h2><p>正文内容。</p>",
                excerpt,
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          const slug = created.post.slug;
          const publish = await context.request.patch(
            new URL(`/api/v1/posts/${postId}`, site.url).toString(),
            { data: { status: "published" } },
          );
          expect(publish.ok()).toBeTruthy();

          await page.goto(
            new URL(`/manage/posts/${slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );
          await page.getByRole("button", { name: "文章设置" }).click();
          const settings = page.getByRole("dialog", { name: "文章设置" });
          await settings.getByRole("button", { name: /发布设置/ }).click();
          await expect(
            settings.getByText("展示浏览量", { exact: true }),
          ).toHaveCount(0);
          await expect(
            settings.getByText("修改后会影响归档顺序和文章显示日期。", {
              exact: true,
            }),
          ).toHaveCount(0);
          const localPublishedAt = "2026-07-01T08:30";
          await settings.getByLabel("发布日期").fill(localPublishedAt);

          await settings.getByRole("button", { name: /搜索优化/ }).click();
          await settings.getByRole("button", { name: "从文章填充" }).click();
          await expect(settings.getByLabel("Meta 标题")).toHaveValue(title);
          await expect(settings.getByLabel("OG 标题")).toHaveValue(title);
          await expect(settings.getByLabel("Meta 描述")).toHaveValue(excerpt);

          const metadataResponses = Promise.all([
            page.waitForResponse(
              (response) =>
                response.request().method() === "PATCH" &&
                response.url().endsWith(`/api/v1/posts/${postId}`),
            ),
            page.waitForResponse(
              (response) =>
                response.request().method() === "PUT" &&
                response.url().endsWith(`/api/v1/posts/${postId}/seo`),
            ),
          ]);
          await settings
            .getByRole("button", { name: "保存", exact: true })
            .click();
          for (const response of await metadataResponses)
            expect(response.ok()).toBeTruthy();

          const detail = await context.request.get(
            new URL(`/api/v1/posts/${slug}`, site.url).toString(),
          );
          expect(detail.ok()).toBeTruthy();
          const detailBody = await detail.json();
          expect(new Date(detailBody.post.publishedAt).toISOString()).toBe(
            new Date(localPublishedAt).toISOString(),
          );
          expect(detailBody.seo.metaTitle).toBe(title);
          expect(detailBody.seo.metaDesc).toBe(excerpt);
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          await context.close();
        }
      });

      test("保存结果原地反馈且错误 Toast 保持低干扰", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let postId = "";
        let blockerId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const blocker = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `保存冲突占位 ${Date.now()}`,
                content: "<p>blocker</p>",
              },
            },
          );
          expect(blocker.ok()).toBeTruthy();
          const blocked = await blocker.json();
          blockerId = blocked.post.id;
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `保存反馈验收 ${Date.now()}`,
                content: "<p>temporary</p>",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          const originalSlug = created.post.slug;
          await page.goto(
            new URL(`/manage/posts/${created.post.slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );

          const postRoute = `**/api/v1/posts/${postId}`;
          await page.route(postRoute, async (route) => {
            if (route.request().method() !== "PATCH") {
              await route.continue();
              return;
            }
            await new Promise((resolve) => setTimeout(resolve, 250));
            await route.continue();
          });

          await page.getByLabel("文章永久链接").fill(blocked.post.slug);
          await page
            .getByRole("button", { name: "保存", exact: true })
            .last()
            .click();
          await expect(
            page.getByRole("button", { name: "保存中", exact: true }).last(),
          ).toBeVisible();
          const errorToast = page
            .locator("[data-yueli-toast]")
            .filter({ hasText: "保存失败" })
            .last();
          await expect(errorToast).toBeVisible({ timeout: 8_000 });
          await expect(
            errorToast.locator("[data-yueli-toast-progress]"),
          ).toHaveCount(0);
          await expect(
            errorToast.getByRole("button", { name: "关闭通知" }),
          ).toBeVisible();
          const presentation = await errorToast.evaluate((element) => {
            const style = getComputedStyle(element);
            const box = element.getBoundingClientRect();
            return {
              width: box.width,
              bottom: window.innerHeight - box.bottom,
              right: window.innerWidth - box.right,
              borderWidth: style.borderTopWidth,
              radius: Number.parseFloat(style.borderRadius),
              shadow: style.boxShadow,
              background: style.backgroundColor,
            };
          });
          expect(presentation.width).toBeLessThanOrEqual(352);
          expect(presentation.bottom).toBeLessThanOrEqual(32);
          expect(presentation.right).toBeLessThanOrEqual(32);
          expect(presentation.borderWidth).toBe("1px");
          expect(presentation.radius).toBeGreaterThanOrEqual(12);
          expect(presentation.shadow).not.toBe("none");
          expect(presentation.background).not.toBe("rgba(0, 0, 0, 0)");
          await expect(
            errorToast.getByText("文章地址已被占用，请换一个地址后重试。", {
              exact: true,
            }),
          ).toBeVisible();

          await page.unroute(postRoute);
          await errorToast.getByRole("button", { name: "关闭通知" }).click();
          await expect(errorToast).toHaveCount(0);
          await page.getByLabel("文章永久链接").fill(originalSlug);
          const saved = page.waitForResponse(
            (response) =>
              response.request().method() === "PATCH" &&
              response.url().endsWith(`/api/v1/posts/${postId}`),
          );
          await page
            .getByRole("button", { name: "保存", exact: true })
            .last()
            .click();
          expect((await saved).ok()).toBeTruthy();
          await expect(
            page.getByRole("button", { name: "已保存", exact: true }).last(),
          ).toBeVisible();
          await expect(
            page
              .locator("[data-yueli-toast]")
              .getByText("已保存", { exact: true }),
          ).toHaveCount(0);
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          if (blockerId) await purgeTestPost(context, site.url, blockerId);
          await context.close();
        }
      });

      test("代码与公式块支持源码预览和重新编辑", async ({
        browser,
      }, testInfo) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        const errors = captureErrors(page);
        let postId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `富文本块验收 ${Date.now()}`,
                content:
                  "```javascript\nconst answer = 1\n```\n\n$$\nE = mc^2\n$$",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          const editorURL = new URL(
            `/manage/posts/${created.post.slug}`,
            site.url,
          ).toString();
          await page.goto(editorURL, { waitUntil: "networkidle" });
          await settleNuxt(page);
          expect(errors).toEqual([]);

          const codeBlock = page.locator("[data-editor-code-block]");
          const mathBlock = page.locator("[data-editor-math-block]");
          await expect(codeBlock).toBeVisible();
          await expect(mathBlock).toBeVisible();
          await expect(
            codeBlock.locator("[data-editor-code-source]"),
          ).toHaveCount(0);
          await expect(
            mathBlock.locator("[data-editor-math-source]"),
          ).toHaveCount(0);
          await codeBlock.getByRole("button", { name: "编辑代码" }).click();
          await page.waitForTimeout(250);
          expect(errors).toEqual([]);
          await expect(
            codeBlock.locator("[data-editor-code-source]"),
          ).toBeVisible();
          await expect(
            codeBlock.locator("[data-editor-code-source]"),
          ).toBeFocused();
          await expect(
            codeBlock.locator("[data-editor-code-preview]"),
          ).toBeVisible();
          await codeBlock.locator("textarea").fill("const answer = 2");
          await expect(
            codeBlock.locator("[data-editor-code-preview]"),
          ).toContainText("const answer = 2");
          await page.screenshot({
            path: testInfo.outputPath("code-block-editing.png"),
            fullPage: false,
          });
          await mathBlock.getByRole("button", { name: "编辑公式" }).click();
          await expect(
            codeBlock.locator("[data-editor-code-source]"),
          ).toHaveCount(0);
          await expect(
            mathBlock.locator("[data-editor-math-source]"),
          ).toBeVisible();
          await expect(
            mathBlock.locator("[data-editor-math-source]"),
          ).toBeFocused();
          await expect(
            mathBlock.locator("[data-editor-math-preview]"),
          ).toBeVisible();
          await page.screenshot({
            path: testInfo.outputPath("rich-block-editing.png"),
            fullPage: true,
          });
          await mathBlock.locator("textarea").fill("E = mc^3");
          await expect(
            mathBlock.locator("[data-editor-math-preview] annotation"),
          ).toHaveText("E = mc^3");
          await page.getByLabel("文章标题").click();
          await expect(
            mathBlock.locator("[data-editor-math-source]"),
          ).toHaveCount(0);

          const editable = page.locator('.tiptap[contenteditable="true"]');
          await editable.click();
          await page.keyboard.press("Control+End");
          await page.keyboard.press("Enter");
          await page.keyboard.type("/");
          const inlineMathItem = page
            .getByText("行内公式", { exact: true })
            .last();
          await expect(inlineMathItem).toBeVisible();
          await inlineMathItem.click();
          await expect(
            editable.locator(
              'span[data-type="inline-math"][data-latex="E = mc^2"]',
            ),
          ).toBeVisible();
          await expect(editable).not.toContainText("$E=mc^2$");
          const inlineSource = editable.locator(
            "[data-editor-inline-math-source]",
          );
          await expect(inlineSource).toBeVisible();
          await expect(inlineSource).toBeFocused();
          await inlineSource.fill("E = mc^4");
          await expect(
            editable.locator("[data-editor-inline-math-preview] annotation"),
          ).toHaveText("E = mc^4");
          await page.getByLabel("文章标题").click();
          await expect(inlineSource).toHaveCount(0);

          const saved = page.waitForResponse(
            (response) =>
              response.request().method() === "PATCH" &&
              response.url().endsWith(`/api/v1/posts/${postId}`),
          );
          await page
            .getByRole("button", { name: "保存", exact: true })
            .last()
            .click();
          expect((await saved).ok()).toBeTruthy();
          await expect(
            page.getByRole("button", { name: "已保存", exact: true }).last(),
          ).toBeVisible();

          await page.reload({ waitUntil: "networkidle" });
          const reopenedCode = page.locator("[data-editor-code-block]");
          const reopenedMath = page.locator("[data-editor-math-block]");
          await expect(
            page.locator(
              '.tiptap span[data-type="inline-math"][data-latex="E = mc^4"]',
            ),
          ).toBeVisible();
          await reopenedCode.locator("[data-editor-code-preview]").click();
          await expect(reopenedCode.locator("textarea")).toHaveValue(
            "const answer = 2",
          );
          await reopenedMath.locator("[data-editor-math-preview]").click();
          await expect(reopenedCode.locator("textarea")).toHaveCount(0);
          await expect(reopenedMath.locator("textarea")).toHaveValue(
            "E = mc^3",
          );
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          await context.close();
        }
      });

      test("Mermaid 创建后立即进入源码与预览状态", async ({
        browser,
      }, testInfo) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        const errors = captureErrors(page);
        let postId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `Mermaid 验收 ${Date.now()}`,
                content: "图表起点",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          await page.goto(
            new URL(`/manage/posts/${created.post.slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );

          const editable = page.locator('.tiptap[contenteditable="true"]');
          await editable.click();
          await page.keyboard.press("Control+End");
          await page.keyboard.press("Enter");
          await page.keyboard.type("/");
          const mermaidItem = page
            .getByText("Mermaid 图表", { exact: true })
            .last();
          await expect(mermaidItem).toBeVisible();
          await mermaidItem.click();

          const source = editable.locator("textarea").last();
          await expect(source).toBeVisible();
          await expect(source).toBeFocused();
          await expect(source).toHaveValue(/graph TD/u);
          await expect(editable.getByText("...", { exact: true })).toHaveCount(
            0,
          );
          await source.fill("graph TD\n  A-->C");
          await editable
            .getByRole("button", { name: "更新预览", exact: true })
            .click();
          await expect(source).toBeVisible();
          await expect(
            editable.locator("[data-editor-mermaid-preview] .nodeLabel").last(),
          ).toHaveText("C");
          await expect(source).toBeVisible();
          await page.screenshot({
            path: testInfo.outputPath("mermaid-editing.png"),
            fullPage: false,
          });
          await page.getByLabel("文章标题").click();
          await expect(source).toHaveCount(0);
          await expect(editable.locator("svg").last()).toBeVisible();
          await expect(
            editable.locator("[data-editor-mermaid-preview] .nodeLabel").last(),
          ).toHaveText("C");
          await page.screenshot({
            path: testInfo.outputPath("mermaid-preview.png"),
            fullPage: false,
          });
          expect(errors).toEqual([]);
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          await context.close();
        }
      });

      test("行内与块公式均可立即编辑并预览", async ({ browser }, testInfo) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let postId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `行内公式验收 ${Date.now()}`,
                content: "质能方程 $E = mc^2$",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          await page.goto(
            new URL(`/manage/posts/${created.post.slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );

          const inlineMath = page.locator(
            '.tiptap span[data-type="inline-math"][data-latex="E = mc^2"]',
          );
          await expect(inlineMath).toBeVisible();
          await inlineMath.click();
          const source = page.locator("[data-editor-inline-math-source]");
          await expect(source).toBeVisible();
          await source.fill("E = mc^4");
          await expect(
            page.locator("[data-editor-inline-math-preview] annotation"),
          ).toHaveText("E = mc^4");
          await page.screenshot({
            path: testInfo.outputPath("inline-math-editing.png"),
            fullPage: false,
          });
          await page.getByLabel("文章标题").click();
          await expect(source).toHaveCount(0);
          await expect(
            page.locator(
              '.tiptap span[data-type="inline-math"][data-latex="E = mc^4"]',
            ),
          ).toBeVisible();

          const editable = page.locator('.tiptap[contenteditable="true"]');
          await editable.click();
          await page.keyboard.press("Control+End");
          await page.keyboard.press("Enter");
          await page.keyboard.type("/");
          const blockMathItem = page
            .getByText("公式块", { exact: true })
            .last();
          await expect(blockMathItem).toBeVisible();
          await blockMathItem.click();
          const blockMath = page.locator("[data-editor-math-block]").last();
          await expect(
            blockMath.locator("[data-editor-math-source]"),
          ).toBeVisible();
          await expect(
            blockMath.locator("[data-editor-math-source]"),
          ).toBeFocused();
          await expect(
            blockMath.locator("[data-editor-math-preview]"),
          ).toBeVisible();
          await page.getByLabel("文章标题").click();
          await expect(
            blockMath.locator("[data-editor-math-source]"),
          ).toHaveCount(0);
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          await context.close();
        }
      });

      test("封面与正文上传遵循 Asset 返回的传输地址", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        const errors = captureErrors(page);
        let postId = "";
        let coverAssetId = "";
        let inlineAssetId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `Asset 地址验收 ${Date.now()}`,
                content: "<p>temporary</p>",
              },
            },
          );
          expect(
            create.ok(),
            `create temporary post failed: ${create.status()}`,
          ).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          const slug = created.post.slug;

          await page.goto(
            new URL(`/manage/posts/${slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );
          await page.getByRole("button", { name: "文章设置" }).click();
          await expect(
            page.locator("[data-blog-settings-accordion]"),
          ).toBeVisible();

          const uploadFlow = Promise.all([
            page.waitForResponse(
              (response) =>
                response.request().method() === "PUT",
            ),
            page.waitForResponse(
              (response) =>
                response.request().method() === "POST" &&
                response
                  .url()
                  .includes(`/api/v1/posts/${postId}/cover/finalize`),
            ),
          ]);
          const coverInput = page.locator(
            '.blog-editor-settings input[type="file"][accept="image/*"]',
          );
          const realUploadFixture = process.env.BLOG_E2E_UPLOAD_FILE?.trim();
          let fixtureBuffer = realUploadFixture
            ? await readFile(realUploadFixture)
            : Buffer.from(imageFixtureBase64, "base64");
          const paddedBytes = Number(
            process.env.BLOG_E2E_UPLOAD_PAD_BYTES || "0",
          );
          if (
            Number.isSafeInteger(paddedBytes) &&
            paddedBytes > fixtureBuffer.length
          ) {
            fixtureBuffer = Buffer.concat([
              fixtureBuffer,
              Buffer.alloc(paddedBytes - fixtureBuffer.length),
            ]);
          }
          await coverInput.setInputFiles({
            name: `cover-probe-${Date.now()}.png`,
            mimeType: "image/png",
            buffer: fixtureBuffer,
          });
          await expect(
            page.locator("[data-asset-image-processor-controls]"),
          ).toBeVisible();
          await page.getByRole("button", { name: "使用处理后的图片" }).click();
          const [upload, finalize] = await uploadFlow;
          expect(upload.status()).toBe(200);
          expect(finalize.status()).toBe(200);
          expect(new URL(upload.url()).protocol).toMatch(/^https?:$/);
          await expect(page.locator('img[alt="文章封面"]')).toBeVisible();

          const detail = await context.request.get(
            new URL(`/api/v1/posts/${slug}`, site.url).toString(),
          );
          expect(detail.ok()).toBeTruthy();
          const detailBody = await detail.json();
          coverAssetId = detailBody.post.coverAssetId || "";
          expect(new URL(detailBody.post.coverUrl).protocol).toMatch(/^https?:$/);

          await page.keyboard.press("Escape");
          const inlineName = `inline-probe-${Date.now()}.png`;
          const inlineProcessedName = inlineName.replace(
            /\.png$/,
            "-crop.webp",
          );
          const inlineFileChooser = page.waitForEvent("filechooser");
          await page
            .locator(".blog-editor-rich-text button:has(.i-tabler\\:photo)")
            .first()
            .click();
          await (
            await inlineFileChooser
          ).setFiles({
            name: inlineName,
            mimeType: "image/png",
            buffer: fixtureBuffer,
          });
          await expect(
            page.getByRole("heading", { name: "处理正文图片" }),
          ).toBeVisible();
          await expect(page.getByText("最终预览", { exact: true })).toHaveCount(
            0,
          );
          const imageToolbar = page.getByRole("toolbar", {
            name: "图片变换工具",
          });
          await expect(imageToolbar).toBeVisible();
          await expect(
            imageToolbar.getByRole("button", { name: "向左旋转" }),
          ).toBeVisible();
          await expect(
            imageToolbar.getByRole("button", { name: "向右旋转" }),
          ).toBeVisible();
          const stageBox = await page
            .locator(".asset-image-cropper-stage")
            .boundingBox();
          const toolbarBox = await imageToolbar.boundingBox();
          expect(toolbarBox?.y || 0).toBeGreaterThanOrEqual(
            (stageBox?.y || 0) + (stageBox?.height || 0),
          );
          const editorHelp = page.getByRole("button", { name: "图片编辑帮助" });
          await expect(editorHelp).toBeVisible();
          await expect(
            page.getByText(
              "拖动图片调整位置，滚轮或工具栏缩放，拖动裁剪框调整范围。",
              { exact: true },
            ),
          ).toHaveCount(0);
          await editorHelp.hover();
          await expect(
            page
              .getByText(
                "拖动图片调整位置；滚轮或工具栏缩放；拖动裁剪框调整范围。",
                { exact: true },
              )
              .first(),
          ).toBeVisible();
          await page.mouse.move(0, 0);
          const zoomSlider = imageToolbar.getByRole("slider", {
            name: "缩放比例",
          });
          await expect(zoomSlider).toBeVisible();
          await expect(zoomSlider).toHaveValue("100");
          await expect(
            imageToolbar.getByText("100%", { exact: true }),
          ).toBeVisible();
          const freeCrop = await page.evaluate(() => {
            const image = document.querySelector(
              ".asset-image-cropper-stage img.cropper-hidden",
            ) as
              | (HTMLImageElement & {
                  cropper?: {
                    getData: (rounded?: boolean) => {
                      x: number;
                      y: number;
                      width: number;
                      height: number;
                    };
                    getImageData: () => {
                      naturalWidth: number;
                      naturalHeight: number;
                    };
                  };
                })
              | null;
            const cropper = image?.cropper;
            if (!cropper) throw new Error("正文图片裁剪器未就绪");
            return {
              crop: cropper.getData(true),
              image: cropper.getImageData(),
            };
          });
          expect(freeCrop.crop.x).toBeCloseTo(0, 0);
          expect(freeCrop.crop.y).toBeCloseTo(0, 0);
          expect(freeCrop.crop.width).toBeCloseTo(
            freeCrop.image.naturalWidth,
            0,
          );
          expect(freeCrop.crop.height).toBeCloseTo(
            freeCrop.image.naturalHeight,
            0,
          );
          const canvasSize = () =>
            page.evaluate(() => {
              const image = document.querySelector(
                ".asset-image-cropper-stage img.cropper-hidden",
              ) as
                | (HTMLImageElement & {
                    cropper?: {
                      getCanvasData: () => { width: number; height: number };
                    };
                  })
                | null;
              const cropper = image?.cropper;
              if (!cropper) throw new Error("正文图片裁剪器未就绪");
              const canvas = cropper.getCanvasData();
              return { width: canvas.width, height: canvas.height };
            });
          const beforeZoom = await canvasSize();
          await imageToolbar.getByRole("button", { name: "放大" }).click();
          const afterZoom = await canvasSize();
          expect(afterZoom.width).toBeGreaterThan(beforeZoom.width);
          expect(afterZoom.height).toBeGreaterThan(beforeZoom.height);
          await expect(zoomSlider).toHaveValue("110");
          await expect(
            imageToolbar.getByText("110%", { exact: true }),
          ).toBeVisible();
          await zoomSlider.fill("220");
          await expect(zoomSlider).toHaveValue("220");
          const zoomedCanvas = await page.evaluate(() => {
            const image = document.querySelector(
              ".asset-image-cropper-stage img.cropper-hidden",
            ) as
              | (HTMLImageElement & {
                  cropper?: {
                    getCanvasData: () => { width: number; height: number };
                  };
                })
              | null;
            const canvas = document.querySelector(
              ".asset-image-cropper-stage .cropper-canvas",
            );
            const stage = document.querySelector(".asset-image-cropper-stage");
            if (!image?.cropper || !canvas || !stage) {
              throw new Error("缩放画布尚未准备好");
            }
            return {
              modelWidth: image.cropper.getCanvasData().width,
              renderedWidth: canvas.getBoundingClientRect().width,
              stageWidth: stage.getBoundingClientRect().width,
            };
          });
          expect(zoomedCanvas.modelWidth).toBeGreaterThan(
            zoomedCanvas.stageWidth,
          );
          expect(zoomedCanvas.renderedWidth).toBeCloseTo(
            zoomedCanvas.modelWidth,
            0,
          );
          await imageToolbar.getByRole("button", { name: "水平翻转" }).click();
          await imageToolbar.getByRole("button", { name: "垂直翻转" }).click();
          const flippedImage = await page.evaluate(() => {
            const image = document.querySelector(
              ".asset-image-cropper-stage img.cropper-hidden",
            ) as
              | (HTMLImageElement & {
                  cropper?: {
                    getImageData: () => { scaleX: number; scaleY: number };
                  };
                })
              | null;
            if (!image?.cropper) throw new Error("翻转画布尚未准备好");
            const data = image.cropper.getImageData();
            return { scaleX: data.scaleX, scaleY: data.scaleY };
          });
          expect(flippedImage).toEqual({ scaleX: -1, scaleY: -1 });
          const processorControls = page.locator(
            "[data-asset-image-processor-controls]",
          );
          const cropperImage = page.locator(
            ".asset-image-cropper-stage img.cropper-hidden",
          );
          await cropperImage.evaluate((image) => {
            image.setAttribute("data-e2e-cropper-instance", "stable");
          });
          await expect(
            processorControls.locator("[data-asset-image-fixed-size]"),
          ).toHaveText("最长边 1200px");
          await expect(
            processorControls.getByRole("heading", { name: "裁剪" }),
          ).toBeVisible();
          await expect(
            processorControls.getByRole("heading", { name: "导出" }),
          ).toBeVisible();
          await expect(
            processorControls.getByRole("combobox", { name: "尺寸" }),
          ).toHaveCount(0);
          await expect(
            processorControls.getByRole("combobox", { name: "格式" }),
          ).toHaveCount(0);
          await expect(
            processorControls.locator("[data-asset-image-fixed-format]"),
          ).toContainText("WebP");
          await expect(cropperImage).toHaveAttribute(
            "data-e2e-cropper-instance",
            "stable",
          );
          await expect(
            processorControls.getByText(/^\d+ × \d+$/),
          ).toBeVisible();
          const formatHelp = page.getByRole("button", { name: "输出格式说明" });
          await expect(formatHelp).toBeVisible();
          await expect(
            page.getByText(/WebP 通常体积更小.*资源中心限制校验/),
          ).toHaveCount(0);
          await expect(
            processorControls.getByRole("button", { name: "文件信息" }),
          ).toHaveCount(0);
          await expect(
            processorControls.getByText(inlineName, { exact: true }),
          ).toBeVisible();
          await expect(
            processorControls.getByText(/^160 × 90 ·/),
          ).toBeVisible();
          const inlineFlow = Promise.all([
            page.waitForResponse(
              (response) =>
                response.request().method() === "PUT",
            ),
            page.waitForResponse(
              (response) =>
                response.request().method() === "POST" &&
                response.url().includes("/api/v1/images/finalize"),
            ),
          ]);
          await page.getByRole("button", { name: "使用处理后的图片" }).click();
          const [inlineUpload, inlineFinalize] = await inlineFlow;
          expect(inlineUpload.status()).toBe(200);
          expect(new URL(inlineUpload.url()).protocol).toMatch(/^https?:$/);
          expect(inlineFinalize.status()).toBe(200);
          expect((await inlineFinalize.json()).url).toMatch(
            /^\/media\/[0-9A-Za-z_-]+\?format=webp&name=inline$/,
          );
          await expect(
            page.locator(`.blog-editor-rich-text img[alt="${inlineName}"]`),
          ).toBeVisible();

          const assets = await context.request.get(
            new URL(
              "/asset-api/api/v1/assets?siteKey=blog-main&profileKey=blog-post&page=1&size=100",
              site.url,
            ).toString(),
          );
          expect(assets.ok()).toBeTruthy();
          const assetItems = (await assets.json()).items || [];
          inlineAssetId =
            assetItems.find(
              (asset: { filename?: string }) =>
                asset.filename === inlineProcessedName,
            )?.id || "";
          expect(inlineAssetId).not.toBe("");
          expect(errors).toEqual([]);
        } finally {
          if (coverAssetId && postId) {
            const referenceURL = new URL(
              "/asset-api/api/v1/asset-references",
              site.url,
            );
            referenceURL.searchParams.set("assetId", coverAssetId);
            referenceURL.searchParams.set("siteKey", "blog-main");
            referenceURL.searchParams.set("refType", "post-cover");
            referenceURL.searchParams.set("refId", postId);
            await context.request.delete(referenceURL.toString());
          }
          if (postId) {
            await purgeTestPost(context, site.url, postId);
          }
          if (coverAssetId) {
            await context.request.delete(
              new URL(
                `/asset-api/api/v1/assets/${coverAssetId}`,
                site.url,
              ).toString(),
            );
          }
          if (inlineAssetId) {
            await context.request.delete(
              new URL(
                `/asset-api/api/v1/assets/${inlineAssetId}`,
                site.url,
              ).toString(),
            );
          }
          await context.close();
        }
      });

      test("站点封面比例驱动处理器并允许临时自定义", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let originalConfig: Record<string, unknown> | null = null;
        let postId = "";
        try {
          await page.goto(new URL("/manage", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const home = await context.request.get(
            new URL("/api/v1/home", site.url).toString(),
          );
          expect(home.ok()).toBeTruthy();
          originalConfig = (await home.json()).config;

          await page.goto(
            new URL("/manage/settings?section=site", site.url).toString(),
            { waitUntil: "networkidle" },
          );
          const settingsRatio = page.getByRole("combobox").first();
          await settingsRatio.click();
          await page.getByRole("option", { name: "自定义" }).click();
          await page
            .getByRole("spinbutton", { name: "封面比例宽度" })
            .fill("5");
          await page
            .getByRole("spinbutton", { name: "封面比例高度" })
            .fill("4");
          const settingsUpdate = page.waitForResponse(
            (response) =>
              response.request().method() === "PATCH" &&
              response.url().endsWith("/api/v1/home"),
          );
          await page
            .locator("[data-settings-header-actions]")
            .getByRole("button", { name: "保存", exact: true })
            .click();
          expect((await settingsUpdate).ok()).toBeTruthy();
          await expect(
            page.getByRole("button", { name: "已保存", exact: true }),
          ).toBeVisible();

          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `封面比例验收 ${Date.now()}`,
                content: "<p>temporary</p>",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          await page.goto(
            new URL(`/manage/posts/${created.post.slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );
          await page.getByRole("button", { name: "文章设置" }).click();
          await page
            .locator(
              '.blog-editor-settings input[type="file"][accept="image/*"]',
            )
            .setInputFiles({
              name: "ratio-probe.png",
              mimeType: "image/png",
              buffer: Buffer.from(imageFixtureBase64, "base64"),
            });
          const controls = page.locator(
            "[data-asset-image-processor-controls]",
          );
          await expect(controls).toBeVisible();
          const processorRatio = controls.getByRole("combobox").first();
          await expect(processorRatio).toContainText("站点默认 · 5:4");
          await processorRatio.click();
          await expect(
            page.getByRole("option", { name: "横向 3:2" }),
          ).toBeVisible();
          await page.getByRole("option", { name: "自定义比例" }).click();
          await expect(
            page.locator("[data-asset-image-custom-ratio]"),
          ).toBeVisible();
          await page.getByRole("spinbutton", { name: "宽度" }).fill("7");
          await page.getByRole("spinbutton", { name: "高度" }).fill("5");
          await expect(
            page.locator("[data-asset-image-cropper]"),
          ).toHaveAttribute("data-can-confirm", "true");
          await page.getByRole("button", { name: "取消" }).last().click();
        } finally {
          if (postId) await purgeTestPost(context, site.url, postId);
          if (originalConfig) {
            await context.request.patch(
              new URL("/api/v1/home", site.url).toString(),
              { data: originalConfig },
            );
          }
          await context.close();
        }
      });

      test("图片处理器在资源上限前压缩大 PNG", async ({ browser }) => {
        const context = await loginE2E(browser, {
          viewport: { width: 1440, height: 900 },
        });
        const page = await context.newPage();
        let postId = "";
        let coverAssetId = "";
        try {
          await page.goto(new URL("/manage/posts", site.url).toString(), {
            waitUntil: "networkidle",
          });
          const create = await context.request.post(
            new URL("/api/v1/posts", site.url).toString(),
            {
              data: {
                title: `真实图片上限验收 ${Date.now()}`,
                content: "<p>temporary</p>",
              },
            },
          );
          expect(create.ok()).toBeTruthy();
          const created = await create.json();
          postId = created.post.id;
          await page.goto(
            new URL(`/manage/posts/${created.post.slug}`, site.url).toString(),
            { waitUntil: "networkidle" },
          );
          await page.getByRole("button", { name: "文章设置" }).click();

          const source = Buffer.from(imageFixtureBase64, "base64");
          await page
            .locator(
              '.blog-editor-settings input[type="file"][accept="image/*"]',
            )
            .setInputFiles({
              name: "real-limit.png",
              mimeType: "image/png",
              buffer: Buffer.concat([
                source,
                Buffer.alloc(11 * 1024 * 1024 - source.length),
              ]),
            });
          await expect(
            page.locator("[data-asset-image-processor-controls]"),
          ).toBeVisible();
          await expect(page.getByText(/11 MB/)).toBeVisible();

          const uploadFlow = Promise.all([
            page.waitForResponse(
              (response) =>
                response.request().method() === "PUT",
            ),
            page.waitForResponse(
              (response) =>
                response.request().method() === "POST" &&
                response
                  .url()
                  .includes(`/api/v1/posts/${postId}/cover/finalize`),
            ),
          ]);
          await page.getByRole("button", { name: "使用处理后的图片" }).click();
          const [upload, finalize] = await uploadFlow;
          expect(upload.status()).toBe(200);
          expect(finalize.status()).toBe(200);
          await expect(page.locator('img[alt="文章封面"]')).toBeVisible();
          await expect(
            page.locator("[data-blog-image-compression-dialog]"),
          ).toHaveCount(0);

          const detail = await context.request.get(
            new URL(`/api/v1/posts/${created.post.slug}`, site.url).toString(),
          );
          expect(detail.ok()).toBeTruthy();
          coverAssetId = (await detail.json()).post.coverAssetId || "";
          expect(coverAssetId).not.toBe("");
          const assets = await context.request.get(
            new URL(
              "/asset-api/api/v1/assets?siteKey=blog-main&profileKey=blog-cover&page=1&size=100",
              site.url,
            ).toString(),
          );
          expect(assets.ok()).toBeTruthy();
          const processed = ((await assets.json()).items || []).find(
            (asset: { id?: string }) => asset.id === coverAssetId,
          );
          expect(processed?.mime).toBe("image/webp");
          expect(processed?.filename).toMatch(/\.webp$/);
        } finally {
          if (coverAssetId && postId) {
            const referenceURL = new URL(
              "/asset-api/api/v1/asset-references",
              site.url,
            );
            referenceURL.searchParams.set("assetId", coverAssetId);
            referenceURL.searchParams.set("siteKey", "blog-main");
            referenceURL.searchParams.set("refType", "post-cover");
            referenceURL.searchParams.set("refId", postId);
            await context.request.delete(referenceURL.toString());
          }
          if (postId) {
            await purgeTestPost(context, site.url, postId);
          }
          if (coverAssetId) {
            await context.request.delete(
              new URL(
                `/asset-api/api/v1/assets/${coverAssetId}`,
                site.url,
              ).toString(),
            );
          }
          await context.close();
        }
      });

      test("设置页脏数据保护阻止意外离开", async ({ browser }) => {
        const settings = contract.manage.settings;
        test.skip(!settings, "该产品没有可编辑的通用设置页");
        if (!settings) return;
        const context = await loginE2E(browser);
        const page = await context.newPage();
        const errors = captureErrors(page);
        try {
          await page.goto(new URL(settings.path, site.url).toString(), {
            waitUntil: "domcontentloaded",
          });
          await settleNuxt(page);
          const field = page
            .getByRole("textbox", { name: settings.fieldLabel, exact: false })
            .first();
          await expect(field).toBeVisible();
          await expect(page.locator('[data-manage-dock="save"]')).toHaveCount(
            0,
          );
          await field.fill(`${await field.inputValue()} · 未保存`);
          await expect(page.locator('[data-manage-dock="save"]')).toBeVisible();

          const dialogHandled = new Promise<void>((resolve) => {
            page.once("dialog", async (dialog) => {
              expect(dialog.type()).toBe("confirm");
              expect(dialog.message()).toContain("未保存");
              await dialog.dismiss();
              resolve();
            });
          });
          await page
            .locator(`a[href="${contract.manage.path}"]`)
            .first()
            .click();
          await dialogHandled;
          await expect(page).toHaveURL(
            new RegExp(
              `${settings.path.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}(?:\\?|$)`,
            ),
          );
          expect(errors).toEqual([]);
        } finally {
          await context.close();
        }
      });

      test("空结果状态有明确反馈", async ({ browser, page }) => {
        const context = contract.empty.authenticated
          ? await loginE2E(browser)
          : undefined;
        const targetPage = context ? await context.newPage() : page;
        const errors = captureErrors(targetPage);
        try {
          await targetPage.goto(
            new URL(contract.empty.path, site.url).toString(),
            { waitUntil: "domcontentloaded" },
          );
          const emptyState = targetPage
            .getByText(contract.empty.text, { exact: true })
            .first();
          if (contract.empty.inputPlaceholder) {
            const input = targetPage.getByPlaceholder(
              contract.empty.inputPlaceholder,
            );
            await expect(async () => {
              await input.fill(missing);
              await expect(input).toHaveValue(missing);
              await expect(emptyState).toBeVisible({ timeout: 2_000 });
            }).toPass({ timeout: 15_000, intervals: [250, 500, 1_000] });
          }
          await expect(emptyState).toBeVisible();
          await settleNuxt(targetPage);
          expect(errors).toEqual([]);
        } finally {
          await context?.close();
        }
      });

      test("缺失实体返回产品错误状态", async ({ page }) => {
        const errors = captureErrors(page);
        const response = await page.goto(
          new URL(contract.error.path, site.url).toString(),
          { waitUntil: "domcontentloaded" },
        );
        expect(response?.status()).toBe(contract.error.status);
        await expect(
          page.getByText(contract.error.text, { exact: false }).first(),
        ).toBeVisible();
        await settleNuxt(page);
        expect(
          errors.filter(
            (error) => !error.includes(`status of ${contract.error.status}`),
          ),
        ).toEqual([]);
      });
    });
  }
}
